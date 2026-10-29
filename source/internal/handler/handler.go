package handler

import (
	"net/http"
	"sync/atomic"

	"github.com/go-chi/chi/v5"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

type Handler struct {
	svc          *service.Service
	repo         *repository.Repository
	auth         *service.Auth
	audit        *service.Audit
	wa1500       *service.WA1500Lookup
	secureCookie bool
	servingTLS   bool
	backupDir    string
	version      string
	limits       streamLimits
	render       *RenderGuard
	streams      [streamRoles]atomic.Int64
}

func NewRouter(svc *service.Service, repo *repository.Repository, opts ...Option) http.Handler {
	h := &Handler{svc: svc, repo: repo, limits: defaultStreamLimits()}
	for _, opt := range opts {
		opt(h)
	}
	if h.render == nil {
		h.render = NewRenderGuard(DefaultRenderLimits())
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
			r.With(h.RequireAdmin).Put("/club", h.setClub)
			r.With(h.RequireAdmin).Put("/club/logo", h.putClubLogo)
			r.With(h.RequireAdmin).Delete("/club/logo", h.deleteClubLogo)
		}
		r.Get("/version", h.getVersion)
		h.clubPublicRoutes(r)
		h.rulebookRoutes(r)
		h.spectatorRoutes(r)
		h.screenPublicRoutes(r)
		h.scorecardRoutes(r)
		h.resultsPDFRoutes(r)
		r.Group(func(r chi.Router) {
			r.Use(h.RequireDevice)
			h.scorerRoutes(r)
		})
		r.Group(func(r chi.Router) {
			r.Use(h.RequireUser)
			h.adminRoutes(r)
			h.wa1500Routes(r)
			h.visibilityRoutes(r)
			h.screenAdminRoutes(r)
		})
	})
	return r
}
