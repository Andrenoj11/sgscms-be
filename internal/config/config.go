package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App       AppConfig
	Database  DatabaseConfig
	SeedAdmin SeedAdminConfig
	JWT       JWTConfig
	Storage   StorageConfig
}

type AppConfig struct {
	Name  string
	Env   string
	Port  string
	Debug bool
}

type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
	Timezone string
}

type SeedAdminConfig struct {
	Name     string
	Email    string
	Password string
}

type JWTConfig struct {
	Secret    string
	Issuer    string
	AccessTTL time.Duration
}

type StorageConfig struct {
	Directory    string
	BaseURL      string
	MaxImageSize int64
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		fmt.Println(
			"warning: .env file not found, using system environment variables",
		)
	}

	debug, err := strconv.ParseBool(
		getEnv("APP_DEBUG", "false"),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid APP_DEBUG value: %w",
			err,
		)
	}

	accessTTL, err := time.ParseDuration(
		getEnv("JWT_ACCESS_TTL", "15m"),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid JWT_ACCESS_TTL value: %w",
			err,
		)
	}

	maxImageSize, err := strconv.ParseInt(
		getEnv(
			"UPLOAD_MAX_IMAGE_SIZE",
			"2097152",
		),
		10,
		64,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid UPLOAD_MAX_IMAGE_SIZE value: %w",
			err,
		)
	}

	cfg := &Config{
		App: AppConfig{
			Name:  getEnv("APP_NAME", "SGS CMS API"),
			Env:   getEnv("APP_ENV", "development"),
			Port:  getEnv("APP_PORT", "8080"),
			Debug: debug,
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			Name:     getEnv("DB_NAME", "sgscms"),
			User:     getEnv("DB_USER", "sgscms"),
			Password: getEnv("DB_PASSWORD", ""),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
			Timezone: getEnv("DB_TIMEZONE", "Asia/Jakarta"),
		},
		SeedAdmin: SeedAdminConfig{
			Name: getEnv(
				"SEED_ADMIN_NAME",
				"Super Admin",
			),
			Email: getEnv(
				"SEED_ADMIN_EMAIL",
				"",
			),
			Password: getEnv(
				"SEED_ADMIN_PASSWORD",
				"",
			),
		},
		JWT: JWTConfig{
			Secret: getEnv(
				"JWT_SECRET",
				"",
			),
			Issuer: getEnv(
				"JWT_ISSUER",
				"sgscms-api",
			),
			AccessTTL: accessTTL,
		},
		Storage: StorageConfig{
			Directory: getEnv(
				"UPLOAD_DIRECTORY",
				"uploads",
			),
			BaseURL: getEnv(
				"UPLOAD_BASE_URL",
				"http://localhost:8080/uploads",
			),
			MaxImageSize: maxImageSize,
		},
	}

	if cfg.Database.Password == "" {
		return nil, fmt.Errorf(
			"DB_PASSWORD is required",
		)
	}

	if len(cfg.JWT.Secret) < 32 {
		return nil, fmt.Errorf(
			"JWT_SECRET must contain at least 32 characters",
		)
	}

	if cfg.Storage.Directory == "" {
		return nil, fmt.Errorf(
			"UPLOAD_DIRECTORY is required",
		)
	}

	if cfg.Storage.BaseURL == "" {
		return nil, fmt.Errorf(
			"UPLOAD_BASE_URL is required",
		)
	}

	if cfg.Storage.MaxImageSize <= 0 {
		return nil, fmt.Errorf(
			"UPLOAD_MAX_IMAGE_SIZE must be greater than zero",
		)
	}

	return cfg, nil
}

func getEnv(
	key string,
	fallback string,
) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}