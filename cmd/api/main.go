package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Andrenoj11/sgscms-be/internal/config"
	"github.com/Andrenoj11/sgscms-be/internal/database"
	"github.com/Andrenoj11/sgscms-be/internal/router"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	db, err := database.NewPostgresConnection(cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("failed to close database connection: %v", err)
		}
	}()

	log.Println("database connection established")

	appRouter := router.New(cfg, db)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.App.Port),
		Handler:           appRouter,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf(
			"%s running on http://localhost:%s in %s mode",
			cfg.App.Name,
			cfg.App.Port,
			cfg.App.Env,
		)

		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed to start HTTP server: %v", err)
		}
	}()

	shutdownSignal := make(chan os.Signal, 1)

	signal.Notify(
		shutdownSignal,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	<-shutdownSignal

	log.Println("shutting down server...")

	shutdownContext, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		log.Printf("server shutdown failed: %v", err)
		return
	}

	log.Println("server stopped gracefully")
}