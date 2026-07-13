package router

import (
	"database/sql"
	"net/http"

	"github.com/Andrenoj11/sgscms-be/internal/config"
	"github.com/Andrenoj11/sgscms-be/internal/handler"
	"github.com/Andrenoj11/sgscms-be/internal/response"
	"github.com/gin-gonic/gin"
)

func New(
	cfg *config.Config,
	db *sql.DB,
) *gin.Engine {
	if !cfg.App.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.HandleMethodNotAllowed = true

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	healthHandler := handler.NewHealthHandler(
		cfg.App.Name,
		cfg.App.Env,
		db,
	)

	r.GET("/health", healthHandler.Check)

	r.NoRoute(func(c *gin.Context) {
		response.Error(
			c,
			http.StatusNotFound,
			"Endpoint not found",
			nil,
		)
	})

	r.NoMethod(func(c *gin.Context) {
		response.Error(
			c,
			http.StatusMethodNotAllowed,
			"HTTP method not allowed",
			nil,
		)
	})

	return r
}