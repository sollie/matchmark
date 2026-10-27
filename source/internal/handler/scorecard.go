package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) scorecardRoutes(r chi.Router) {
	r.With(h.render.Middleware(ResourceByPath)).Get("/scorer/scorecard/{matchID}/{lane}", h.getScorecard)
	r.With(h.render.Middleware(ResourceByPath)).Get("/scorer/scorecard/{matchID}/{lane}/summary", h.getScorecardSummary)
}

func (h *Handler) getScorecard(w http.ResponseWriter, r *http.Request) {
	if err := requireFormat(r, "png"); err != nil {
		writeError(w, err)
		return
	}
	compID, err := pathInt(r, "matchID")
	if err != nil {
		writeError(w, err)
		return
	}
	lane, err := pathInt(r, "lane")
	if err != nil {
		writeError(w, err)
		return
	}
	card, err := h.svc.ScorecardPNG(r.Context(), compID, int(lane))
	if err != nil {
		writeError(w, err)
		return
	}
	writeRendered(w, "image/png", "png", card, r.URL.Query().Get("download") == "1")
}

func (h *Handler) getScorecardSummary(w http.ResponseWriter, r *http.Request) {
	compID, err := pathInt(r, "matchID")
	if err != nil {
		writeError(w, err)
		return
	}
	lane, err := pathInt(r, "lane")
	if err != nil {
		writeError(w, err)
		return
	}
	summary, err := h.svc.ScorecardSummary(r.Context(), compID, int(lane))
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "private, no-cache")
	writeJSON(w, http.StatusOK, summary)
}
