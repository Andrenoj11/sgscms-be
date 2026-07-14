package router

import (
	"database/sql"
	"net/http"

	"github.com/Andrenoj11/sgscms-be/internal/config"
	"github.com/Andrenoj11/sgscms-be/internal/handler"
	"github.com/Andrenoj11/sgscms-be/internal/middleware"
	"github.com/Andrenoj11/sgscms-be/internal/repository"
	"github.com/Andrenoj11/sgscms-be/internal/response"
	"github.com/Andrenoj11/sgscms-be/internal/security"
	"github.com/Andrenoj11/sgscms-be/internal/service"
	"github.com/gin-gonic/gin"
)

func New(
	cfg *config.Config,
	db *sql.DB,
) *gin.Engine {
	if !cfg.App.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.HandleMethodNotAllowed = true

	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	/*
		Repository
	*/

	adminRepository :=
		repository.NewPostgresAdminRepository(db)

	practiceAreaRepository :=
		repository.NewPostgresPracticeAreaRepository(db)

	teamRepository :=
		repository.NewPostgresTeamRepository(db)

	/*
		Security
	*/

	passwordHasher := security.NewPasswordHasher()
	jwtManager := security.NewJWTManager(cfg.JWT)

	/*
		Service
	*/

	authService := service.NewAuthService(
		adminRepository,
		passwordHasher,
		jwtManager,
	)

	practiceAreaService :=
		service.NewPracticeAreaService(
			practiceAreaRepository,
		)

	teamService := service.NewTeamService(
		teamRepository,
		practiceAreaRepository,
	)

	/*
		Handler
	*/

	healthHandler := handler.NewHealthHandler(
		cfg.App.Name,
		cfg.App.Env,
		db,
	)

	authHandler := handler.NewAuthHandler(
		authService,
	)

	practiceAreaHandler :=
		handler.NewPracticeAreaHandler(
			practiceAreaService,
		)

	teamHandler := handler.NewTeamHandler(
		teamService,
	)

	/*
		Middleware
	*/

	authMiddleware :=
		middleware.NewAuthMiddleware(
			jwtManager,
			adminRepository,
		)

	/*
		General routes
	*/

	router.GET(
		"/health",
		healthHandler.Check,
	)

	/*
		API version 1
	*/

	apiV1 := router.Group("/api/v1")

	/*
		Admin routes
	*/

	adminAPI := apiV1.Group("/admin")

	authAPI := adminAPI.Group("/auth")
	{
		authAPI.POST(
			"/login",
			authHandler.Login,
		)
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

		practiceAreaAPI :=
			protectedAdminAPI.Group(
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

		teamAPI := protectedAdminAPI.Group(
			"/teams",
		)

		{
			teamAPI.GET(
				"",
				teamHandler.List,
			)

			teamAPI.POST(
				"",
				teamHandler.Create,
			)

			teamAPI.GET(
				"/:id",
				teamHandler.GetByID,
			)

			teamAPI.PUT(
				"/:id",
				teamHandler.Update,
			)

			teamAPI.DELETE(
				"/:id",
				teamHandler.Delete,
			)
		}
	}

	router.NoRoute(func(c *gin.Context) {
		response.Error(
			c,
			http.StatusNotFound,
			"Endpoint not found",
			nil,
		)
	})

	router.NoMethod(func(c *gin.Context) {
		response.Error(
			c,
			http.StatusMethodNotAllowed,
			"HTTP method not allowed",
			nil,
		)
	})

	return router
}