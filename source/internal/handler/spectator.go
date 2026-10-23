package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

func (h *Handler) spectatorRoutes(r chi.Router) {
	r.Get("/spectator/competitions", h.spectatorCompetitions)
	r.Get("/spectator/competitions/{competitionID}/standings", h.spectatorStandings)
	r.Get("/spectator/competitions/{competitionID}/stream", h.spectatorStream)
}

func (h *Handler) visibilityRoutes(r chi.Router) {
	r.Get("/competitions/{competitionID}/visibility", h.getVisibility)
	r.Put("/competitions/{competitionID}/visibility", h.setVisibility)
}

type visibilityBody struct {
	ShowNames       bool `json:"show_names"`
	ShowClubs       bool `json:"show_clubs"`
	ShowCountries   bool `json:"show_countries"`
	ShowStageScores bool `json:"show_stage_scores"`
	ShowRingCounts  bool `json:"show_ring_counts"`
}

func visibilityJSON(v repository.Visibility) visibilityBody {
	return visibilityBody{v.Names, v.Clubs, v.Countries, v.StageScores, v.RingCounts}
}

func (h *Handler) getVisibility(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	v, err := h.svc.GetVisibility(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, visibilityJSON(v))
}

func (h *Handler) setVisibility(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	var body visibilityBody
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	v := repository.Visibility{Names: body.ShowNames, Clubs: body.ShowClubs, Countries: body.ShowCountries, StageScores: body.ShowStageScores, RingCounts: body.ShowRingCounts}
	if err := h.svc.SetVisibility(r.Context(), id, v); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, visibilityJSON(v))
}

func (h *Handler) spectatorCompetitions(w http.ResponseWriter, r *http.Request) {
	comps, err := h.svc.ListPublicCompetitions(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(comps))
	for _, c := range comps {
		out = append(out, competitionJSON(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"competitions": out})
}

func (h *Handler) spectatorStandings(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := h.svc.Standings(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"standings": rows})
}

func (h *Handler) spectatorStream(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	h.serveStream(w, r, roleSpectator, func(ctx context.Context) error { return h.svc.CheckPublicStream(ctx, id) }, func() *service.Subscription { return h.svc.Subscribe(id) }, func(ctx context.Context) (streamEvent, error) {
		snap, err := h.svc.SpectatorSnapshot(ctx, id)
		return streamEvent{Name: "snapshot", Data: snap}, err
	})
}
