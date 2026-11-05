package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"go-be/internal/i18n"
	"go-be/internal/response"
	"go-be/internal/service"
)

type AuthHandler struct {
	AuthService *service.AuthService
}

func NewAuthHandler(
	authService *service.AuthService,
) *AuthHandler {
	return &AuthHandler{
		AuthService: authService,
	}
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type DeviceRequest struct {
	DeviceID string `json:"deviceId" binding:"omitempty,max=128"`
	Name     string `json:"name" binding:"omitempty,max=100"`
}

type LoginRequest struct {
	Email    string         `json:"email" binding:"required,email,max=255"`
	Password string         `json:"password" binding:"required,min=8,max=72"`
	Device   *DeviceRequest `json:"device"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string         `json:"refreshToken" binding:"required"`
	Device       *DeviceRequest `json:"device"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email,max=255"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token" binding:"required,min=16,max=128"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type DiscordLoginRequest struct {
	Code        string         `json:"code" binding:"required,min=1,max=512"`
	RedirectURI string         `json:"redirectUri" binding:"required,url,max=512"`
	Device      *DeviceRequest `json:"device"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "auth.invalid_request")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := h.AuthService.Register(ctx, &service.RegisterInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})

	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Invalid input",
			})

		case errors.Is(err, service.ErrEmailExists):
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": i18n.T(c, "auth.email_exists"),
			})

		case errors.Is(err, service.ErrUsernameExists):
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Username already exists",
			})

		default:
			log.Printf("register failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Internal server error",
			})
		}

		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "User registered successfully",
		"user":    user,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	accessToken, refreshToken, user, err := h.AuthService.Login(
		ctx,
		req.Email,
		req.Password,
		toDeviceInput(c, req.Device),
	)

	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": i18n.T(c, "auth.invalid_credentials"),
			})
			return

		case errors.Is(err, service.ErrAccountLocked):
			c.JSON(http.StatusForbidden, gin.H{
				"status":  "error",
				"message": i18n.T(c, "auth.account_locked"),
			})
			return

		case errors.Is(err, service.ErrJWTSecretMissing):
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": i18n.T(c, "common.internal_error"),
			})
			return

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": i18n.T(c, "common.internal_error"),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Login successful",
		"data": gin.H{
			"accessToken":  accessToken,
			"refreshToken": refreshToken,
			"tokenType":    "Bearer",
			"expiresIn":    900, // 15 minutes
			"user":         user,
		},
	})
}

func (h *AuthHandler) DiscordLogin(c *gin.Context) {
	var req DiscordLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "auth.invalid_request")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	accessToken, refreshToken, user, err := h.AuthService.LoginWithDiscord(
		ctx,
		req.Code,
		req.RedirectURI,
		toDeviceInput(c, req.Device),
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDiscordNotConfigured):
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "error",
				"message": i18n.T(c, "auth.discord_not_configured"),
			})
		case errors.Is(err, service.ErrDiscordEmailMissing):
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": i18n.T(c, "auth.discord_email_missing"),
			})
		case errors.Is(err, service.ErrDiscordAuthFailed):
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": i18n.T(c, "auth.discord_failed"),
			})
		case errors.Is(err, service.ErrEmailExists):
			c.JSON(http.StatusConflict, gin.H{
				"status":  "error",
				"message": i18n.T(c, "auth.discord_email_conflict"),
			})
		case errors.Is(err, service.ErrJWTSecretMissing):
			response.InternalServerError(c, "common.internal_error")
		default:
			log.Printf("discord login failed: %v", err)
			response.InternalServerError(c, "common.internal_error")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": i18n.T(c, "auth.login_success"),
		"data": gin.H{
			"accessToken":  accessToken,
			"refreshToken": refreshToken,
			"tokenType":    "Bearer",
			"expiresIn":    900,
			"user":         user,
		},
	})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {

	userId := c.GetString("userID")

	if userId == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized",
		})
		return
	}

	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		5*time.Second,
	)
	defer cancel()

	user, err := h.AuthService.GetProfile(ctx, userId)

	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Unauthorized",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   user,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {

	var req LogoutRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request data")
		return
	}

	err := h.AuthService.Logout(
		c.Request.Context(),
		req.RefreshToken,
	)

	if err != nil {
		response.InternalServerError(c, "Internal server error")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logout successful",
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request data")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	accessToken, refreshToken, err := h.AuthService.Refresh(
		ctx,
		req.RefreshToken,
		toDeviceInput(c, req.Device),
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRefreshToken) {
			response.Unauthorized(c, "Invalid refresh token")
			return
		}
		response.InternalServerError(c, "Internal server error")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"accessToken":  accessToken,
			"refreshToken": refreshToken,
			"tokenType":    "Bearer",
			"expiresIn":    900,
		},
	})
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "auth.invalid_request")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	locale, _ := c.Get("locale")
	localeStr, _ := locale.(string)

	err := h.AuthService.RequestPasswordReset(ctx, req.Email, localeStr)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			response.BadRequest(c, "auth.invalid_input")
			return
		}
		log.Printf("forgot password failed: %v", err)
		response.InternalServerError(c, "common.internal_error")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": i18n.T(c, "auth.password_reset_sent"),
	})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "auth.invalid_request")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	err := h.AuthService.ResetPassword(ctx, req.Token, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			response.BadRequest(c, "auth.invalid_input")
		case errors.Is(err, service.ErrInvalidResetToken):
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": i18n.T(c, "auth.invalid_reset_token"),
			})
		default:
			log.Printf("reset password failed: %v", err)
			response.InternalServerError(c, "common.internal_error")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": i18n.T(c, "auth.password_reset_success"),
	})
}

func (h *AuthHandler) ListDevices(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	devices, err := h.AuthService.ListDevices(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   devices,
	})
}

func toDeviceInput(c *gin.Context, device *DeviceRequest) *service.DeviceInput {
	input := &service.DeviceInput{
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	}

	if device != nil {
		input.DeviceKey = device.DeviceID
		input.Name = device.Name
	}

	return input
}
