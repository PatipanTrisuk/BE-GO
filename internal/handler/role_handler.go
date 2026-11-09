package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"go-be/internal/i18n"
	"go-be/internal/service"
)

type RoleHandler struct {
	RoleService *service.RoleService
}

func NewRoleHandler(roleService *service.RoleService) *RoleHandler {
	return &RoleHandler{RoleService: roleService}
}

type roleWriteRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

func (h *RoleHandler) writeRoleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrRoleNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": i18n.T(c, "admin.role_not_found"),
			"code":    "admin.role_not_found",
		})
	case errors.Is(err, service.ErrRoleExists):
		c.JSON(http.StatusConflict, gin.H{
			"status":  "error",
			"message": i18n.T(c, "admin.role_exists"),
			"code":    "admin.role_exists",
		})
	case errors.Is(err, service.ErrRoleSystem):
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": i18n.T(c, "admin.role_system"),
			"code":    "admin.role_system",
		})
	case errors.Is(err, service.ErrRoleInUse):
		c.JSON(http.StatusConflict, gin.H{
			"status":  "error",
			"message": i18n.T(c, "admin.role_in_use"),
			"code":    "admin.role_in_use",
		})
	case errors.Is(err, service.ErrInvalidPermission):
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": i18n.T(c, "admin.invalid_permission"),
			"code":    "admin.invalid_permission",
		})
	case errors.Is(err, service.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": i18n.T(c, "auth.invalid_input"),
			"code":    "auth.invalid_input",
		})
	default:
		log.Printf("role action failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": i18n.T(c, "common.internal_error"),
		})
	}
}

func (h *RoleHandler) ListPermissions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	items, err := h.RoleService.ListPermissions(ctx)
	if err != nil {
		h.writeRoleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   items,
	})
}

func (h *RoleHandler) ListRoles(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	items, err := h.RoleService.ListRoles(ctx)
	if err != nil {
		h.writeRoleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   items,
	})
}

func (h *RoleHandler) GetRole(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	role, err := h.RoleService.GetRole(ctx, c.Param("id"))
	if err != nil {
		h.writeRoleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   role,
	})
}

func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req roleWriteRequest
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

	role, err := h.RoleService.CreateRole(ctx, service.RoleInput{
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
	})
	if err != nil {
		h.writeRoleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": i18n.T(c, "admin.role_created"),
		"data":    role,
	})
}

func (h *RoleHandler) UpdateRole(c *gin.Context) {
	var req roleWriteRequest
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

	role, err := h.RoleService.UpdateRole(ctx, c.Param("id"), service.RoleInput{
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
	})
	if err != nil {
		h.writeRoleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": i18n.T(c, "admin.role_updated"),
		"data":    role,
	})
}

func (h *RoleHandler) DeleteRole(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.RoleService.DeleteRole(ctx, c.Param("id")); err != nil {
		h.writeRoleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": i18n.T(c, "admin.role_deleted"),
	})
}
