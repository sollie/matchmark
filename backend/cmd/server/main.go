package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "modernc.org/sqlite"

	"github.com/sollie/matchmark/internal/db"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	database, err := sql.Open("sqlite", cfg.dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	if err := db.Migrate(database); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	if err := db.Seed(database, cfg.adminPassword); err != nil {
		log.Fatalf("seed: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Printf("server listening on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

type config struct {
	dbPath        string
	jwtSecret     string
	adminPassword string
}

func loadConfig() (config, error) {
	var errs []error

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "/data/matchmark.db"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		errs = append(errs, errors.New("JWT_SECRET must be at least 32 characters"))
	}

	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if len(adminPassword) < 16 {
		errs = append(errs, errors.New("ADMIN_PASSWORD must be at least 16 characters"))
	}

	if len(errs) > 0 {
		msg := "invalid configuration:"
		for _, e := range errs {
			msg += "\n  - " + e.Error()
		}
		return config{}, fmt.Errorf("%s", msg)
	}

	return config{
		dbPath:        dbPath,
		jwtSecret:     jwtSecret,
		adminPassword: adminPassword,
	}, nil
}
