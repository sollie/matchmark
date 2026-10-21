package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/go-chi/chi/v5"

	"github.com/sollie/matchmark/internal/service"
)

type Handler struct {
	svc          *service.Service
	auth         *service.Auth
	audit        *service.Audit
	secureCookie bool
	backupDir    string
	limits       streamLimits
	streams      [streamRoles]atomic.Int64
}

func NewRouter(svc *service.Service, opts ...Option) http.Handler {
	h := &Handler{svc: svc, limits: defaultStreamLimits()}
	for _, opt := range opts {
		opt(h)
	}
	r := chi.NewRouter()
	r.Route("/api", func(r chi.Router) {
		if h.auth != nil {
			r.Post("/admin/login", h.adminLogin)
			r.Post("/admin/logout", h.adminLogout)
			r.Post("/admin/password", h.changePassword)
			r.With(h.RequireUser).Get("/admin/session", h.adminSession)
			r.With(h.RequireAdmin).Get("/admin/backup", h.backup)
			r.With(h.RequireAdmin).Get("/audit-log", h.listAudit)
			r.With(h.RequireAdmin).Get("/users", h.listUsers)
			r.With(h.RequireAdmin).Post("/users", h.createUser)
			r.With(h.RequireAdmin).Post("/users/{userID}/deactivate", h.deactivateUser)
			r.With(h.RequireAdmin).Post("/users/{userID}/reactivate", h.reactivateUser)
			r.With(h.RequireAdmin).Post("/users/{userID}/reset-password", h.resetUserPassword)
			r.With(h.RequireAdmin).Put("/users/{userID}/role", h.setUserRole)
			r.With(h.RequireAdmin).Put("/users/{userID}/backup-permission", h.setBackupPermission)
		}
		h.rulebookRoutes(r)
		h.spectatorRoutes(r)
		r.Group(func(r chi.Router) {
			r.Use(h.RequireDevice)
			h.scorerRoutes(r)
		})
		r.Group(func(r chi.Router) {
			r.Use(h.RequireUser)
			h.adminRoutes(r)
			h.visibilityRoutes(r)
		})
	})
	return r
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	status, msg := http.StatusInternalServerError, "internal error"
	switch {
	case errors.Is(err, service.ErrNotFound):
		status, msg = http.StatusNotFound, "not found"
	case errors.Is(err, service.ErrInvalid):
		status, msg = http.StatusUnprocessableEntity, "invalid input"
	case errors.Is(err, service.ErrInvalidUsername), errors.Is(err, service.ErrWeakPassword):
		status, msg = http.StatusUnprocessableEntity, err.Error()
	case errors.Is(err, service.ErrForbidden):
		status, msg = http.StatusForbidden, "forbidden"
	case errors.Is(err, service.ErrBackupNotPermitted):
		status, msg = http.StatusForbidden, "backup_not_permitted"
	case errors.Is(err, service.ErrLastAdmin):
		status, msg = http.StatusConflict, err.Error()
	case errors.Is(err, service.ErrConflict):
		status, msg = http.StatusConflict, "conflict"
	case errors.Is(err, service.ErrLaneNotManned):
		status, msg = http.StatusConflict, "lane is not manned"
	case errors.Is(err, service.ErrNotBetweenStages):
		status, msg = http.StatusConflict, "entries can only change between stages"
	case errors.Is(err, service.ErrStageLocked):
		status, msg = http.StatusConflict, "stage is locked"
	case errors.Is(err, service.ErrUnauthenticated):
		status, msg = http.StatusUnauthorized, "unauthorized"
	case errors.Is(err, errTooManyStreams):
		status, msg = http.StatusServiceUnavailable, "too many streams"
	case errors.Is(err, service.ErrAdminScoringNeeded):
		status, msg = http.StatusForbidden, "admin scoring required"
	}
	writeJSON(w, status, map[string]string{"error": msg})
}

func decode(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return service.ErrInvalid
	}
	return nil
}

func pathInt(r *http.Request, name string) (int64, error) {
	n, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil {
		return 0, service.ErrNotFound
	}
	return n, nil
}
