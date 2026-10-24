package handler

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sollie/matchmark/internal/service"
)

func (h *Handler) scorecardRoutes(r chi.Router) {
	r.With(h.render.Middleware(ResourceByPath)).Get("/scorer/scorecard/{competitionID}/{lane}", h.getScorecard)
	r.With(h.render.Middleware(ResourceByPath)).Get("/scorer/scorecard/{competitionID}/{lane}/summary", h.getScorecardSummary)
}

func (h *Handler) getScorecard(w http.ResponseWriter, r *http.Request) {
	if f := r.URL.Query().Get("format"); r.URL.Query().Has("format") && f != "png" {
		writeError(w, service.ErrInvalid)
		return
	}
	compID, err := pathInt(r, "competitionID")
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
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", card.BaseName+".png"))
	}
	_, _ = w.Write(card.Data)
}

func (h *Handler) getScorecardSummary(w http.ResponseWriter, r *http.Request) {
	compID, err := pathInt(r, "competitionID")
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
