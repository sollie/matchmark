package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/sollie/matchmark/internal/service"
)

func (h *Handler) clubPublicRoutes(r chi.Router) {
	r.Get("/club", h.getClub)
	r.Get("/club/logo", h.getClubLogo)
}

type clubResponse struct {
	Name    string `json:"name"`
	HasLogo bool   `json:"has_logo"`
	Version int64  `json:"version"`
}

func toClubResponse(c service.ClubSettings) clubResponse {
	return clubResponse{Name: c.Name, HasLogo: c.HasLogo, Version: c.Version}
}

func (h *Handler) getClub(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.GetClub(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	writeJSON(w, http.StatusOK, toClubResponse(c))
}

func (h *Handler) setClub(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	c, err := h.svc.SetClubName(r.Context(), body.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toClubResponse(c))
}

func (h *Handler) getClubLogo(w http.ResponseWriter, r *http.Request) {
	l, err := h.svc.GetClubLogo(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", l.ContentType)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(l.Data)
}

func (h *Handler) putClubLogo(w http.ResponseWriter, r *http.Request) {
	contentType, _, _ := strings.Cut(r.Header.Get("Content-Type"), ";")
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, service.MaxLogoBytes+1))
	if err != nil {
		writeError(w, service.ErrInvalid)
		return
	}
	if err := h.svc.SetClubLogo(r.Context(), strings.TrimSpace(contentType), data); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteClubLogo(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteClubLogo(r.Context()); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
