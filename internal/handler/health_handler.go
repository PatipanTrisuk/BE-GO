package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	DB *pgxpool.Pool
}

func NewHealthHandler(db *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{DB: db}
}

func (h *HealthHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Go Auth API is running",
	})
}

func (h *HealthHandler) DatabaseHealthCheck(c *gin.Context) {
	if err := h.DB.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Database unavailable",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "PostgreSQL connected",
	})
}
