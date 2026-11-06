package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"go-be/internal/i18n"
	"go-be/internal/service"
)

type NotificationHandler struct {
	NotificationService *service.NotificationService
}

func NewNotificationHandler(notificationService *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{NotificationService: notificationService}
}

func (h *NotificationHandler) List(c *gin.Context) {
	userID := actorID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": i18n.T(c, "auth.unauthorized"),
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	items, unread, err := h.NotificationService.List(ctx, userID, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": i18n.T(c, "common.internal_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   items,
		"meta": gin.H{
			"unread_count": unread,
		},
	})
}

func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	userID := actorID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": i18n.T(c, "auth.unauthorized"),
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := h.NotificationService.UnreadCount(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": i18n.T(c, "common.internal_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"unread_count": count,
		},
	})
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID := actorID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": i18n.T(c, "auth.unauthorized"),
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.NotificationService.MarkRead(ctx, c.Param("id"), userID); err != nil {
		if errors.Is(err, service.ErrNotificationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "error",
				"message": i18n.T(c, "common.not_found"),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": i18n.T(c, "common.internal_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": i18n.T(c, "common.ok"),
	})
}

func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID := actorID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": i18n.T(c, "auth.unauthorized"),
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.NotificationService.MarkAllRead(ctx, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": i18n.T(c, "common.internal_error"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": i18n.T(c, "common.ok"),
	})
}
