package middleware

import (
	"net/http"
	"strings"

	"github.com/Andrenoj11/sgscms-be/internal/response"
	"github.com/gin-gonic/gin"
)

func JSONBodyLimit(
	maxBytes int64,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes <= 0 {
			response.Error(
				c,
				http.StatusInternalServerError,
				"Server configuration is invalid",
				nil,
			)

			c.Abort()
			return
		}

		contentType := strings.ToLower(
			strings.TrimSpace(
				c.GetHeader("Content-Type"),
			),
		)

		if !strings.HasPrefix(
			contentType,
			"application/json",
		) {
			c.Next()
			return
		}

		if c.Request.Body == nil {
			c.Next()
			return
		}

		c.Request.Body = http.MaxBytesReader(
			c.Writer,
			c.Request.Body,
			maxBytes,
		)

		c.Next()
	}
}