package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(
			c.GetHeader(RequestIDHeader),
		)

		if !isValidRequestID(requestID) {
			requestID = uuid.NewString()
		}

		c.Set(
			RequestIDHeader,
			requestID,
		)

		c.Header(
			RequestIDHeader,
			requestID,
		)

		c.Next()
	}
}

func GetRequestID(
	c *gin.Context,
) string {
	value, exists := c.Get(
		RequestIDHeader,
	)
	if !exists {
		return ""
	}

	requestID, ok := value.(string)
	if !ok {
		return ""
	}

	return requestID
}

func isValidRequestID(
	requestID string,
) bool {
	if requestID == "" {
		return false
	}

	if len(requestID) > 100 {
		return false
	}

	for _, character := range requestID {
		isAllowed :=
			character >= 'a' &&
				character <= 'z' ||
				character >= 'A' &&
					character <= 'Z' ||
				character >= '0' &&
					character <= '9' ||
				character == '-' ||
				character == '_'

		if !isAllowed {
			return false
		}
	}

	return true
}