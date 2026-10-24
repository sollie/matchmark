package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sollie/matchmark/internal/service"
)

func (h *Handler) resultsPDFRoutes(r chi.Router) {
	r.With(h.render.Middleware(ResourceByPath)).Get("/competitions/{competitionID}/results", h.publicResultsPDF)
}

func (h *Handler) publicResultsPDF(w http.ResponseWriter, r *http.Request) {
	h.resultsPDF(w, r, h.svc.PublicResultsPDF)
}

func (h *Handler) adminResultsPDF(w http.ResponseWriter, r *http.Request) {
	h.resultsPDF(w, r, h.svc.AdminResultsPDF)
}

func (h *Handler) resultsPDF(w http.ResponseWriter, r *http.Request, build func(ctx context.Context, id int64) (service.Download, error)) {
	if r.URL.Query().Has("format") && r.URL.Query().Get("format") != "pdf" {
		writeError(w, service.ErrInvalid)
		return
	}
	id, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	file, err := build(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", contentDisposition(file.BaseName, "pdf"))
	_, _ = w.Write(file.Data)
}
