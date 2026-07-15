package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Andrenoj11/sgscms-be/internal/dto"
	"github.com/Andrenoj11/sgscms-be/internal/helper"
	"github.com/Andrenoj11/sgscms-be/internal/repository"
	"github.com/Andrenoj11/sgscms-be/internal/response"
	"github.com/Andrenoj11/sgscms-be/internal/security"
	"github.com/Andrenoj11/sgscms-be/internal/service"
	"github.com/gin-gonic/gin"
)

type NewsHandler struct {
	service *service.NewsService
}

func NewNewsHandler(
	newsService *service.NewsService,
) *NewsHandler {
	return &NewsHandler{
		service: newsService,
	}
}

// Create godoc
//
// @Summary Membuat news
// @Tags Admin News
// @Accept json
// @Produce json
// @Security BearerAuth
// @Security XSignature
// @Security XTimestamp
// @Security XNonce
// @Param request body dto.CreateNewsRequest true "News data"
// @Success 201 {object} dto.SwaggerSuccessResponse{data=dto.NewsResponse}
// @Failure 400 {object} dto.SwaggerErrorResponse
// @Failure 401 {object} dto.SwaggerErrorResponse
// @Failure 409 {object} dto.SwaggerErrorResponse
// @Failure 500 {object} dto.SwaggerErrorResponse
// @Router /admin/news [post]

func (h *NewsHandler) Create(
	c *gin.Context,
) {
	var request dto.CreateNewsRequest

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

	news, err := h.service.Create(
		c.Request.Context(),
		request,
		admin.ID,
	)

	if handleNewsServiceError(c, err) {
		return
	}

	response.Success(
		c,
		http.StatusCreated,
		"News created successfully",
		news,
	)
}

// List godoc
//
// @Summary Daftar news admin
// @Tags Admin News
// @Produce json
// @Security BearerAuth
// @Security XSignature
// @Security XTimestamp
// @Security XNonce
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(10) maximum(100)
// @Param search query string false "Search"
// @Param status query string false "Status" Enums(draft,published,archived)
// @Param is_featured query bool false "Featured filter"
// @Success 200 {object} dto.SwaggerPaginatedResponse{data=[]dto.NewsResponse}
// @Failure 400 {object} dto.SwaggerErrorResponse
// @Failure 401 {object} dto.SwaggerErrorResponse
// @Failure 500 {object} dto.SwaggerErrorResponse
// @Router /admin/news [get]

func (h *NewsHandler) List(
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

	var isFeatured *bool

	isFeaturedQuery := strings.TrimSpace(
		c.Query("is_featured"),
	)

	if isFeaturedQuery != "" {
		value, err := strconv.ParseBool(
			isFeaturedQuery,
		)
		if err != nil {
			response.Error(
				c,
				http.StatusBadRequest,
				"Featured filter must be true or false",
				nil,
			)
			return
		}

		isFeatured = &value
	}

	filter := dto.NewsListQuery{
		Page:       page,
		Limit:      limit,
		Search:     c.Query("search"),
		Status:     c.Query("status"),
		IsFeatured: isFeatured,
	}

	newsList, meta, err := h.service.List(
		c.Request.Context(),
		filter,
	)

	if handleNewsServiceError(c, err) {
		return
	}

	response.SuccessWithMeta(
		c,
		http.StatusOK,
		"News retrieved successfully",
		newsList,
		meta,
	)
}


// GetByID godoc
//
// @Summary Detail news admin
// @Tags Admin News
// @Produce json
// @Security BearerAuth
// @Security XSignature
// @Security XTimestamp
// @Security XNonce
// @Param id path string true "News UUID"
// @Success 200 {object} dto.SwaggerSuccessResponse{data=dto.NewsResponse}
// @Failure 400 {object} dto.SwaggerErrorResponse
// @Failure 401 {object} dto.SwaggerErrorResponse
// @Failure 404 {object} dto.SwaggerErrorResponse
// @Failure 500 {object} dto.SwaggerErrorResponse
// @Router /admin/news/{id} [get]

func (h *NewsHandler) GetByID(
	c *gin.Context,
) {
	id := strings.TrimSpace(c.Param("id"))

	if !helper.IsValidUUID(id) {
		response.Error(
			c,
			http.StatusBadRequest,
			"News ID is invalid",
			nil,
		)
		return
	}

	news, err := h.service.GetByID(
		c.Request.Context(),
		id,
	)

	if handleNewsServiceError(c, err) {
		return
	}

	response.Success(
		c,
		http.StatusOK,
		"News retrieved successfully",
		news,
	)
}

// Update godoc
//
// @Summary Memperbarui news
// @Tags Admin News
// @Accept json
// @Produce json
// @Security BearerAuth
// @Security XSignature
// @Security XTimestamp
// @Security XNonce
// @Param id path string true "News UUID"
// @Param request body dto.UpdateNewsRequest true "News data"
// @Success 200 {object} dto.SwaggerSuccessResponse{data=dto.NewsResponse}
// @Failure 400 {object} dto.SwaggerErrorResponse
// @Failure 401 {object} dto.SwaggerErrorResponse
// @Failure 404 {object} dto.SwaggerErrorResponse
// @Failure 409 {object} dto.SwaggerErrorResponse
// @Failure 500 {object} dto.SwaggerErrorResponse
// @Router /admin/news/{id} [put]

func (h *NewsHandler) Update(
	c *gin.Context,
) {
	id := strings.TrimSpace(c.Param("id"))

	if !helper.IsValidUUID(id) {
		response.Error(
			c,
			http.StatusBadRequest,
			"News ID is invalid",
			nil,
		)
		return
	}

	var request dto.UpdateNewsRequest

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

	news, err := h.service.Update(
		c.Request.Context(),
		id,
		request,
		admin.ID,
	)

	if handleNewsServiceError(c, err) {
		return
	}

	response.Success(
		c,
		http.StatusOK,
		"News updated successfully",
		news,
	)
}

// Delete godoc
//
// @Summary Menghapus news
// @Description Melakukan soft delete.
// @Tags Admin News
// @Produce json
// @Security BearerAuth
// @Security XSignature
// @Security XTimestamp
// @Security XNonce
// @Param id path string true "News UUID"
// @Success 200 {object} dto.SwaggerSuccessResponse
// @Failure 400 {object} dto.SwaggerErrorResponse
// @Failure 401 {object} dto.SwaggerErrorResponse
// @Failure 404 {object} dto.SwaggerErrorResponse
// @Failure 500 {object} dto.SwaggerErrorResponse
// @Router /admin/news/{id} [delete]

func (h *NewsHandler) Delete(
	c *gin.Context,
) {
	id := strings.TrimSpace(c.Param("id"))

	if !helper.IsValidUUID(id) {
		response.Error(
			c,
			http.StatusBadRequest,
			"News ID is invalid",
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

	if handleNewsServiceError(c, err) {
		return
	}

	response.Success(
		c,
		http.StatusOK,
		"News deleted successfully",
		nil,
	)
}

func handleNewsServiceError(
	c *gin.Context,
	err error,
) bool {
	switch {
	case err == nil:
		return false

	case errors.Is(
		err,
		repository.ErrNewsNotFound,
	):
		response.Error(
			c,
			http.StatusNotFound,
			"News not found",
			nil,
		)

	case errors.Is(
		err,
		service.ErrNewsSlugExists,
	):
		response.Error(
			c,
			http.StatusConflict,
			"News slug already exists",
			nil,
		)

	case errors.Is(
		err,
		service.ErrInvalidNewsTitle,
	):
		response.Error(
			c,
			http.StatusBadRequest,
			"News title is invalid",
			nil,
		)

	case errors.Is(
		err,
		service.ErrInvalidNewsContent,
	):
		response.Error(
			c,
			http.StatusBadRequest,
			"News content is invalid",
			nil,
		)

	case errors.Is(
		err,
		service.ErrInvalidNewsStatus,
	):
		response.Error(
			c,
			http.StatusBadRequest,
			"News status must be draft, published, or archived",
			nil,
		)

	default:
		response.Error(
			c,
			http.StatusInternalServerError,
			"Internal server error",
			nil,
		)
	}

	return true
}