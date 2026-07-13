package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/Andrenoj11/sgscms-be/internal/response"
	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	appName string
	appEnv  string
	db      *sql.DB
}

func NewHealthHandler(
	appName string,
	appEnv string,
	db *sql.DB,
) *HealthHandler {
	return &HealthHandler{
		appName: appName,
		appEnv:  appEnv,
		db:      db,
	}
}

func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		2*time.Second,
	)
	defer cancel()

	databaseStatus := "healthy"
	httpStatus := http.StatusOK
	message := "SGS CMS API is running"

	if err := h.db.PingContext(ctx); err != nil {
		databaseStatus = "unhealthy"
		httpStatus = http.StatusServiceUnavailable
		message = "SGS CMS API is unavailable"
	}

	data := gin.H{
		"application": h.appName,
		"environment": h.appEnv,
		"status": func() string {
			if databaseStatus == "healthy" {
				return "healthy"
			}

			return "unhealthy"
		}(),
		"checks": gin.H{
			"database": databaseStatus,
		},
		"timestamp": time.Now().UTC(),
	}

	if httpStatus != http.StatusOK {
		response.Error(
			c,
			httpStatus,
			message,
			data,
		)
		return
	}

	response.Success(
		c,
		httpStatus,
		message,
		data,
	)
}