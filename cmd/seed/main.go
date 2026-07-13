package main

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/Andrenoj11/sgscms-be/internal/config"
	"github.com/Andrenoj11/sgscms-be/internal/database"
	"github.com/Andrenoj11/sgscms-be/internal/domain"
	"github.com/Andrenoj11/sgscms-be/internal/repository"
	"github.com/Andrenoj11/sgscms-be/internal/security"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	if strings.TrimSpace(cfg.SeedAdmin.Email) == "" {
		log.Fatal("SEED_ADMIN_EMAIL is required")
	}

	if len(cfg.SeedAdmin.Password) < 8 {
		log.Fatal("SEED_ADMIN_PASSWORD must contain at least 8 characters")
	}

	db, err := database.NewPostgresConnection(cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	adminRepository := repository.NewPostgresAdminRepository(db)
	passwordHasher := security.NewPasswordHasher()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	existingAdmin, err := adminRepository.FindByEmail(
		ctx,
		cfg.SeedAdmin.Email,
	)

	if err == nil && existingAdmin != nil {
		log.Printf(
			"admin with email %s already exists",
			cfg.SeedAdmin.Email,
		)
		return
	}

	if !errors.Is(err, repository.ErrAdminNotFound) {
		log.Fatalf("failed to check existing admin: %v", err)
	}

	passwordHash, err := passwordHasher.Hash(
		cfg.SeedAdmin.Password,
	)
	if err != nil {
		log.Fatalf("failed to hash admin password: %v", err)
	}

	admin := &domain.Admin{
		Name:         strings.TrimSpace(cfg.SeedAdmin.Name),
		Email:        strings.ToLower(strings.TrimSpace(cfg.SeedAdmin.Email)),
		PasswordHash: passwordHash,
		Role:         domain.AdminRoleSuperAdmin,
		IsActive:     true,
	}

	if err := adminRepository.Create(ctx, admin); err != nil {
		log.Fatalf("failed to create admin: %v", err)
	}

	log.Printf(
		"super admin created successfully: %s",
		admin.Email,
	)
}