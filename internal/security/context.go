package security

import (
	"errors"

	"github.com/Andrenoj11/sgscms-be/internal/domain"
	"github.com/gin-gonic/gin"
)

const (
	CurrentAdminContextKey   = "current_admin"
	CurrentSessionContextKey = "current_session_id"
)

var (
	ErrCurrentAdminNotFound = errors.New(
		"current admin not found in context",
	)

	ErrCurrentSessionNotFound = errors.New(
		"current session not found in context",
	)
)

func SetCurrentAdmin(
	c *gin.Context,
	admin *domain.Admin,
) {
	c.Set(CurrentAdminContextKey, admin)
}

func GetCurrentAdmin(
	c *gin.Context,
) (*domain.Admin, error) {
	value, exists := c.Get(
		CurrentAdminContextKey,
	)
	if !exists {
		return nil, ErrCurrentAdminNotFound
	}

	admin, ok := value.(*domain.Admin)

	if !ok || admin == nil {
		return nil, ErrCurrentAdminNotFound
	}

	return admin, nil
}

func SetCurrentSessionID(
	c *gin.Context,
	sessionID string,
) {
	c.Set(
		CurrentSessionContextKey,
		sessionID,
	)
}

func GetCurrentSessionID(
	c *gin.Context,
) (string, error) {
	value, exists := c.Get(
		CurrentSessionContextKey,
	)
	if !exists {
		return "", ErrCurrentSessionNotFound
	}

	sessionID, ok := value.(string)

	if !ok || sessionID == "" {
		return "", ErrCurrentSessionNotFound
	}

	return sessionID, nil
}