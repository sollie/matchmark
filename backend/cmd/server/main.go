package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/time/rate"
	_ "modernc.org/sqlite"

	"github.com/sollie/matchmark/internal/auth"
	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/handler"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
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

	userRepo := repository.NewUserRepository(database)
	competitionRepo := repository.NewCompetitionRepository(database)
	shooterRepo := repository.NewShooterRepository(database)
	stageRepo := repository.NewStageRepository(database)
	hitRepo := repository.NewHitRepository(database)
	lockRepo := repository.NewLockRepository(database)
	assignmentRepo := repository.NewAssignmentRepository(database)
	exportRepo := repository.NewExportRepository(database)

	authSvc := service.NewAuthService(userRepo, database, cfg.jwtSecret)
	userSvc := service.NewUserService(userRepo)
	competitionSvc := service.NewCompetitionService(competitionRepo)
	shooterSvc := service.NewShooterService(shooterRepo, stageRepo, competitionRepo)
	scoringSvc := service.NewScoringService(competitionRepo, shooterRepo, stageRepo, hitRepo, lockRepo, assignmentRepo)
	assignmentSvc := service.NewAssignmentService(assignmentRepo, competitionRepo, shooterRepo)
	monitorSvc := service.NewMonitorService(database, competitionSvc)
	exportSvc := service.NewExportService(exportRepo, competitionRepo)
	spectatorSvc := service.NewSpectatorService(database)
	sseTicketSvc := service.NewSSETicketService(database)

	authH := handler.NewAuthHandler(authSvc)
	userH := handler.NewUserHandler(userSvc)
	competitionH := handler.NewCompetitionHandler(competitionSvc)
	shooterH := handler.NewShooterHandler(shooterSvc)
	scoringH := handler.NewScoringHandler(scoringSvc)
	assignmentH := handler.NewAssignmentHandler(assignmentSvc)
	monitorH := handler.NewMonitorHandler(monitorSvc, sseTicketSvc)
	exportH := handler.NewExportHandler(exportSvc)
	spectatorH := handler.NewSpectatorHandler(spectatorSvc)
	sseTicketH := handler.NewSSETicketHandler(sseTicketSvc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	authRateLimiter := newIPRateLimiter(5, 10)

	r.Route("/api", func(r chi.Router) {
		r.Route("/spectator", func(r chi.Router) {
			r.Get("/competitions", spectatorH.ListCompetitions)
			r.Get("/competitions/{id}/live", spectatorH.Stream)
		})

		r.Route("/auth", func(r chi.Router) {
			r.With(authRateLimiter).Post("/login", authH.Login)
			r.With(authRateLimiter).Post("/refresh", authH.Refresh)
			r.With(auth.RequireAuth(cfg.jwtSecret)).Post("/logout", authH.Logout)
		})

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(cfg.jwtSecret))

			r.Get("/users/me", userH.Me)

			r.Get("/competitions", competitionH.List)
			r.Get("/competitions/{id}", competitionH.Get)

			r.Get("/competitions/{id}/shooters", shooterH.List)

			r.Post("/competitions/{id}/assign", assignmentH.SelectShooter)
			r.Get("/competitions/{id}/assign", assignmentH.GetMyAssignment)
			r.Get("/competitions/{id}/assignments", assignmentH.ListAssignments)

			r.Route("/competitions/{id}/lanes/{lane}/stage/{stage}", func(r chi.Router) {
				r.Get("/", scoringH.GetStage)
				r.Post("/hits", scoringH.SubmitHits)
				r.Post("/lock", scoringH.LockStage)
			})

			r.Group(func(r chi.Router) {
				r.Use(auth.RequireAdmin)
				r.Get("/users", userH.List)
				r.Post("/users", userH.Create)
				r.Put("/users/{id}", userH.Update)
				r.Delete("/users/{id}", userH.Delete)
				r.Post("/users/{id}/reset-password", userH.ResetPassword)

				r.Post("/competitions", competitionH.Create)
				r.Put("/competitions/{id}", competitionH.Update)

				r.Post("/competitions/{id}/shooters", shooterH.Create)
				r.Put("/competitions/{id}/shooters/{sid}", shooterH.Update)
				r.Delete("/competitions/{id}/shooters/{sid}", shooterH.Delete)

				r.Delete("/competitions/{id}/assign", assignmentH.ReleaseAssignment)

				r.Post("/competitions/{id}/lanes/{lane}/stage/{stage}/unlock", scoringH.UnlockStage)

				r.Get("/competitions/{id}/export/json", exportH.ExportJSON)
				r.Get("/competitions/{id}/export/csv", exportH.ExportCSV)
				r.Get("/competitions/{id}/export/pdf", exportH.ExportPDF)

				r.Post("/admin/sse-ticket", sseTicketH.Issue)
			})
		})

		r.Get("/admin/monitor", monitorH.Stream)
	})

	log.Printf("server listening on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

type ipLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipRateLimiter struct {
	mu      sync.Mutex
	entries map[string]*ipLimiterEntry
	r       rate.Limit
	burst   int
}

func newIPRateLimiter(r rate.Limit, burst int) func(http.Handler) http.Handler {
	l := &ipRateLimiter{
		entries: make(map[string]*ipLimiterEntry),
		r:       r,
		burst:   burst,
	}
	go l.cleanup()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limiter := l.getLimiter(r.RemoteAddr)
			if !limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"too many requests"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (l *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry, ok := l.entries[ip]
	if !ok {
		entry = &ipLimiterEntry{limiter: rate.NewLimiter(l.r, l.burst)}
		l.entries[ip] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

func (l *ipRateLimiter) cleanup() {
	for {
		time.Sleep(10 * time.Minute)
		l.mu.Lock()
		for ip, entry := range l.entries {
			if time.Since(entry.lastSeen) > 30*time.Minute {
				delete(l.entries, ip)
			}
		}
		l.mu.Unlock()
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
