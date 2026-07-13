package security

import (
	"errors"

	"github.com/Andrenoj11/sgscms-be/internal/domain"
	"github.com/gin-gonic/gin"
)

const (
	CurrentAdminContextKey = "current_admin"
)

var (
	ErrCurrentAdminNotFound = errors.New(
		"current admin not found in context",
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
	value, exists := c.Get(CurrentAdminContextKey)
	if !exists {
		return nil, ErrCurrentAdminNotFound
	}

	admin, ok := value.(*domain.Admin)
	if !ok || admin == nil {
		return nil, ErrCurrentAdminNotFound
	}

	return admin, nil
}