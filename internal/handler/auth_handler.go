package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/Andrenoj11/sgscms-be/internal/dto"
	"github.com/Andrenoj11/sgscms-be/internal/response"
	"github.com/Andrenoj11/sgscms-be/internal/security"
	"github.com/Andrenoj11/sgscms-be/internal/service"
	"github.com/gin-gonic/gin"
)

const refreshCookieName = "sgs_refresh_token"

type AuthHandler struct {
	authService *service.AuthService

	refreshTTL time.Duration

	cookieSecure bool
}

func NewAuthHandler(
	authService *service.AuthService,
	refreshTTL time.Duration,
	cookieSecure bool,
) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		refreshTTL:   refreshTTL,
		cookieSecure: cookieSecure,
	}
}

func (h *AuthHandler) Login(
	c *gin.Context,
) {
	var request dto.LoginRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			"Invalid request body",
			err.Error(),
		)
		return
	}

	result, err := h.authService.Login(
		c.Request.Context(),
		request,
		c.Request.UserAgent(),
		c.ClientIP(),
	)

	switch {
	case errors.Is(
		err,
		service.ErrInvalidCredentials,
	):
		response.Error(
			c,
			http.StatusUnauthorized,
			"Invalid email or password",
			nil,
		)
		return

	case errors.Is(
		err,
		service.ErrAdminInactive,
	):
		response.Error(
			c,
			http.StatusForbidden,
			"Admin account is inactive",
			nil,
		)
		return

	case err != nil:
		response.Error(
			c,
			http.StatusInternalServerError,
			"Internal server error",
			nil,
		)
		return
	}

	h.setRefreshCookie(
		c,
		result.RefreshToken,
	)

	response.Success(
		c,
		http.StatusOK,
		"Login successful",
		result.Response,
	)
}

func (h *AuthHandler) Refresh(
	c *gin.Context,
) {
	refreshToken, err := c.Cookie(
		refreshCookieName,
	)
	if err != nil {
		response.Error(
			c,
			http.StatusUnauthorized,
			"Refresh token is required",
			nil,
		)
		return
	}

	result, err := h.authService.Refresh(
		c.Request.Context(),
		refreshToken,
		c.Request.UserAgent(),
		c.ClientIP(),
	)

	switch {
	case errors.Is(
		err,
		service.ErrInvalidRefreshToken,
	),
		errors.Is(
			err,
			service.ErrExpiredRefreshToken,
		):
		h.clearRefreshCookie(c)

		response.Error(
			c,
			http.StatusUnauthorized,
			"Refresh token is invalid or expired",
			nil,
		)
		return

	case errors.Is(
		err,
		service.ErrAdminInactive,
	):
		h.clearRefreshCookie(c)

		response.Error(
			c,
			http.StatusForbidden,
			"Admin account is inactive",
			nil,
		)
		return

	case err != nil:
		response.Error(
			c,
			http.StatusInternalServerError,
			"Internal server error",
			nil,
		)
		return
	}

	h.setRefreshCookie(
		c,
		result.RefreshToken,
	)

	response.Success(
		c,
		http.StatusOK,
		"Token refreshed successfully",
		result.Response,
	)
}

func (h *AuthHandler) Logout(
	c *gin.Context,
) {
	refreshToken, _ := c.Cookie(
		refreshCookieName,
	)

	if err := h.authService.Logout(
		c.Request.Context(),
		refreshToken,
	); err != nil {
		response.Error(
			c,
			http.StatusInternalServerError,
			"Internal server error",
			nil,
		)
		return
	}

	h.clearRefreshCookie(c)

	response.Success(
		c,
		http.StatusOK,
		"Logout successful",
		nil,
	)
}

func (h *AuthHandler) Me(
	c *gin.Context,
) {
	admin, err := security.GetCurrentAdmin(c)
	if err != nil {
		response.Error(
			c,
			http.StatusUnauthorized,
			"Admin is not authenticated",
			nil,
		)
		return
	}

	response.Success(
		c,
		http.StatusOK,
		"Current admin retrieved successfully",
		dto.AdminResponse{
			ID:    admin.ID,
			Name:  admin.Name,
			Email: admin.Email,
			Role:  string(admin.Role),
		},
	)
}

func (h *AuthHandler) setRefreshCookie(
	c *gin.Context,
	token string,
) {
	c.SetSameSite(
		http.SameSiteLaxMode,
	)

	c.SetCookie(
		refreshCookieName,
		token,
		int(h.refreshTTL.Seconds()),
		"/api/v1/admin/auth",
		"",
		h.cookieSecure,
		true,
	)
}

func (h *AuthHandler) clearRefreshCookie(
	c *gin.Context,
) {
	c.SetSameSite(
		http.SameSiteLaxMode,
	)

	c.SetCookie(
		refreshCookieName,
		"",
		-1,
		"/api/v1/admin/auth",
		"",
		h.cookieSecure,
		true,
	)
}