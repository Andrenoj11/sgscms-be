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

	"github.com/Andrenoj11/sgscms-be/internal/middleware"
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
adminRepository :=
	repository.NewPostgresAdminRepository(db)

passwordHasher := security.NewPasswordHasher()
jwtManager := security.NewJWTManager(cfg.JWT)

authService := service.NewAuthService(
	adminRepository,
	passwordHasher,
	jwtManager,
)

authHandler := handler.NewAuthHandler(authService)

authMiddleware := middleware.NewAuthMiddleware(
	jwtManager,
	adminRepository,
)
practiceAreaRepository :=
	repository.NewPostgresPracticeAreaRepository(db)

practiceAreaService :=
	service.NewPracticeAreaService(
		practiceAreaRepository,
	)

practiceAreaHandler :=
	handler.NewPracticeAreaHandler(
		practiceAreaService,
	)

r.GET("/health", healthHandler.Check)

apiV1 := r.Group("/api/v1")

adminAPI := apiV1.Group("/admin")

authAPI := adminAPI.Group("/auth")
{
	authAPI.POST("/login", authHandler.Login)
}

protectedAdminAPI := adminAPI.Group("")
protectedAdminAPI.Use(
	authMiddleware.Authenticate(),
)
{
	protectedAdminAPI.GET(
		"/auth/me",
		authHandler.Me,
	)
}

practiceAreaAPI := protectedAdminAPI.Group(
	"/practice-areas",
)
{
	practiceAreaAPI.GET(
		"",
		practiceAreaHandler.List,
	)

	practiceAreaAPI.POST(
		"",
		practiceAreaHandler.Create,
	)

	practiceAreaAPI.GET(
		"/:id",
		practiceAreaHandler.GetByID,
	)

	practiceAreaAPI.PUT(
		"/:id",
		practiceAreaHandler.Update,
	)

	practiceAreaAPI.DELETE(
		"/:id",
		practiceAreaHandler.Delete,
	)
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