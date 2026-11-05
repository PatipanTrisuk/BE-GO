package middleware

import (
	"go-be/internal/service"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const LocaleKey = "locale"

func LocaleMiddleware() gin.HandlerFunc {
	supported := map[string]bool{"en": true, "th": true}
	return func(c *gin.Context) {
		header := c.GetHeader("Accept-Language")
		lang := "en"
		if header != "" {
			primary := strings.Split(strings.Split(header, ",")[0], ";")[0]
			primary = strings.ToLower(strings.TrimSpace(primary))
			if len(primary) >= 2 {
				primary = primary[:2]
			}
			if supported[primary] {
				lang = primary
			}
		}
		c.Set(LocaleKey, lang)
		c.Next()
	}
}

func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Unauthorized",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)

		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Unauthorized",
			})
			return
		}

		claims := &service.AccessTokenClaims{}

		token, err := jwt.ParseWithClaims(
			parts[1],
			claims,
			func(token *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			},
			jwt.WithValidMethods([]string{"HS256"}),
		)

		if err != nil || !token.Valid || claims.Subject == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Unauthorized",
			})
			return
		}

		c.Set("userID", claims.Subject)
		c.Set("role", claims.Role)

		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		roleStr, _ := role.(string)
		if !strings.EqualFold(roleStr, "admin") {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"status":  "error",
				"message": "Forbidden",
				"code":    "auth.forbidden",
			})
			return
		}
		c.Next()
	}
}
