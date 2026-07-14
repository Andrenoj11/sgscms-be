package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Andrenoj11/sgscms-be/internal/dto"
	"github.com/Andrenoj11/sgscms-be/internal/repository"
	"github.com/Andrenoj11/sgscms-be/internal/response"
	"github.com/Andrenoj11/sgscms-be/internal/service"
	"github.com/gin-gonic/gin"
)

type PublicHandler struct {
	service *service.PublicService
}

func NewPublicHandler(
	publicService *service.PublicService,
) *PublicHandler {
	return &PublicHandler{
		service: publicService,
	}
}

func (h *PublicHandler) ListPracticeAreas(
	c *gin.Context,
) {
	practiceAreas, err :=
		h.service.ListPracticeAreas(
			c.Request.Context(),
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

	response.Success(
		c,
		http.StatusOK,
		"Practice areas retrieved successfully",
		practiceAreas,
	)
}

func (h *PublicHandler) ListTeams(
	c *gin.Context,
) {
	filter := dto.PublicListQuery{
		Page: parsePositiveInt(
			c.Query("page"),
			1,
		),
		Limit: parsePositiveInt(
			c.Query("limit"),
			10,
		),
		Search: c.Query("search"),
	}

	teams, meta, err := h.service.ListTeams(
		c.Request.Context(),
		filter,
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
		"Teams retrieved successfully",
		teams,
		meta,
	)
}

func (h *PublicHandler) GetTeamBySlug(
	c *gin.Context,
) {
	slugValue := strings.TrimSpace(
		c.Param("slug"),
	)

	if slugValue == "" {
		response.Error(
			c,
			http.StatusBadRequest,
			"Team slug is required",
			nil,
		)
		return
	}

	team, err := h.service.GetTeamBySlug(
		c.Request.Context(),
		slugValue,
	)

	switch {
	case errors.Is(
		err,
		repository.ErrTeamNotFound,
	):
		response.Error(
			c,
			http.StatusNotFound,
			"Team not found",
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
		"Team retrieved successfully",
		team,
	)
}

func (h *PublicHandler) ListNews(
	c *gin.Context,
) {
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

	filter := dto.PublicNewsListQuery{
		Page: parsePositiveInt(
			c.Query("page"),
			1,
		),
		Limit: parsePositiveInt(
			c.Query("limit"),
			10,
		),
		Search:     c.Query("search"),
		IsFeatured: isFeatured,
	}

	newsList, meta, err := h.service.ListNews(
		c.Request.Context(),
		filter,
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
		"News retrieved successfully",
		newsList,
		meta,
	)
}

func (h *PublicHandler) GetNewsBySlug(
	c *gin.Context,
) {
	slugValue := strings.TrimSpace(
		c.Param("slug"),
	)

	if slugValue == "" {
		response.Error(
			c,
			http.StatusBadRequest,
			"News slug is required",
			nil,
		)
		return
	}

	news, err := h.service.GetNewsBySlug(
		c.Request.Context(),
		slugValue,
	)

	switch {
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
		"News retrieved successfully",
		news,
	)
}