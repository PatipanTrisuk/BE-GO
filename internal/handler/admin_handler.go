package handler

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"go-be/internal/i18n"
	"go-be/internal/repository"
)

const (
	defaultPage  = 1
	defaultLimit = 10
	maxLimit     = 100
)

type AdminHandler struct {
	UserRepo *repository.UserRepository
	DB       *pgxpool.Pool
}

func NewAdminHandler(userRepo *repository.UserRepository, db *pgxpool.Pool) *AdminHandler {
	return &AdminHandler{
		UserRepo: userRepo,
		DB:       db,
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

func (h *AdminHandler) ListUsers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	page := parsePositiveInt(c.Query("page"), defaultPage, 0)
	limit := parsePositiveInt(c.Query("limit"), defaultLimit, maxLimit)
	query := strings.TrimSpace(c.Query("search"))

	users, err := h.UserRepo.List(ctx, repository.ListParams{
		Page:  page,
		Limit: limit,
		Query: query,
	})
	if err != nil {
		log.Printf("admin list users failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": i18n.T(c, "common.internal_error"),
		})
		return
	}

	total, err := h.UserRepo.Count(ctx, query)
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

	userCount, err := h.UserRepo.Count(ctx, "")
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
