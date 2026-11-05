package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"time"

	"go-be/internal/models"
	"go-be/internal/repository"
	"go-be/internal/utils"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidInput   = errors.New("invalid input")
	ErrEmailExists    = errors.New("email already exists")
	ErrUsernameExists = errors.New("username already exists")
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrJWTSecretMissing   = errors.New("JWT secret is missing")
	ErrUserNotFound       = errors.New("user not found")
)

type AuthService struct {
	UserRepo         *repository.UserRepository
	RefreshTokenRepo *repository.RefreshTokenRepository
	DeviceRepo       *repository.DeviceRepository
}

type AccessTokenClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func NewAuthService(
	userRepo *repository.UserRepository,
	refreshTokenRepo *repository.RefreshTokenRepository,
	deviceRepo *repository.DeviceRepository,
) *AuthService {
	return &AuthService{
		UserRepo:         userRepo,
		RefreshTokenRepo: refreshTokenRepo,
		DeviceRepo:       deviceRepo,
	}
}

type RegisterInput struct {
	Username string
	Email    string
	Password string
}

type DeviceInput struct {
	DeviceKey string
	Name      string
	UserAgent string
	IPAddress string
}

func (s *AuthService) Register(
	ctx context.Context,
	input *RegisterInput,
) (*models.User, error) {

	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.TrimSpace(input.Email)

	if len(input.Username) < 3 || len(input.Username) > 50 {
		return nil, ErrInvalidInput
	}

	if len(input.Email) == 0 || len(input.Email) > 255 || !strings.Contains(input.Email, "@") {
		return nil, ErrInvalidInput
	}

	if len(input.Password) < 8 || len(input.Password) > 72 {
		return nil, ErrInvalidInput
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username: input.Username,
		Email:    input.Email,
		Password: string(hashedPassword),
		Role:     "user",
	}

	err = s.UserRepo.Create(ctx, user)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "users_email_key":
				return nil, ErrEmailExists
			case "users_username_key":
				return nil, ErrUsernameExists
			}
		}

		return nil, err
	}

	return user, nil
}

func (s *AuthService) Login(
	ctx context.Context,
	email string,
	password string,
	device *DeviceInput,
) (string, string, *models.User, error) {

	email = strings.TrimSpace(email)

	user, err := s.UserRepo.FindByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil, ErrInvalidCredentials
	}
	if err != nil {
		return "", "", nil, err
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(password),
	)
	if err != nil {
		return "", "", nil, ErrInvalidCredentials
	}

	accessToken, err := s.issueAccessToken(user)
	if err != nil {
		return "", "", nil, err
	}

	savedDevice, err := s.upsertDevice(ctx, user.ID.String(), device)
	if err != nil {
		return "", "", nil, err
	}

	var deviceID *string
	if savedDevice != nil {
		deviceID = &savedDevice.ID
	}

	refreshToken, err := s.issueRefreshToken(ctx, user.ID.String(), deviceID)
	if err != nil {
		return "", "", nil, err
	}

	return accessToken, refreshToken, user, nil
}

func (s *AuthService) GetProfile(
	ctx context.Context,
	userId string,
) (*models.User, error) {
	return s.UserRepo.FindByID(ctx, userId)
}

func (s *AuthService) ListDevices(
	ctx context.Context,
	userID string,
) ([]models.Device, error) {
	return s.DeviceRepo.ListByUserID(ctx, userID)
}

func (s *AuthService) Logout(
	ctx context.Context,
	refreshToken string,
) error {

	tokenHash := utils.HashToken(refreshToken)

	return s.RefreshTokenRepo.Revoke(ctx, tokenHash)
}

var ErrInvalidRefreshToken = errors.New("invalid refresh token")

func (s *AuthService) Refresh(
	ctx context.Context,
	refreshToken string,
	device *DeviceInput,
) (string, string, error) {
	active, err := s.RefreshTokenRepo.FindActive(
		ctx,
		utils.HashToken(refreshToken),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrInvalidRefreshToken
		}
		return "", "", err
	}

	user, err := s.UserRepo.FindByID(ctx, active.UserID)
	if err != nil {
		return "", "", err
	}

	accessToken, err := s.issueAccessToken(user)
	if err != nil {
		return "", "", err
	}

	deviceID := active.DeviceID
	switch {
	case device != nil && strings.TrimSpace(device.DeviceKey) != "":
		savedDevice, err := s.upsertDevice(ctx, user.ID.String(), device)
		if err != nil {
			return "", "", err
		}
		if savedDevice != nil {
			deviceID = &savedDevice.ID
		}
	case deviceID != nil:
		_ = s.DeviceRepo.Touch(ctx, *deviceID, deviceUserAgent(device), deviceIPAddress(device))
	default:
		savedDevice, err := s.upsertDevice(ctx, user.ID.String(), device)
		if err != nil {
			return "", "", err
		}
		if savedDevice != nil {
			deviceID = &savedDevice.ID
		}
	}

	newRefreshToken, err := s.issueRefreshToken(ctx, user.ID.String(), deviceID)
	if err != nil {
		return "", "", err
	}

	if err := s.Logout(ctx, refreshToken); err != nil {
		return "", "", err
	}

	return accessToken, newRefreshToken, nil
}

func (s *AuthService) issueAccessToken(user *models.User) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", ErrJWTSecretMissing
	}

	now := time.Now()
	claims := AccessTokenClaims{
		Role: user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (s *AuthService) issueRefreshToken(
	ctx context.Context,
	userID string,
	deviceID *string,
) (string, error) {
	raw, err := GenerateRefreshToken()
	if err != nil {
		return "", err
	}

	err = s.RefreshTokenRepo.Create(
		ctx,
		userID,
		utils.HashToken(raw),
		time.Now().Add(7*24*time.Hour),
		deviceID,
	)
	if err != nil {
		return "", err
	}

	return raw, nil
}

func (s *AuthService) upsertDevice(
	ctx context.Context,
	userID string,
	device *DeviceInput,
) (*models.Device, error) {
	if device == nil {
		device = &DeviceInput{}
	}

	deviceKey := strings.TrimSpace(device.DeviceKey)
	if deviceKey == "" {
		generated, err := generateDeviceKey()
		if err != nil {
			return nil, err
		}
		deviceKey = generated
	}

	if len(deviceKey) > 128 {
		return nil, ErrInvalidInput
	}

	name := strings.TrimSpace(device.Name)
	if len(name) > 100 {
		name = name[:100]
	}

	return s.DeviceRepo.Upsert(ctx, repository.DeviceUpsertInput{
		UserID:    userID,
		DeviceKey: deviceKey,
		Name:      name,
		UserAgent: strings.TrimSpace(device.UserAgent),
		IPAddress: strings.TrimSpace(device.IPAddress),
	})
}

func generateDeviceKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func deviceUserAgent(device *DeviceInput) string {
	if device == nil {
		return ""
	}
	return device.UserAgent
}

func deviceIPAddress(device *DeviceInput) string {
	if device == nil {
		return ""
	}
	return device.IPAddress
}
