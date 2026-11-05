package response

import (
	"go-be/internal/i18n"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	StatusCode int    `json:"statusCode"`
	Code       string `json:"code"`
	Message    string `json:"message"`
}

func Error(
	c *gin.Context,
	statusCode int,
	code string,
	message string,
) {
	c.JSON(statusCode, ErrorResponse{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
	})
}

func BadRequest(c *gin.Context, key string) {
	Error(c, http.StatusBadRequest, "BAD_REQUEST", i18n.T(c, key))
}

func Unauthorized(c *gin.Context, key string) {
	Error(c, http.StatusUnauthorized, "UNAUTHORIZED", i18n.T(c, key))
}

func TooManyRequests(c *gin.Context) {
	Error(c, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", i18n.T(c, "common.too_many_requests"))
}

func InternalServerError(c *gin.Context, _ string) {
	Error(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", i18n.T(c, "common.internal_error"))
}
