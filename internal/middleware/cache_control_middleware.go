package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func NoStore() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header(
			"Cache-Control",
			"no-store, no-cache, must-revalidate, private",
		)

		c.Header(
			"Pragma",
			"no-cache",
		)

		c.Header(
			"Expires",
			"0",
		)

		c.Next()
	}
}

func PublicCache(
	maxAgeSeconds int,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxAgeSeconds <= 0 {
			c.Next()
			return
		}

		cacheControl := strings.Join(
			[]string{
				"public",
				"max-age=" +
					intToString(maxAgeSeconds),
			},
			", ",
		)

		c.Header(
			"Cache-Control",
			cacheControl,
		)

		c.Next()
	}
}

func intToString(
	value int,
) string {
	if value == 0 {
		return "0"
	}

	digits := make([]byte, 0, 10)

	for value > 0 {
		digit := byte(value%10) + '0'

		digits = append(
			[]byte{digit},
			digits...,
		)

		value /= 10
	}

	return string(digits)
}