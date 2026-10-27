package handler

import (
	"net/http"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

func (h *Handler) createShooter(r *http.Request) (int, any, error) {
	req := in(r)
	var body shooterBody
	req.body(&body)
	if req.err != nil {
		return fail(req.err)
	}
	id, err := h.svc.CreateShooter(r.Context(), body.shooter(0))
	if err != nil {
		return fail(err)
	}
	return http.StatusCreated, map[string]int64{"id": id}, nil
}

func (h *Handler) createMatch(r *http.Request) (int, any, error) {
	req := in(r)
	var body matchBody
	req.body(&body)
	if req.err != nil {
		return fail(req.err)
	}
	m := repository.Match{CompetitionID: body.CompetitionID, Name: body.Name, Date: body.Date, Course: body.Course}
	var id int64
	var err error
	switch {
	case body.Competition != nil && body.CompetitionID != 0:
		err = service.ErrInvalid
	case body.Competition != nil:
		id, err = h.svc.CreateMatchInNewCompetition(r.Context(), body.Competition.competition(0), m)
	default:
		id, err = h.svc.CreateMatch(r.Context(), m)
	}
	if err != nil {
		return fail(err)
	}
	return http.StatusCreated, map[string]int64{"id": id}, nil
}

func (h *Handler) addEntry(r *http.Request) (int, any, error) {
	req := in(r)
	compID := req.path("matchID")
	var body struct {
		ShooterID int64  `json:"shooter_id"`
		Category  string `json:"category"`
		Lane      int    `json:"lane"`
	}
	req.body(&body)
	if req.err != nil {
		return fail(req.err)
	}
	id, err := h.svc.AddEntry(r.Context(), repository.Entry{
		MatchID: compID, ShooterID: body.ShooterID, Category: body.Category, Lane: body.Lane,
	})
	if err != nil {
		return fail(err)
	}
	return http.StatusCreated, map[string]int64{"id": id}, nil
}

func (h *Handler) withdrawEntry(r *http.Request) (int, any, error) {
	req := in(r)
	compID, entryID := req.path("matchID"), req.path("entryID")
	if req.err != nil {
		return fail(req.err)
	}
	if err := h.svc.WithdrawEntry(r.Context(), compID, entryID); err != nil {
		return fail(err)
	}
	return okJSON(map[string]bool{"withdrawn": true})
}

func (h *Handler) scoreLane(r *http.Request) (int, any, error) {
	req := in(r)
	compID, lane := req.path("matchID"), req.path("lane")
	if req.err != nil {
		return fail(req.err)
	}
	scoreLane, err := h.svc.ScoreLane(r.Context(), compID, int(lane))
	if err != nil {
		return fail(err)
	}
	return okJSON(map[string]int{"score_lane": scoreLane})
}

func (h *Handler) setRingCounts(r *http.Request) (int, any, error) {
	req := in(r)
	compID, entryID, stage := req.path("matchID"), req.path("entryID"), req.path("stage")
	var body ringCountsBody
	req.body(&body)
	if req.err != nil {
		return fail(req.err)
	}
	if err := h.svc.AdminSetRingCounts(r.Context(), compID, entryID, int(stage), body.counts()); err != nil {
		return fail(err)
	}
	return okJSON(body)
}
