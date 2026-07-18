package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/Andrenoj11/sgscms-be/docs"
	"github.com/Andrenoj11/sgscms-be/internal/config"
	"github.com/Andrenoj11/sgscms-be/internal/database"
	"github.com/Andrenoj11/sgscms-be/internal/router"
)

var (
	initOnce sync.Once
	app      http.Handler
	appDB    *sql.DB
	initErr  error
)

// Handler adapts the existing Gin router to the Vercel Go runtime.
func Handler(w http.ResponseWriter, r *http.Request) {
	initOnce.Do(initialize)
	if initErr != nil {
		log.Printf("failed to initialize serverless application: %v", initErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Application initialization failed",
		})
		return
	}

	app.ServeHTTP(w, r)
}

func initialize() {
	cfg, err := config.Load()
	if err != nil {
		initErr = err
		return
	}

	// Empty host and schemes make Swagger UI use the current Vercel domain and protocol.
	docs.SwaggerInfo.Host = ""
	docs.SwaggerInfo.Schemes = nil

	appDB, err = database.NewPostgresConnection(cfg.Database)
	if err != nil {
		initErr = err
		return
	}

	app = router.New(cfg, appDB)
}
