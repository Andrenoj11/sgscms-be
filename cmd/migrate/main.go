package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"

	"github.com/Andrenoj11/sgscms-be/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	direction := flag.String(
		"direction",
		"up",
		"migration direction: up or down",
	)

	steps := flag.Int(
		"steps",
		0,
		"number of migration steps, zero means all for up",
	)

	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	databaseURL := buildDatabaseURL(cfg)

	migration, err := migrate.New(
		"file://migrations",
		databaseURL,
	)
	if err != nil {
		log.Fatalf("failed to create migration instance: %v", err)
	}

	defer func() {
		sourceErr, databaseErr := migration.Close()

		if sourceErr != nil {
			log.Printf("failed to close migration source: %v", sourceErr)
		}

		if databaseErr != nil {
			log.Printf("failed to close migration database: %v", databaseErr)
		}
	}()

	switch *direction {
	case "up":
		err = runUp(migration, *steps)
	case "down":
		err = runDown(migration, *steps)
	default:
		log.Fatalf(
			"invalid migration direction %q: use up or down",
			*direction,
		)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		log.Println("no migration changes")
		return
	}

	if err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	log.Printf(
		"migration %s completed successfully",
		*direction,
	)
}

func runUp(
	migration *migrate.Migrate,
	steps int,
) error {
	if steps > 0 {
		return migration.Steps(steps)
	}

	return migration.Up()
}

func runDown(
	migration *migrate.Migrate,
	steps int,
) error {
	if steps <= 0 {
		return fmt.Errorf(
			"down migration requires steps greater than zero",
		)
	}

	return migration.Steps(-steps)
}

func buildDatabaseURL(
	cfg *config.Config,
) string {
	userInfo := url.UserPassword(
		cfg.Database.User,
		cfg.Database.Password,
	)

	databaseURL := url.URL{
		Scheme: "postgres",
		User:   userInfo,
		Host: fmt.Sprintf(
			"%s:%s",
			cfg.Database.Host,
			cfg.Database.Port,
		),
		Path: cfg.Database.Name,
	}

	query := databaseURL.Query()
	query.Set("sslmode", cfg.Database.SSLMode)
	databaseURL.RawQuery = query.Encode()

	return databaseURL.String()
}