package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"go-be/internal/authz"
	"go-be/internal/service"
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
		c.Set("roleID", claims.RoleID)

		c.Next()
	}
}

func roleFromContext(c *gin.Context) string {
	role, _ := c.Get("role")
	roleStr, _ := role.(string)
	return roleStr
}

// roleKeyFromContext prefers role_id for permission lookups.
func roleKeyFromContext(c *gin.Context) string {
	if id, ok := c.Get("roleID"); ok {
		if s, _ := id.(string); strings.TrimSpace(s) != "" {
			return s
		}
	}
	return roleFromContext(c)
}

func forbid(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"status":  "error",
		"message": "Forbidden",
		"code":    "auth.forbidden",
	})
}

// RequireAdmin allows only the admin role (legacy helper).
func RequireAdmin() gin.HandlerFunc {
	return RequireRoles(authz.RoleAdmin)
}

// RequireRoles allows access when the JWT role matches any of the given roles.
func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[authz.NormalizeRole(r)] = struct{}{}
	}
	return func(c *gin.Context) {
		role := authz.NormalizeRole(roleFromContext(c))
		if _, ok := allowed[role]; !ok {
			forbid(c)
			return
		}
		c.Next()
	}
}

// RequirePermission requires the caller's role to have every listed permission.
func RequirePermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !authz.HasAllPermissions(roleKeyFromContext(c), permissions...) {
			forbid(c)
			return
		}
		c.Next()
	}
}

// RequireAnyPermission requires the caller's role to have at least one listed permission.
func RequireAnyPermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !authz.HasAnyPermission(roleKeyFromContext(c), permissions...) {
			forbid(c)
			return
		}
		c.Next()
	}
}

// RequireAdminArea allows any role that has at least one management permission.
func RequireAdminArea() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !authz.CanAccessAdminArea(roleKeyFromContext(c)) {
			forbid(c)
			return
		}
		c.Next()
	}
}
