package router

import (
	"database/sql"
	"net/http"

	"github.com/Andrenoj11/sgscms-be/internal/config"
	"github.com/Andrenoj11/sgscms-be/internal/handler"
	"github.com/Andrenoj11/sgscms-be/internal/response"
	"github.com/gin-gonic/gin"

	"github.com/Andrenoj11/sgscms-be/internal/repository"
	"github.com/Andrenoj11/sgscms-be/internal/security"
	"github.com/Andrenoj11/sgscms-be/internal/service"
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

adminRepository := repository.NewPostgresAdminRepository(db)
passwordHasher := security.NewPasswordHasher()

authService := service.NewAuthService(
	adminRepository,
	passwordHasher,
)

authHandler := handler.NewAuthHandler(authService)

r.GET("/health", healthHandler.Check)

apiV1 := r.Group("/api/v1")

adminAPI := apiV1.Group("/admin")

authAPI := adminAPI.Group("/auth")
{
	authAPI.POST("/login", authHandler.Login)
}

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