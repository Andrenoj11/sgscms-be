package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Andrenoj11/sgscms-be/internal/repository"
	"github.com/Andrenoj11/sgscms-be/internal/response"
	"github.com/Andrenoj11/sgscms-be/internal/security"
	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	jwtManager      *security.JWTManager
	adminRepository repository.AdminRepository
}

func NewAuthMiddleware(
	jwtManager *security.JWTManager,
	adminRepository repository.AdminRepository,
) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager:      jwtManager,
		adminRepository: adminRepository,
	}
}

func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := extractBearerToken(
			c.GetHeader("Authorization"),
		)
		if err != nil {
			abortUnauthorized(
				c,
				"Authentication token is required",
			)
			return
		}

		claims, err := m.jwtManager.ParseAccessToken(
			tokenString,
		)

		switch {
		case errors.Is(err, security.ErrExpiredToken):
			abortUnauthorized(
				c,
				"Authentication token has expired",
			)
			return

		case err != nil:
			abortUnauthorized(
				c,
				"Authentication token is invalid",
			)
			return
		}

		admin, err := m.adminRepository.FindByID(
			c.Request.Context(),
			claims.Subject,
		)

		switch {
		case errors.Is(err, repository.ErrAdminNotFound):
			abortUnauthorized(
				c,
				"Admin account was not found",
			)
			return

		case err != nil:
			response.Error(
				c,
				http.StatusInternalServerError,
				"Internal server error",
				nil,
			)
			c.Abort()
			return
		}

		if !admin.IsActive {
			response.Error(
				c,
				http.StatusForbidden,
				"Admin account is inactive",
				nil,
			)
			c.Abort()
			return
		}

		security.SetCurrentAdmin(c, admin)

		c.Next()
	}
}

func extractBearerToken(
	authorizationHeader string,
) (string, error) {
	parts := strings.Fields(authorizationHeader)

	if len(parts) != 2 {
		return "", security.ErrInvalidToken
	}

	if !strings.EqualFold(parts[0], "Bearer") {
		return "", security.ErrInvalidToken
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", security.ErrInvalidToken
	}

	return token, nil
}

func abortUnauthorized(
	c *gin.Context,
	message string,
) {
	response.Error(
		c,
		http.StatusUnauthorized,
		message,
		nil,
	)

	c.Abort()
}