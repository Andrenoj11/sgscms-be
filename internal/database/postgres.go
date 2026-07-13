package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Andrenoj11/sgscms-be/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewPostgresConnection(
	cfg config.DatabaseConfig,
) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Name,
		cfg.SSLMode,
		cfg.Timezone,
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL connection: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()

		return nil, fmt.Errorf("ping PostgreSQL database: %w", err)
	}

	return db, nil
}