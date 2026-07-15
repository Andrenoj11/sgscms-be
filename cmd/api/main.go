package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Andrenoj11/sgscms-be/internal/config"
	"github.com/Andrenoj11/sgscms-be/internal/database"
	"github.com/Andrenoj11/sgscms-be/internal/router"
	"github.com/Andrenoj11/sgscms-be/internal/server"
)

// @title SGS CMS API
// @version 1.0
// @description Backend CMS API untuk SGS Law Firm.
// @description
// @description Admin API menggunakan JWT Bearer Token.
// @description Sebagian besar admin endpoint juga membutuhkan X-Timestamp, X-Nonce, dan X-Signature.
// @description Upload gambar hanya membutuhkan JWT karena menggunakan multipart/form-data.
// @description
// @description Format canonical X-Signature:
// @description METHOD + newline + REQUEST_URI + newline + TIMESTAMP + newline + NONCE + newline + SHA256_BODY
//
// @contact.name SGS CMS Development Team
//
// @license.name Private
//
// @host localhost:8080
// @BasePath /api/v1
//
// @schemes http https
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Masukkan token dengan format: Bearer {access_token}
//
// @securityDefinitions.apikey XSignature
// @in header
// @name X-Signature
// @description HMAC-SHA256 signature dari canonical request.
//
// @securityDefinitions.apikey XTimestamp
// @in header
// @name X-Timestamp
// @description Unix timestamp dalam detik.
//
// @securityDefinitions.apikey XNonce
// @in header
// @name X-Nonce
// @description Nilai unik untuk mencegah replay attack.

func main() {
	/*
		|--------------------------------------------------------------------------
		| Load Configuration
		|--------------------------------------------------------------------------
	*/

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf(
			"failed to load configuration: %v",
			err,
		)
	}

	/*
		|--------------------------------------------------------------------------
		| Database Connection
		|--------------------------------------------------------------------------
	*/

	db, err := database.NewPostgresConnection(
		cfg.Database,
	)
	if err != nil {
		log.Fatalf(
			"failed to connect to database: %v",
			err,
		)
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Printf(
				"failed to close database connection: %v",
				err,
			)
		}
	}()

	log.Println(
		"database connection established",
	)

	/*
		|--------------------------------------------------------------------------
		| Application Router
		|--------------------------------------------------------------------------
	*/

	appRouter := router.New(
		cfg,
		db,
	)

	/*
		|--------------------------------------------------------------------------
		| Application Context
		|--------------------------------------------------------------------------
		|
		| Context akan dibatalkan ketika aplikasi menerima:
		|
		| SIGINT  → Ctrl + C
		| SIGTERM → sinyal penghentian dari Docker, Kubernetes, atau server
		|
	*/

	applicationContext, stop :=
		signal.NotifyContext(
			context.Background(),
			os.Interrupt,
			syscall.SIGTERM,
		)

	defer stop()

	/*
		|--------------------------------------------------------------------------
		| HTTP Server
		|--------------------------------------------------------------------------
	*/

	httpServer := server.NewHTTPServer(
		cfg,
		appRouter,
	)

	log.Printf(
		"%s starting in %s mode",
		cfg.App.Name,
		cfg.App.Env,
	)

	if err := httpServer.Run(
		applicationContext,
	); err != nil {
		log.Fatalf(
			"HTTP server stopped with error: %v",
			err,
		)
	}

	log.Println(
		"application stopped successfully",
	)
}