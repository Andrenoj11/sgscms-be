package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App       AppConfig
	Database  DatabaseConfig
	SeedAdmin SeedAdminConfig
	JWT       JWTConfig
	Storage   StorageConfig
	Security  SecurityConfig
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
	Secret     string
	Issuer     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type StorageConfig struct {
	Directory    string
	BaseURL      string
	MaxImageSize int64
}

type SecurityConfig struct {
	SignatureEncryptionKey []byte
	SignatureMaxAge         time.Duration
	CORSAllowedOrigins      []string
	LoginRateLimit          int
	LoginRateWindow         time.Duration
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
			"invalid JWT_ACCESS_TTL: %w",
			err,
		)
	}

	refreshTTL, err := time.ParseDuration(
		getEnv("JWT_REFRESH_TTL", "168h"),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid JWT_REFRESH_TTL: %w",
			err,
		)
	}

	signatureMaxAge, err := time.ParseDuration(
		getEnv("SIGNATURE_MAX_AGE", "5m"),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid SIGNATURE_MAX_AGE: %w",
			err,
		)
	}

	loginRateLimit, err := strconv.Atoi(
		getEnv("LOGIN_RATE_LIMIT", "5"),
	)
	if err != nil || loginRateLimit < 1 {
		return nil, fmt.Errorf(
			"LOGIN_RATE_LIMIT must be greater than zero",
		)
	}

	loginRateWindow, err := time.ParseDuration(
		getEnv("LOGIN_RATE_WINDOW", "1m"),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid LOGIN_RATE_WINDOW: %w",
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
			"invalid UPLOAD_MAX_IMAGE_SIZE: %w",
			err,
		)
	}

	encryptionKey, err := base64.StdEncoding.DecodeString(
		getEnv("SIGNATURE_ENCRYPTION_KEY", ""),
	)
	if err != nil || len(encryptionKey) != 32 {
		return nil, fmt.Errorf(
			"SIGNATURE_ENCRYPTION_KEY must be base64 encoded 32 bytes",
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
			Name:     getEnv("SEED_ADMIN_NAME", "Super Admin"),
			Email:    getEnv("SEED_ADMIN_EMAIL", ""),
			Password: getEnv("SEED_ADMIN_PASSWORD", ""),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", ""),
			Issuer:     getEnv("JWT_ISSUER", "sgscms-api"),
			AccessTTL:  accessTTL,
			RefreshTTL: refreshTTL,
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
		Security: SecurityConfig{
			SignatureEncryptionKey: encryptionKey,
			SignatureMaxAge:         signatureMaxAge,
			CORSAllowedOrigins: parseCSV(
				getEnv(
					"CORS_ALLOWED_ORIGINS",
					"http://localhost:3000,http://localhost:5173",
				),
			),
			LoginRateLimit:  loginRateLimit,
			LoginRateWindow: loginRateWindow,
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

	if cfg.JWT.RefreshTTL <= cfg.JWT.AccessTTL {
		return nil, fmt.Errorf(
			"JWT_REFRESH_TTL must be longer than JWT_ACCESS_TTL",
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

func parseCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)

		if part != "" {
			result = append(result, part)
		}
	}

	return result
}