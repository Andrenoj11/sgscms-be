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

	newsRepository :=
		repository.NewPostgresNewsRepository(db)

	publicRepository :=
		repository.NewPostgresPublicRepository(db)

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

	newsService := service.NewNewsService(
		newsRepository,
	)

	publicService := service.NewPublicService(
		publicRepository,
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

	newsHandler := handler.NewNewsHandler(
		newsService,
	)

	publicHandler := handler.NewPublicHandler(
		publicService,
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
		Public routes
	*/

	publicAPI := apiV1.Group("/public")
	{
		publicAPI.GET(
			"/practice-areas",
			publicHandler.ListPracticeAreas,
		)

		publicAPI.GET(
			"/teams",
			publicHandler.ListTeams,
		)

		publicAPI.GET(
			"/teams/:slug",
			publicHandler.GetTeamBySlug,
		)

		publicAPI.GET(
			"/news",
			publicHandler.ListNews,
		)

		publicAPI.GET(
			"/news/:slug",
			publicHandler.GetNewsBySlug,
		)
	}

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

		newsAPI := protectedAdminAPI.Group(
			"/news",
		)

		{
			newsAPI.GET(
				"",
				newsHandler.List,
			)

			newsAPI.POST(
				"",
				newsHandler.Create,
			)

			newsAPI.GET(
				"/:id",
				newsHandler.GetByID,
			)

			newsAPI.PUT(
				"/:id",
				newsHandler.Update,
			)

			newsAPI.DELETE(
				"/:id",
				newsHandler.Delete,
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