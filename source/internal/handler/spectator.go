package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

func (h *Handler) spectatorRoutes(r chi.Router) {
	r.Get("/spectator/competitions", serve(h.spectatorCompetitions))
	r.Get("/spectator/competitions/{competitionID}/standings", serve(h.spectatorStandings))
	r.Get("/spectator/competitions/{competitionID}/results", h.spectatorResults)
	r.Get("/spectator/competitions/{competitionID}/stream", h.spectatorStream)
}

func (h *Handler) visibilityRoutes(r chi.Router) {
	r.Get("/competitions/{competitionID}/visibility", serve(h.getVisibility))
	r.Put("/competitions/{competitionID}/visibility", serve(h.setVisibility))
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

func (h *Handler) getVisibility(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("competitionID")
	if req.err != nil {
		return fail(req.err)
	}
	v, err := h.repo.GetVisibility(r.Context(), id)
	if err != nil {
		return fail(err)
	}
	return okJSON(visibilityJSON(v))
}

func (h *Handler) setVisibility(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("competitionID")
	var body visibilityBody
	req.body(&body)
	if req.err != nil {
		return fail(req.err)
	}
	v := repository.Visibility{Names: body.ShowNames, Clubs: body.ShowClubs, Countries: body.ShowCountries, StageScores: body.ShowStageScores, RingCounts: body.ShowRingCounts}
	if err := h.svc.SetVisibility(r.Context(), id, v); err != nil {
		return fail(err)
	}
	return okJSON(visibilityJSON(v))
}

func (h *Handler) spectatorCompetitions(r *http.Request) (int, any, error) {
	comps, err := h.repo.ListPublicCompetitions(r.Context())
	if err != nil {
		return fail(err)
	}
	return okJSON(map[string]any{"competitions": competitionsJSON(comps)})
}

func (h *Handler) spectatorStandings(r *http.Request) (int, any, error) {
	req := in(r)
	id := req.path("competitionID")
	if req.err != nil {
		return fail(req.err)
	}
	rows, err := h.svc.Standings(r.Context(), id)
	if err != nil {
		return fail(err)
	}
	return okJSON(map[string]any{"standings": rows})
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

func (h *Handler) spectatorResults(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	res, err := h.svc.PublicResults(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	writeJSON(w, http.StatusOK, res)
}
