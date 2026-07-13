package handler

import (
	"errors"
	"net/http"

	"github.com/Andrenoj11/sgscms-be/internal/dto"
	"github.com/Andrenoj11/sgscms-be/internal/response"
	"github.com/Andrenoj11/sgscms-be/internal/service"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(
	authService *service.AuthService,
) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
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

	loginResponse, err := h.authService.Login(
		c.Request.Context(),
		request,
	)

	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		response.Error(
			c,
			http.StatusUnauthorized,
			"Invalid email or password",
			nil,
		)
		return

	case errors.Is(err, service.ErrAdminInactive):
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

	response.Success(
		c,
		http.StatusOK,
		"Login successful",
		loginResponse,
	)
}