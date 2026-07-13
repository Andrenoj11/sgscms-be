package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Environment, HTTPAddr, DatabaseURL, CookieName, PublicBaseURL, MediaDir string
	S3Endpoint, S3Bucket, S3AccessKey, S3SecretKey, S3Region, S3PublicURL   string
	APIClientEncryptionKey                                                  string
	SessionTTL, SignatureTolerance                                          time.Duration
	CookieSecure                                                            bool
	S3UseSSL                                                                bool
	MaxUploadBytes                                                          int64
	MaxImageWidth, MaxImageHeight                                           int
}

func Load() (Config, error) {
	c := Config{
		Environment: env("APP_ENV", "development"), HTTPAddr: env("HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"), CookieName: env("SESSION_COOKIE_NAME", "sgscms_session"),
		PublicBaseURL: env("PUBLIC_BASE_URL", "http://localhost:8080"), MediaDir: env("MEDIA_DIR", "./var/media"),
		S3Endpoint: os.Getenv("S3_ENDPOINT"), S3Bucket: env("S3_BUCKET", "sgscms-media"), S3AccessKey: os.Getenv("S3_ACCESS_KEY"), S3SecretKey: os.Getenv("S3_SECRET_KEY"), S3Region: env("S3_REGION", "ap-southeast-3"), S3PublicURL: os.Getenv("S3_PUBLIC_URL"),
		APIClientEncryptionKey: env("API_CLIENT_ENCRYPTION_KEY", "development-only-change-me"),
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	var err error
	if c.SessionTTL, err = time.ParseDuration(env("SESSION_TTL", "12h")); err != nil {
		return c, fmt.Errorf("SESSION_TTL: %w", err)
	}
	if c.SignatureTolerance, err = time.ParseDuration(env("SIGNATURE_TOLERANCE", "5m")); err != nil {
		return c, fmt.Errorf("SIGNATURE_TOLERANCE: %w", err)
	}
	c.CookieSecure, err = strconv.ParseBool(env("SESSION_SECURE", "true"))
	if err != nil {
		return c, fmt.Errorf("SESSION_SECURE: %w", err)
	}
	c.S3UseSSL, err = strconv.ParseBool(env("S3_USE_SSL", "true"))
	if err != nil {
		return c, fmt.Errorf("S3_USE_SSL: %w", err)
	}
	c.MaxUploadBytes, err = strconv.ParseInt(env("MAX_UPLOAD_BYTES", "10485760"), 10, 64)
	if err != nil {
		return c, err
	}
	c.MaxImageWidth, _ = strconv.Atoi(env("MAX_IMAGE_WIDTH", "8000"))
	c.MaxImageHeight, _ = strconv.Atoi(env("MAX_IMAGE_HEIGHT", "8000"))
	return c, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
