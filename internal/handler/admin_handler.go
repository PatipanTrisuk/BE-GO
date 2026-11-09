package handler

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"go-be/internal/i18n"
	"go-be/internal/middleware"
	"go-be/internal/repository"
	"go-be/internal/service"
)

const (
	defaultPage  = 1
	defaultLimit = 10
	maxLimit     = 100
)

type AdminHandler struct {
	AuthService   *service.AuthService
	UserRepo      *repository.UserRepository
	ExportService *service.ExportService
	DB            *pgxpool.Pool
}

func NewAdminHandler(
	authService *service.AuthService,
	userRepo *repository.UserRepository,
	exportService *service.ExportService,
	db *pgxpool.Pool,
) *AdminHandler {
	return &AdminHandler{
		AuthService:   authService,
		UserRepo:      userRepo,
		ExportService: exportService,
		DB:            db,
	}
}

func parsePositiveInt(raw string, fallback, max int) int {
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return fallback
	}
	if max > 0 && n > max {
		return max
	}
	return n
}

func actorID(c *gin.Context) string {
	v, _ := c.Get("userID")
	s, _ := v.(string)
	return s
}

func localeFromContext(c *gin.Context) string {
	if v, ok := c.Get(middleware.LocaleKey); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return "en"
}

func (h *AdminHandler) writeAdminError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": i18n.T(c, "admin.user_not_found"),
			"code":    "admin.user_not_found",
		})
	case errors.Is(err, service.ErrCannotTargetSelf):
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": i18n.T(c, "admin.cannot_target_self"),
			"code":    "admin.cannot_target_self",
		})
	case errors.Is(err, service.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": i18n.T(c, "auth.invalid_input"),
			"code":    "auth.invalid_input",
		})
	case errors.Is(err, service.ErrEmailVerified):
		c.JSON(http.StatusConflict, gin.H{
			"status":  "error",
			"message": i18n.T(c, "admin.email_already_verified"),
			"code":    "admin.email_already_verified",
		})
	case errors.Is(err, service.ErrUserAlreadyDeleted):
		c.JSON(http.StatusConflict, gin.H{
			"status":  "error",
			"message": i18n.T(c, "admin.user_already_deleted"),
			"code":    "admin.user_already_deleted",
		})
	case errors.Is(err, service.ErrUserNotDeleted):
		c.JSON(http.StatusConflict, gin.H{
			"status":  "error",
			"message": i18n.T(c, "admin.user_not_deleted"),
			"code":    "admin.user_not_deleted",
		})
	case errors.Is(err, service.ErrExportJobNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": i18n.T(c, "admin.export_not_found"),
			"code":    "admin.export_not_found",
		})
	case errors.Is(err, service.ErrExportForbidden):
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": i18n.T(c, "auth.forbidden"),
			"code":    "auth.forbidden",
		})
	case errors.Is(err, service.ErrExportNotReady):
		c.JSON(http.StatusConflict, gin.H{
			"status":  "error",
			"message": i18n.T(c, "admin.export_not_ready"),
			"code":    "admin.export_not_ready",
		})
	case errors.Is(err, service.ErrEmailExists):
		c.JSON(http.StatusConflict, gin.H{
			"status":  "error",
			"message": i18n.T(c, "auth.email_exists"),
			"code":    "auth.email_exists",
		})
	case errors.Is(err, service.ErrUsernameExists):
		c.JSON(http.StatusConflict, gin.H{
			"status":  "error",
			"message": i18n.T(c, "auth.username_exists"),
			"code":    "auth.username_exists",
		})
	default:
		log.Printf("admin action failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": i18n.T(c, "common.internal_error"),
		})
	}
}

func parseDateQuery(raw string, endOfDay bool) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	t, err := time.ParseInLocation("2006-01-02", raw, time.Local)
	if err != nil {
		return nil
	}
	if endOfDay {
		t = t.Add(24*time.Hour - time.Nanosecond)
	}
	return &t
}

func parseAdminListParams(c *gin.Context, page, limit int) repository.ListParams {
	return repository.ListParams{
		Page:          page,
		Limit:         limit,
		Query:         strings.TrimSpace(c.Query("search")),
		Role:          strings.TrimSpace(c.Query("role")),
		AuthProvider:  strings.TrimSpace(c.Query("auth_provider")),
		DeletedStatus: strings.TrimSpace(c.Query("deleted_status")),
		CreatedFrom:   parseDateQuery(c.Query("created_from"), false),
		CreatedTo:     parseDateQuery(c.Query("created_to"), true),
		UpdatedFrom:   parseDateQuery(c.Query("updated_from"), false),
		UpdatedTo:     parseDateQuery(c.Query("updated_to"), true),
		SortBy:        strings.TrimSpace(c.Query("sort_by")),
		SortDir:       strings.TrimSpace(c.Query("sort_dir")),
	}
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	page := parsePositiveInt(c.Query("page"), defaultPage, 0)
	limit := parsePositiveInt(c.Query("limit"), defaultLimit, maxLimit)
	params := parseAdminListParams(c, page, limit)

	users, err := h.UserRepo.List(ctx, params)
	if err != nil {
		log.Printf("admin list users failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": i18n.T(c, "common.internal_error"),
		})
		return
	}

	total, err := h.UserRepo.Count(ctx, params)
	if err != nil {
		log.Printf("admin list users count failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": i18n.T(c, "common.internal_error"),
		})
		return
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   users,
		"meta": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": totalPages,
		},
	})
}

func (h *AdminHandler) Stats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	userCount, err := h.UserRepo.Count(ctx, repository.ListParams{})
	if err != nil {
		log.Printf("admin stats failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": i18n.T(c, "common.internal_error"),
		})
		return
	}

	dbOK := true
	if err := h.DB.Ping(ctx); err != nil {
		dbOK = false
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"userCount": userCount,
			"apiStatus": "ok",
			"dbStatus":  map[bool]string{true: "ok", false: "error"}[dbOK],
		},
	})
}

type adminSetPasswordRequest struct {
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type adminCreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email,max=255"`
	RoleID   string `json:"role_id" binding:"omitempty,uuid"`
}

type adminUpdateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email,max=255"`
	RoleID   string `json:"role_id" binding:"required,uuid"`
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req adminCreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": i18n.T(c, "auth.invalid_request"),
			"code":    "auth.invalid_request",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	user, err := h.AuthService.AdminCreateUser(ctx, service.AdminUserInput{
		Username: req.Username,
		Email:    req.Email,
		RoleID:   req.RoleID, // empty → service defaults to "user" role
	})
	if err != nil {
		h.writeAdminError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": i18n.T(c, "admin.user_created"),
		"data":    user,
	})
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	var req adminUpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": i18n.T(c, "auth.invalid_request"),
			"code":    "auth.invalid_request",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	user, err := h.AuthService.AdminUpdateUser(ctx, actorID(c), c.Param("id"), service.AdminUserInput{
		Username: req.Username,
		Email:    req.Email,
		RoleID:   req.RoleID,
	})
	if err != nil {
		h.writeAdminError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": i18n.T(c, "admin.user_updated"),
		"data":    user,
	})
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.AuthService.AdminDeleteUser(ctx, actorID(c), c.Param("id")); err != nil {
		h.writeAdminError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": i18n.T(c, "admin.user_deleted"),
	})
}

func (h *AdminHandler) RestoreUser(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	user, err := h.AuthService.AdminRestoreUser(ctx, actorID(c), c.Param("id"))
	if err != nil {
		h.writeAdminError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": i18n.T(c, "admin.user_restored"),
		"data":    user,
	})
}

type exportUsersRequest struct {
	Search        string `json:"search"`
	Role          string `json:"role"`
	AuthProvider  string `json:"auth_provider"`
	DeletedStatus string `json:"deleted_status"`
	CreatedFrom   string `json:"created_from"`
	CreatedTo     string `json:"created_to"`
	UpdatedFrom   string `json:"updated_from"`
	UpdatedTo     string `json:"updated_to"`
	SortBy        string `json:"sort_by"`
	SortDir       string `json:"sort_dir"`
}

func (h *AdminHandler) ExportUsers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req exportUsersRequest
	_ = c.ShouldBindJSON(&req)

	job, err := h.ExportService.EnqueueUsersCSV(ctx, actorID(c), service.ExportFilters{
		Search:        strings.TrimSpace(req.Search),
		Role:          strings.TrimSpace(req.Role),
		AuthProvider:  strings.TrimSpace(req.AuthProvider),
		DeletedStatus: strings.TrimSpace(req.DeletedStatus),
		CreatedFrom:   strings.TrimSpace(req.CreatedFrom),
		CreatedTo:     strings.TrimSpace(req.CreatedTo),
		UpdatedFrom:   strings.TrimSpace(req.UpdatedFrom),
		UpdatedTo:     strings.TrimSpace(req.UpdatedTo),
		SortBy:        strings.TrimSpace(req.SortBy),
		SortDir:       strings.TrimSpace(req.SortDir),
	})
	if err != nil {
		log.Printf("admin enqueue export failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": i18n.T(c, "common.internal_error"),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status":  "success",
		"message": i18n.T(c, "admin.export_queued"),
		"data":    job,
	})
}

func (h *AdminHandler) GetExportJob(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	job, err := h.ExportService.GetJobForUser(ctx, c.Param("id"), actorID(c))
	if err != nil {
		h.writeAdminError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   job,
	})
}

func (h *AdminHandler) DownloadExportJob(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	job, file, err := h.ExportService.OpenCompletedFile(ctx, c.Param("id"), actorID(c))
	if err != nil {
		h.writeAdminError(c, err)
		return
	}
	defer file.Close()

	fileName := "users-export.csv"
	if job.FileName != nil && strings.TrimSpace(*job.FileName) != "" {
		fileName = *job.FileName
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+fileName+`"`)
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, file)
}

func (h *AdminHandler) KickUser(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.AuthService.AdminKickUser(ctx, actorID(c), c.Param("id")); err != nil {
		h.writeAdminError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": i18n.T(c, "admin.user_kicked"),
	})
}

func (h *AdminHandler) SetPassword(c *gin.Context) {
	var req adminSetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": i18n.T(c, "auth.invalid_request"),
			"code":    "auth.invalid_request",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.AuthService.AdminSetPassword(ctx, actorID(c), c.Param("id"), req.Password); err != nil {
		h.writeAdminError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": i18n.T(c, "admin.password_reset"),
	})
}

func (h *AdminHandler) SendResetPassword(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := h.AuthService.AdminSendPasswordReset(ctx, c.Param("id"), localeFromContext(c)); err != nil {
		h.writeAdminError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": i18n.T(c, "admin.reset_email_sent"),
	})
}

func (h *AdminHandler) VerifyEmail(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	user, err := h.AuthService.AdminVerifyEmail(ctx, c.Param("id"))
	if err != nil {
		h.writeAdminError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": i18n.T(c, "admin.email_verified"),
		"data":    user,
	})
}
