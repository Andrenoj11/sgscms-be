package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Andrenoj11/sgscms-be/internal/dto"
	"github.com/Andrenoj11/sgscms-be/internal/repository"
	"github.com/Andrenoj11/sgscms-be/internal/response"
	"github.com/Andrenoj11/sgscms-be/internal/security"
	"github.com/Andrenoj11/sgscms-be/internal/service"
	"github.com/gin-gonic/gin"

	"github.com/Andrenoj11/sgscms-be/internal/helper"
)

type PracticeAreaHandler struct {
	service *service.PracticeAreaService
}

func NewPracticeAreaHandler(
	practiceAreaService *service.PracticeAreaService,
) *PracticeAreaHandler {
	return &PracticeAreaHandler{
		service: practiceAreaService,
	}
}

// Create godoc
//
// @Summary Membuat practice area
// @Tags Admin Practice Areas
// @Accept json
// @Produce json
// @Security BearerAuth
// @Security XSignature
// @Security XTimestamp
// @Security XNonce
// @Param request body dto.CreatePracticeAreaRequest true "Practice-area data"
// @Success 201 {object} dto.SwaggerSuccessResponse{data=dto.PracticeAreaResponse}
// @Failure 400 {object} dto.SwaggerErrorResponse
// @Failure 401 {object} dto.SwaggerErrorResponse
// @Failure 409 {object} dto.SwaggerErrorResponse
// @Failure 500 {object} dto.SwaggerErrorResponse
// @Router /admin/practice-areas [post]
func (h *PracticeAreaHandler) Create(
	c *gin.Context,
) {
	var request dto.CreatePracticeAreaRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			"Invalid request body",
			err.Error(),
		)
		return
	}

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

	practiceArea, err := h.service.Create(
		c.Request.Context(),
		request,
		admin.ID,
	)

	switch {
	case errors.Is(
		err,
		service.ErrPracticeAreaNameExists,
	):
		response.Error(
			c,
			http.StatusConflict,
			"Practice area name already exists",
			nil,
		)
		return

	case errors.Is(
		err,
		service.ErrPracticeAreaSlugExists,
	):
		response.Error(
			c,
			http.StatusConflict,
			"Practice area slug already exists",
			nil,
		)
		return

	case errors.Is(
		err,
		service.ErrInvalidPracticeAreaName,
	):
		response.Error(
			c,
			http.StatusBadRequest,
			"Practice area name is invalid",
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
		http.StatusCreated,
		"Practice area created successfully",
		practiceArea,
	)
}

// GetByID godoc
//
// @Summary Detail practice area
// @Tags Admin Practice Areas
// @Produce json
// @Security BearerAuth
// @Security XSignature
// @Security XTimestamp
// @Security XNonce
// @Param id path string true "Practice-area UUID"
// @Success 200 {object} dto.SwaggerSuccessResponse{data=dto.PracticeAreaResponse}
// @Failure 400 {object} dto.SwaggerErrorResponse
// @Failure 401 {object} dto.SwaggerErrorResponse
// @Failure 404 {object} dto.SwaggerErrorResponse
// @Failure 500 {object} dto.SwaggerErrorResponse
// @Router /admin/practice-areas/{id} [get]
func (h *PracticeAreaHandler) GetByID(
	c *gin.Context,
) {
	id := strings.TrimSpace(c.Param("id"))

	if id == "" {
		response.Error(
			c,
			http.StatusBadRequest,
			"Practice area ID is required",
			nil,
		)
		return
	}

	if !helper.IsValidUUID(id) {
		response.Error(
			c,
			http.StatusBadRequest,
			"Practice area ID is invalid",
			nil,
		)
		return
	}

	practiceArea, err := h.service.GetByID(
		c.Request.Context(),
		id,
	)

	switch {
	case errors.Is(
		err,
		repository.ErrPracticeAreaNotFound,
	):
		response.Error(
			c,
			http.StatusNotFound,
			"Practice area not found",
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
		"Practice area retrieved successfully",
		practiceArea,
	)
}

// List godoc
//
// @Summary Daftar practice area admin
// @Tags Admin Practice Areas
// @Produce json
// @Security BearerAuth
// @Security XSignature
// @Security XTimestamp
// @Security XNonce
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(10) maximum(100)
// @Param search query string false "Search by name"
// @Param active query bool false "Filter active status"
// @Success 200 {object} dto.SwaggerPaginatedResponse{data=[]dto.PracticeAreaResponse}
// @Failure 400 {object} dto.SwaggerErrorResponse
// @Failure 401 {object} dto.SwaggerErrorResponse
// @Failure 500 {object} dto.SwaggerErrorResponse
// @Router /admin/practice-areas [get]
func (h *PracticeAreaHandler) List(
	c *gin.Context,
) {
	page := parsePositiveInt(
		c.Query("page"),
		1,
	)

	limit := parsePositiveInt(
		c.Query("limit"),
		10,
	)

	var active *bool

	activeQuery := strings.TrimSpace(
		c.Query("active"),
	)

	if activeQuery != "" {
		value, err := strconv.ParseBool(
			activeQuery,
		)
		if err != nil {
			response.Error(
				c,
				http.StatusBadRequest,
				"Active filter must be true or false",
				nil,
			)
			return
		}

		active = &value
	}

	query := dto.PracticeAreaListQuery{
		Page:   page,
		Limit:  limit,
		Search: c.Query("search"),
		Active: active,
	}

	practiceAreas, meta, err := h.service.List(
		c.Request.Context(),
		query,
	)
	if err != nil {
		response.Error(
			c,
			http.StatusInternalServerError,
			"Internal server error",
			nil,
		)
		return
	}

	response.SuccessWithMeta(
		c,
		http.StatusOK,
		"Practice areas retrieved successfully",
		practiceAreas,
		meta,
	)
}

func parsePositiveInt(
	value string,
	fallback int,
) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}

	return parsed
}

// Update godoc
//
// @Summary Memperbarui practice area
// @Tags Admin Practice Areas
// @Accept json
// @Produce json
// @Security BearerAuth
// @Security XSignature
// @Security XTimestamp
// @Security XNonce
// @Param id path string true "Practice-area UUID"
// @Param request body dto.UpdatePracticeAreaRequest true "Practice-area data"
// @Success 200 {object} dto.SwaggerSuccessResponse{data=dto.PracticeAreaResponse}
// @Failure 400 {object} dto.SwaggerErrorResponse
// @Failure 401 {object} dto.SwaggerErrorResponse
// @Failure 404 {object} dto.SwaggerErrorResponse
// @Failure 409 {object} dto.SwaggerErrorResponse
// @Failure 500 {object} dto.SwaggerErrorResponse
// @Router /admin/practice-areas/{id} [put]
func (h *PracticeAreaHandler) Update(
	c *gin.Context,
) {
	id := strings.TrimSpace(c.Param("id"))

	if id == "" {
		response.Error(
			c,
			http.StatusBadRequest,
			"Practice area ID is required",
			nil,
		)
		return
	}

	if !helper.IsValidUUID(id) {
		response.Error(
			c,
			http.StatusBadRequest,
			"Practice area ID is invalid",
			nil,
		)
		return
	}

	var request dto.UpdatePracticeAreaRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			"Invalid request body",
			err.Error(),
		)
		return
	}

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

	practiceArea, err := h.service.Update(
		c.Request.Context(),
		id,
		request,
		admin.ID,
	)

	switch {
	case errors.Is(
		err,
		repository.ErrPracticeAreaNotFound,
	):
		response.Error(
			c,
			http.StatusNotFound,
			"Practice area not found",
			nil,
		)
		return

	case errors.Is(
		err,
		service.ErrPracticeAreaNameExists,
	):
		response.Error(
			c,
			http.StatusConflict,
			"Practice area name already exists",
			nil,
		)
		return

	case errors.Is(
		err,
		service.ErrPracticeAreaSlugExists,
	):
		response.Error(
			c,
			http.StatusConflict,
			"Practice area slug already exists",
			nil,
		)
		return

	case errors.Is(
		err,
		service.ErrInvalidPracticeAreaName,
	):
		response.Error(
			c,
			http.StatusBadRequest,
			"Practice area name is invalid",
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
		"Practice area updated successfully",
		practiceArea,
	)
}

// Delete godoc
//
// @Summary Menghapus practice area
// @Description Melakukan soft delete.
// @Tags Admin Practice Areas
// @Produce json
// @Security BearerAuth
// @Security XSignature
// @Security XTimestamp
// @Security XNonce
// @Param id path string true "Practice-area UUID"
// @Success 200 {object} dto.SwaggerSuccessResponse
// @Failure 400 {object} dto.SwaggerErrorResponse
// @Failure 401 {object} dto.SwaggerErrorResponse
// @Failure 404 {object} dto.SwaggerErrorResponse
// @Failure 500 {object} dto.SwaggerErrorResponse
// @Router /admin/practice-areas/{id} [delete]
func (h *PracticeAreaHandler) Delete(
	c *gin.Context,
) {
	id := strings.TrimSpace(c.Param("id"))

	if id == "" {
		response.Error(
			c,
			http.StatusBadRequest,
			"Practice area ID is required",
			nil,
		)
		return
	}

	if !helper.IsValidUUID(id) {
		response.Error(
			c,
			http.StatusBadRequest,
			"Practice area ID is invalid",
			nil,
		)
		return
	}

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

	err = h.service.Delete(
		c.Request.Context(),
		id,
		admin.ID,
	)

	switch {
	case errors.Is(
		err,
		repository.ErrPracticeAreaNotFound,
	):
		response.Error(
			c,
			http.StatusNotFound,
			"Practice area not found",
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
		"Practice area deleted successfully",
		nil,
	)
}
