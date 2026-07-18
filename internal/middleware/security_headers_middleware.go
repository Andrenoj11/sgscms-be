package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	apiContentSecurityPolicy     = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'"
	swaggerContentSecurityPolicy = "default-src 'none'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'"
)

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header(
			"X-Content-Type-Options",
			"nosniff",
		)

		c.Header(
			"X-Frame-Options",
			"DENY",
		)

		c.Header(
			"Referrer-Policy",
			"strict-origin-when-cross-origin",
		)

		c.Header(
			"Permissions-Policy",
			"camera=(), microphone=(), geolocation=()",
		)

		c.Header(
			"Cross-Origin-Opener-Policy",
			"same-origin",
		)

		c.Header(
			"Cross-Origin-Resource-Policy",
			"same-site",
		)

		c.Header(
			"X-Permitted-Cross-Domain-Policies",
			"none",
		)

		contentSecurityPolicy := apiContentSecurityPolicy
		if c.Request.URL.Path == "/swagger" ||
			strings.HasPrefix(c.Request.URL.Path, "/swagger/") {
			contentSecurityPolicy = swaggerContentSecurityPolicy
		}

		c.Header("Content-Security-Policy", contentSecurityPolicy)

		c.Next()
	}
}
