package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshTokenRepository struct {
	DB *pgxpool.Pool
}

func NewRefreshTokenRepository(
	db *pgxpool.Pool,
) *RefreshTokenRepository {
	return &RefreshTokenRepository{
		DB: db,
	}
}

type ActiveRefreshToken struct {
	UserID   string
	DeviceID *string
}

type RefreshTokenRecord struct {
	UserID    string
	DeviceID  *string
	RevokedAt *time.Time
}

// Create บันทึก Refresh Token Hash ลง Database
func (r *RefreshTokenRepository) Create(
	ctx context.Context,
	userID string,
	tokenHash string,
	expiresAt time.Time,
	deviceID *string,
) error {

	query := `
		INSERT INTO refresh_tokens (
			user_id,
			token_hash,
			expires_at,
			device_id
		)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.DB.Exec(
		ctx,
		query,
		userID,
		tokenHash,
		expiresAt,
		deviceID,
	)

	return err
}

func (r *RefreshTokenRepository) Revoke(
	ctx context.Context,
	tokenHash string,
) error {
	query := `
        UPDATE refresh_tokens
        SET revoked_at = NOW()
        WHERE token_hash = $1
        AND revoked_at IS NULL
    `

	_, err := r.DB.Exec(ctx, query, tokenHash)
	return err
}

func (r *RefreshTokenRepository) RevokeAllByUserID(
	ctx context.Context,
	userID string,
) error {
	_, err := r.DB.Exec(
		ctx,
		`
			UPDATE refresh_tokens
			SET revoked_at = NOW()
			WHERE user_id = $1
			  AND revoked_at IS NULL
		`,
		userID,
	)
	return err
}

func (r *RefreshTokenRepository) FindActive(
	ctx context.Context,
	tokenHash string,
) (*ActiveRefreshToken, error) {
	query := `
		SELECT user_id::text, device_id::text
		FROM refresh_tokens
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
	`

	var active ActiveRefreshToken
	err := r.DB.QueryRow(ctx, query, tokenHash).Scan(
		&active.UserID,
		&active.DeviceID,
	)
	if err != nil {
		return nil, err
	}

	return &active, nil
}

func (r *RefreshTokenRepository) FindByHash(
	ctx context.Context,
	tokenHash string,
) (*RefreshTokenRecord, error) {
	query := `
		SELECT user_id::text, device_id::text, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	var record RefreshTokenRecord
	err := r.DB.QueryRow(ctx, query, tokenHash).Scan(
		&record.UserID,
		&record.DeviceID,
		&record.RevokedAt,
	)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (r *RefreshTokenRepository) RevokeByDeviceID(
	ctx context.Context,
	userID string,
	deviceID string,
) error {
	_, err := r.DB.Exec(
		ctx,
		`
			UPDATE refresh_tokens
			SET revoked_at = NOW()
			WHERE user_id = $1
			  AND device_id = $2
			  AND revoked_at IS NULL
		`,
		userID,
		deviceID,
	)
	return err
}
