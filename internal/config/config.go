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
	Server    ServerConfig
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

type ServerConfig struct {
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	MaxJSONBodySize   int64
	TrustedProxies    []string
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

	accessTTL, err := parseDurationEnv(
		"JWT_ACCESS_TTL",
		"15m",
	)
	if err != nil {
		return nil, err
	}

	refreshTTL, err := parseDurationEnv(
		"JWT_REFRESH_TTL",
		"168h",
	)
	if err != nil {
		return nil, err
	}

	signatureMaxAge, err := parseDurationEnv(
		"SIGNATURE_MAX_AGE",
		"5m",
	)
	if err != nil {
		return nil, err
	}

	loginRateWindow, err := parseDurationEnv(
		"LOGIN_RATE_WINDOW",
		"1m",
	)
	if err != nil {
		return nil, err
	}

	readTimeout, err := parseDurationEnv(
		"SERVER_READ_TIMEOUT",
		"15s",
	)
	if err != nil {
		return nil, err
	}

	readHeaderTimeout, err := parseDurationEnv(
		"SERVER_READ_HEADER_TIMEOUT",
		"5s",
	)
	if err != nil {
		return nil, err
	}

	writeTimeout, err := parseDurationEnv(
		"SERVER_WRITE_TIMEOUT",
		"30s",
	)
	if err != nil {
		return nil, err
	}

	idleTimeout, err := parseDurationEnv(
		"SERVER_IDLE_TIMEOUT",
		"60s",
	)
	if err != nil {
		return nil, err
	}

	shutdownTimeout, err := parseDurationEnv(
		"SERVER_SHUTDOWN_TIMEOUT",
		"10s",
	)
	if err != nil {
		return nil, err
	}

	loginRateLimit, err := parsePositiveIntEnv(
		"LOGIN_RATE_LIMIT",
		"5",
	)
	if err != nil {
		return nil, err
	}

	maxImageSize, err := parsePositiveInt64Env(
		"UPLOAD_MAX_IMAGE_SIZE",
		"2097152",
	)
	if err != nil {
		return nil, err
	}

	maxJSONBodySize, err := parsePositiveInt64Env(
		"SERVER_MAX_JSON_BODY_SIZE",
		"1048576",
	)
	if err != nil {
		return nil, err
	}

	encryptionKey, err :=
		base64.StdEncoding.DecodeString(
			getEnv(
				"SIGNATURE_ENCRYPTION_KEY",
				"",
			),
		)
	if err != nil ||
		len(encryptionKey) != 32 {
		return nil, fmt.Errorf(
			"SIGNATURE_ENCRYPTION_KEY must be base64 encoded 32 bytes",
		)
	}

	cfg := &Config{
		App: AppConfig{
			Name: getEnv(
				"APP_NAME",
				"SGS CMS API",
			),
			Env: getEnv(
				"APP_ENV",
				"development",
			),
			Port: getEnv(
				"APP_PORT",
				"8080",
			),
			Debug: debug,
		},
		Server: ServerConfig{
			ReadTimeout:       readTimeout,
			ReadHeaderTimeout: readHeaderTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
			ShutdownTimeout:   shutdownTimeout,
			MaxJSONBodySize:   maxJSONBodySize,
			TrustedProxies: parseCSV(
				getEnv(
					"TRUSTED_PROXIES",
					"",
				),
			),
		},
		Database: DatabaseConfig{
			Host: getEnv(
				"DB_HOST",
				"localhost",
			),
			Port: getEnv(
				"DB_PORT",
				"5432",
			),
			Name: getEnv(
				"DB_NAME",
				"sgscms",
			),
			User: getEnv(
				"DB_USER",
				"sgscms",
			),
			Password: getEnv(
				"DB_PASSWORD",
				"",
			),
			SSLMode: getEnv(
				"DB_SSL_MODE",
				"disable",
			),
			Timezone: getEnv(
				"DB_TIMEZONE",
				"Asia/Jakarta",
			),
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

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validate(
	cfg *Config,
) error {
	if cfg.Database.Password == "" {
		return fmt.Errorf(
			"DB_PASSWORD is required",
		)
	}

	if len(cfg.JWT.Secret) < 32 {
		return fmt.Errorf(
			"JWT_SECRET must contain at least 32 characters",
		)
	}

	if cfg.JWT.RefreshTTL <=
		cfg.JWT.AccessTTL {
		return fmt.Errorf(
			"JWT_REFRESH_TTL must be longer than JWT_ACCESS_TTL",
		)
	}

	if cfg.Storage.Directory == "" {
		return fmt.Errorf(
			"UPLOAD_DIRECTORY is required",
		)
	}

	if cfg.Storage.BaseURL == "" {
		return fmt.Errorf(
			"UPLOAD_BASE_URL is required",
		)
	}

	if len(
		cfg.Security.CORSAllowedOrigins,
	) == 0 {
		return fmt.Errorf(
			"CORS_ALLOWED_ORIGINS must contain at least one origin",
		)
	}

	if cfg.App.Env == "production" {
		if cfg.App.Debug {
			return fmt.Errorf(
				"APP_DEBUG must be false in production",
			)
		}

		if cfg.Database.SSLMode ==
			"disable" {
			return fmt.Errorf(
				"DB_SSL_MODE must not be disable in production",
			)
		}

		if strings.Contains(
			cfg.Storage.BaseURL,
			"localhost",
		) {
			return fmt.Errorf(
				"UPLOAD_BASE_URL must not use localhost in production",
			)
		}

		for _, origin :=
			range cfg.Security.CORSAllowedOrigins {
			if origin == "*" {
				return fmt.Errorf(
					"wildcard CORS origin is not allowed in production",
				)
			}
		}
	}

	return nil
}

func parseDurationEnv(
	key string,
	fallback string,
) (time.Duration, error) {
	value := getEnv(key, fallback)

	duration, err := time.ParseDuration(
		value,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"invalid %s value: %w",
			key,
			err,
		)
	}

	if duration <= 0 {
		return 0, fmt.Errorf(
			"%s must be greater than zero",
			key,
		)
	}

	return duration, nil
}

func parsePositiveIntEnv(
	key string,
	fallback string,
) (int, error) {
	value, err := strconv.Atoi(
		getEnv(key, fallback),
	)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf(
			"%s must be greater than zero",
			key,
		)
	}

	return value, nil
}

func parsePositiveInt64Env(
	key string,
	fallback string,
) (int64, error) {
	value, err := strconv.ParseInt(
		getEnv(key, fallback),
		10,
		64,
	)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf(
			"%s must be greater than zero",
			key,
		)
	}

	return value, nil
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

func parseCSV(
	value string,
) []string {
	parts := strings.Split(
		value,
		",",
	)

	result := make(
		[]string,
		0,
		len(parts),
	)

	for _, part := range parts {
		part = strings.TrimSpace(part)

		if part != "" {
			result = append(
				result,
				part,
			)
		}
	}

	return result
}