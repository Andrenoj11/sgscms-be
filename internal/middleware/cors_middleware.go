package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS(
	allowedOrigins []string,
) gin.HandlerFunc {
	allowed := make(
		map[string]struct{},
		len(allowedOrigins),
	)

	for _, origin := range allowedOrigins {
		allowed[strings.TrimSpace(origin)] =
			struct{}{}
	}

	return func(c *gin.Context) {
		origin := strings.TrimSpace(
			c.GetHeader("Origin"),
		)

		if origin != "" {
			if _, exists := allowed[origin]; exists {
				c.Header(
					"Access-Control-Allow-Origin",
					origin,
				)

				c.Header(
					"Access-Control-Allow-Credentials",
					"true",
				)

				c.Header(
					"Vary",
					"Origin",
				)
			}
		}

		c.Header(
			"Access-Control-Allow-Headers",
			"Authorization, Content-Type, X-Timestamp, X-Nonce, X-Signature",
		)

		c.Header(
			"Access-Control-Allow-Methods",
			"GET, POST, PUT, PATCH, DELETE, OPTIONS",
		)

		c.Header(
			"Access-Control-Max-Age",
			"600",
		)

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(
				http.StatusNoContent,
			)
			return
		}

		c.Next()
	}
}