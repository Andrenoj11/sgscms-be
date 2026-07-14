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

type TeamHandler struct {
	service *service.TeamService
}

func NewTeamHandler(
	teamService *service.TeamService,
) *TeamHandler {
	return &TeamHandler{
		service: teamService,
	}
}

func (h *TeamHandler) Create(
	c *gin.Context,
) {
	var request dto.CreateTeamRequest

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

	team, err := h.service.Create(
		c.Request.Context(),
		request,
		admin.ID,
	)

	if handleTeamServiceError(c, err) {
		return
	}

	response.Success(
		c,
		http.StatusCreated,
		"Team created successfully",
		team,
	)
}

func (h *TeamHandler) List(
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

	var published *bool

	publishedQuery := strings.TrimSpace(
		c.Query("published"),
	)

	if publishedQuery != "" {
		value, err := strconv.ParseBool(
			publishedQuery,
		)
		if err != nil {
			response.Error(
				c,
				http.StatusBadRequest,
				"Published filter must be true or false",
				nil,
			)
			return
		}

		published = &value
	}

	filter := dto.TeamListQuery{
		Page:      page,
		Limit:     limit,
		Search:    c.Query("search"),
		Published: published,
	}

	teams, meta, err := h.service.List(
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

func (h *TeamHandler) GetByID(
	c *gin.Context,
) {
	id := strings.TrimSpace(c.Param("id"))

	if !helper.IsValidUUID(id) {
		response.Error(
			c,
			http.StatusBadRequest,
			"Team ID is invalid",
			nil,
		)
		return
	}

	team, err := h.service.GetByID(
		c.Request.Context(),
		id,
	)

	if handleTeamServiceError(c, err) {
		return
	}

	response.Success(
		c,
		http.StatusOK,
		"Team retrieved successfully",
		team,
	)
}

func (h *TeamHandler) Update(
	c *gin.Context,
) {
	id := strings.TrimSpace(c.Param("id"))

	if !helper.IsValidUUID(id) {
		response.Error(
			c,
			http.StatusBadRequest,
			"Team ID is invalid",
			nil,
		)
		return
	}

	var request dto.UpdateTeamRequest

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

	team, err := h.service.Update(
		c.Request.Context(),
		id,
		request,
		admin.ID,
	)

	if handleTeamServiceError(c, err) {
		return
	}

	response.Success(
		c,
		http.StatusOK,
		"Team updated successfully",
		team,
	)
}

func (h *TeamHandler) Delete(
	c *gin.Context,
) {
	id := strings.TrimSpace(c.Param("id"))

	if !helper.IsValidUUID(id) {
		response.Error(
			c,
			http.StatusBadRequest,
			"Team ID is invalid",
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

	if handleTeamServiceError(c, err) {
		return
	}

	response.Success(
		c,
		http.StatusOK,
		"Team deleted successfully",
		nil,
	)
}

func handleTeamServiceError(
	c *gin.Context,
	err error,
) bool {
	switch {
	case err == nil:
		return false

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

	case errors.Is(
		err,
		service.ErrTeamNameExists,
	):
		response.Error(
			c,
			http.StatusConflict,
			"Team name already exists",
			nil,
		)

	case errors.Is(
		err,
		service.ErrTeamSlugExists,
	):
		response.Error(
			c,
			http.StatusConflict,
			"Team slug already exists",
			nil,
		)

	case errors.Is(
		err,
		service.ErrInvalidTeamName,
	):
		response.Error(
			c,
			http.StatusBadRequest,
			"Team name is invalid",
			nil,
		)

	case errors.Is(
		err,
		service.ErrInvalidTeamPosition,
	):
		response.Error(
			c,
			http.StatusBadRequest,
			"Team position is invalid",
			nil,
		)

	case errors.Is(
		err,
		service.ErrInvalidTeamPracticeArea,
	):
		response.Error(
			c,
			http.StatusBadRequest,
			"One or more practice areas are invalid or inactive",
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