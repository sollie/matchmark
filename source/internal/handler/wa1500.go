package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sollie/matchmark/internal/service"
)

func WithWA1500(lookup *service.WA1500Lookup) Option {
	return func(h *Handler) { h.wa1500 = lookup }
}

func (h *Handler) wa1500Routes(r chi.Router) {
	r.Get("/wa1500/search", h.searchWA1500)
}

func (h *Handler) searchWA1500(w http.ResponseWriter, r *http.Request) {
	results, err := h.wa1500.Search(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}
