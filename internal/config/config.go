package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	SeedAdmin SeedAdminConfig
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

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		fmt.Println("warning: .env file not found, using system environment variables")
	}

	debug, err := strconv.ParseBool(getEnv("APP_DEBUG", "false"))
	if err != nil {
		return nil, fmt.Errorf("invalid APP_DEBUG value: %w", err)
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
	}

	if cfg.Database.Password == "" {
		return nil, fmt.Errorf("DB_PASSWORD is required")
	}

	return cfg, nil
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}