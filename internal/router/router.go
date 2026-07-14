package router

import (
	"database/sql"
	"net/http"
	"os"

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

	/*
		|--------------------------------------------------------------------------
		| Global Middleware
		|--------------------------------------------------------------------------
	*/

	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.Use(
		middleware.CORS(
			cfg.Security.CORSAllowedOrigins,
		),
	)

	/*
		|--------------------------------------------------------------------------
		| Repository
		|--------------------------------------------------------------------------
	*/

	adminRepository :=
		repository.NewPostgresAdminRepository(db)

	adminSessionRepository :=
		repository.NewPostgresAdminSessionRepository(db)

	practiceAreaRepository :=
		repository.NewPostgresPracticeAreaRepository(db)

	teamRepository :=
		repository.NewPostgresTeamRepository(db)

	newsRepository :=
		repository.NewPostgresNewsRepository(db)

	publicRepository :=
		repository.NewPostgresPublicRepository(db)

	/*
		|--------------------------------------------------------------------------
		| Security
		|--------------------------------------------------------------------------
	*/

	passwordHasher := security.NewPasswordHasher()

	jwtManager := security.NewJWTManager(
		cfg.JWT,
	)

	secretCipher, err := security.NewSecretCipher(
		cfg.Security.SignatureEncryptionKey,
	)
	if err != nil {
		panic(
			"failed to initialize signature cipher: " +
				err.Error(),
		)
	}

	/*
		|--------------------------------------------------------------------------
		| Service
		|--------------------------------------------------------------------------
	*/

	authService := service.NewAuthService(
		adminRepository,
		adminSessionRepository,
		passwordHasher,
		jwtManager,
		secretCipher,
		cfg.JWT.RefreshTTL,
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

	uploadService := service.NewUploadService(
		cfg.Storage,
	)

	/*
		|--------------------------------------------------------------------------
		| Handler
		|--------------------------------------------------------------------------
	*/

	healthHandler := handler.NewHealthHandler(
		cfg.App.Name,
		cfg.App.Env,
		db,
	)

	authHandler := handler.NewAuthHandler(
		authService,
		cfg.JWT.RefreshTTL,
		cfg.App.Env == "production",
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

	uploadHandler := handler.NewUploadHandler(
		uploadService,
	)

	/*
		|--------------------------------------------------------------------------
		| Application Middleware
		|--------------------------------------------------------------------------
	*/

	authMiddleware :=
		middleware.NewAuthMiddleware(
			jwtManager,
			adminRepository,
			adminSessionRepository,
		)

	signatureMiddleware :=
		middleware.NewSignatureMiddleware(
			adminSessionRepository,
			secretCipher,
			cfg.Security.SignatureMaxAge,
		)

	loginRateLimiter :=
		middleware.NewRateLimiter(
			cfg.Security.LoginRateLimit,
			cfg.Security.LoginRateWindow,
		)

	/*
		|--------------------------------------------------------------------------
		| Upload Directory
		|--------------------------------------------------------------------------
	*/

	if err := os.MkdirAll(
		cfg.Storage.Directory,
		0o755,
	); err != nil {
		panic(
			"failed to create upload directory: " +
				err.Error(),
		)
	}

	/*
		|--------------------------------------------------------------------------
		| General Routes
		|--------------------------------------------------------------------------
	*/

	router.GET(
		"/health",
		healthHandler.Check,
	)

	router.Static(
		"/uploads",
		cfg.Storage.Directory,
	)

	/*
		|--------------------------------------------------------------------------
		| API Version 1
		|--------------------------------------------------------------------------
	*/

	apiV1 := router.Group("/api/v1")

	/*
		|--------------------------------------------------------------------------
		| Public Routes
		|--------------------------------------------------------------------------
		|
		| Route berikut tidak membutuhkan JWT maupun X-Signature.
		|
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
		|--------------------------------------------------------------------------
		| Admin Routes
		|--------------------------------------------------------------------------
	*/

	adminAPI := apiV1.Group("/admin")

	/*
		|--------------------------------------------------------------------------
		| Public Authentication Routes
		|--------------------------------------------------------------------------
		|
		| Login, refresh, dan logout tidak membutuhkan access token.
		| Refresh token dikirim melalui HttpOnly cookie.
		|
	*/

	authAPI := adminAPI.Group("/auth")
	{
		authAPI.POST(
			"/login",
			loginRateLimiter.LimitByIP(),
			authHandler.Login,
		)

		authAPI.POST(
			"/refresh",
			authHandler.Refresh,
		)

		authAPI.POST(
			"/logout",
			authHandler.Logout,
		)
	}

	/*
		|--------------------------------------------------------------------------
		| JWT Protected Admin Routes
		|--------------------------------------------------------------------------
		|
		| Semua route di grup ini membutuhkan access token.
		|
	*/

	authenticatedAdminAPI :=
		adminAPI.Group("")

	authenticatedAdminAPI.Use(
		authMiddleware.Authenticate(),
	)

	/*
		|--------------------------------------------------------------------------
		| Upload Routes
		|--------------------------------------------------------------------------
		|
		| Upload hanya menggunakan JWT.
		| Multipart body tidak menggunakan X-Signature.
		|
	*/

	uploadAPI :=
		authenticatedAdminAPI.Group(
			"/uploads",
		)

	{
		uploadAPI.POST(
			"/images",
			uploadHandler.UploadImage,
		)
	}

	/*
		|--------------------------------------------------------------------------
		| JWT + X-Signature Protected Routes
		|--------------------------------------------------------------------------
		|
		| Semua route di grup ini membutuhkan:
		|
		| Authorization
		| X-Timestamp
		| X-Nonce
		| X-Signature
		|
	*/

	signedAdminAPI :=
		authenticatedAdminAPI.Group("")

	signedAdminAPI.Use(
		signatureMiddleware.Verify(),
	)

	/*
		|--------------------------------------------------------------------------
		| Current Admin
		|--------------------------------------------------------------------------
	*/

	signedAdminAPI.GET(
		"/auth/me",
		authHandler.Me,
	)

	/*
		|--------------------------------------------------------------------------
		| Practice Area Routes
		|--------------------------------------------------------------------------
	*/

	practiceAreaAPI :=
		signedAdminAPI.Group(
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

	/*
		|--------------------------------------------------------------------------
		| Team Routes
		|--------------------------------------------------------------------------
	*/

	teamAPI :=
		signedAdminAPI.Group(
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

	/*
		|--------------------------------------------------------------------------
		| News Routes
		|--------------------------------------------------------------------------
	*/

	newsAPI :=
		signedAdminAPI.Group(
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

	/*
		|--------------------------------------------------------------------------
		| Route Not Found
		|--------------------------------------------------------------------------
	*/

	router.NoRoute(func(c *gin.Context) {
		response.Error(
			c,
			http.StatusNotFound,
			"Endpoint not found",
			nil,
		)
	})

	/*
		|--------------------------------------------------------------------------
		| HTTP Method Not Allowed
		|--------------------------------------------------------------------------
	*/

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