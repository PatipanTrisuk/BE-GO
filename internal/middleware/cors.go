package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func allowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("FRONTEND_URL"))
	if raw == "" {
		raw = "http://localhost:3000"
	}

	origins := []string{strings.TrimRight(raw, "/")}
	// Common local Next.js variants
	for _, extra := range []string{
		"http://localhost:3000",
		"http://127.0.0.1:3000",
	} {
		found := false
		for _, o := range origins {
			if o == extra {
				found = true
				break
			}
		}
		if !found {
			origins = append(origins, extra)
		}
	}
	return origins
}

// CORS allows the Next.js frontend to call the API from the browser.
func CORS() gin.HandlerFunc {
	origins := allowedOrigins()

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := ""
		for _, o := range origins {
			if origin == o {
				allowed = o
				break
			}
		}

		if allowed != "" {
			c.Header("Access-Control-Allow-Origin", allowed)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept-Language")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
