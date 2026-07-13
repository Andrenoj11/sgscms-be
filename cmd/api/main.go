package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sgs-law-firm/cms/internal/application"
	"github.com/sgs-law-firm/cms/internal/config"
	"github.com/sgs-law-firm/cms/internal/delivery/httpapi"
	"github.com/sgs-law-firm/cms/internal/domain"
	"github.com/sgs-law-firm/cms/internal/infrastructure/postgres"
	"github.com/sgs-law-firm/cms/internal/infrastructure/storage"
	"github.com/sgs-law-firm/cms/migrations"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, e := config.Load()
	if e != nil {
		log.Error("configuration failed", "error", e)
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	if os.Getenv("AUTO_MIGRATE") != "false" {
		if e = migrations.Up(ctx, cfg.DatabaseURL); e != nil {
			log.Error("migration failed", "error", e)
			os.Exit(1)
		}
	}
	store, e := postgres.New(ctx, cfg.DatabaseURL, cfg.APIClientEncryptionKey)
	if e != nil {
		log.Error("database connection failed", "error", e)
		os.Exit(1)
	}
	defer store.Close()
	if email, password := os.Getenv("BOOTSTRAP_ADMIN_EMAIL"), os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"); email != "" && password != "" {
		hash, he := application.HashPassword(password)
		if he != nil {
			log.Error("invalid bootstrap password", "error", he)
			os.Exit(1)
		}
		if e = store.BootstrapAdmin(ctx, email, "CMS Administrator", hash); e != nil {
			log.Error("bootstrap failed", "error", e)
			os.Exit(1)
		}
	}
	var objects domain.ObjectStorage = storage.Local{Dir: cfg.MediaDir, BaseURL: cfg.PublicBaseURL}
	if cfg.S3Endpoint != "" {
		objects, e = storage.NewS3(ctx, cfg.S3Endpoint, cfg.S3Bucket, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Region, cfg.S3PublicURL, cfg.S3UseSSL)
		if e != nil {
			log.Error("object storage connection failed", "error", e)
			os.Exit(1)
		}
	}
	handler := httpapi.New(cfg, store, objects, log)
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	go scheduler(ctx, store, log)
	go func() {
		log.Info("server started", "addr", cfg.HTTPAddr, "environment", cfg.Environment)
		if e := server.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			log.Error("server failed", "error", e)
			cancel()
		}
	}()
	<-ctx.Done()
	shutdown, stop := context.WithTimeout(context.Background(), 10*time.Second)
	defer stop()
	_ = server.Shutdown(shutdown)
}

type publisher interface {
	PublishDueNews(context.Context, time.Time) (int64, error)
}

func scheduler(ctx context.Context, p publisher, log *slog.Logger) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			n, e := p.PublishDueNews(ctx, now.UTC())
			if e != nil {
				log.Error("scheduled publication failed", "error", e)
			} else if n > 0 {
				log.Info("scheduled news published", "count", n)
			}
		case <-ctx.Done():
			return
		}
	}
}
