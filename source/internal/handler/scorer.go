package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

type deviceKey struct{}

func (h *Handler) scorerRoutes(r chi.Router) {
	r.Get("/scorer/session", serve(h.scorerSession))
	r.Get("/scorer/stream", h.scorerStream)
	r.Put("/scorer/stages/{stage}", serve(h.scorerSetCounts))
	r.Post("/scorer/stages/{stage}/lock", serve(h.scorerLock))
	r.Post("/scorer/stages/{stage}/accept", serve(h.scorerAccept))
	r.Post("/scorer/stages/{stage}/dispute", serve(h.scorerDispute))
}

func (h *Handler) RequireDevice(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			writeUnauthorized(w)
			return
		}
		d, err := h.svc.AuthenticateDevice(r.Context(), token)
		if err != nil {
			writeError(w, err)
			return
		}
		ctx := context.WithValue(r.Context(), deviceKey{}, d)
		ctx = withActor(ctx, r, 0, service.DeviceActor)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func bearerToken(r *http.Request) string {
	token, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	return token
}

func deviceFrom(r *http.Request) service.Device {
	d, _ := r.Context().Value(deviceKey{}).(service.Device)
	return d
}

func scorerSessionJSON(sess service.ScorerSession) map[string]any {
	stages := make([]map[string]any, 0, len(sess.Stages))
	for _, s := range sess.Stages {
		stage := ringCountsJSON(s.Counts)
		stage["stage_number"] = s.Number
		stage["expected_shots"] = s.ExpectedShots
		stage["status"] = s.Status
		stage["disputed"] = s.DisputedAt != ""
		if part := sess.Parts[s.Number]; part != "" {
			stage["part"] = part
		}
		stages = append(stages, stage)
	}
	out := map[string]any{
		"match_id": sess.MatchID, "lane": sess.Lane, "score_lane": sess.ScoreLane, "entry_id": sess.EntryID,
		"shooter_name": sess.ShooterName, "stages": stages, "phase": sess.Phase, "current_stage": sess.CurrentStage,
	}
	if a := sess.Acceptance; a != nil {
		acceptance := ringCountsJSON(a.Counts)
		acceptance["stage_number"] = a.StageNumber
		acceptance["total"] = a.Total
		acceptance["scorer_name"] = a.ScorerName
		if part := sess.Parts[a.StageNumber]; part != "" {
			acceptance["part"] = part
		}
		out["acceptance"] = acceptance
	}
	return out
}

func (h *Handler) scorerSession(r *http.Request) (int, any, error) {
	sess, err := h.svc.DeviceSession(r.Context(), deviceFrom(r))
	if err != nil {
		return fail(err)
	}
	return okJSON(scorerSessionJSON(sess))
}

func (h *Handler) scorerStream(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	defer h.svc.DeviceConnected(deviceFrom(r))()
	h.serveStream(w, r, roleScorer, func(context.Context) error {
		if deviceFrom(r).Finished {
			return service.ErrGone
		}
		return nil
	}, func() *service.Subscription {
		return h.svc.Subscribe(deviceFrom(r).MatchID)
	}, func(ctx context.Context) (streamEvent, error) {
		sess, err := h.svc.DeviceSnapshot(ctx, token)
		if errors.Is(err, service.ErrUnauthenticated) {
			return streamEvent{Name: "revoked", Data: struct{}{}, Final: true}, nil
		}
		if err != nil {
			return streamEvent{}, err
		}
		return streamEvent{Name: "snapshot", Data: scorerSessionJSON(sess), Final: sess.Phase == service.PhaseFinished}, nil
	})
}

func (h *Handler) scorerSetCounts(r *http.Request) (int, any, error) {
	req := in(r)
	stage := req.path("stage")
	var body ringCountsBody
	req.body(&body)
	if req.err != nil {
		return fail(req.err)
	}
	if err := h.svc.DeviceSetRingCounts(r.Context(), deviceFrom(r), int(stage), body.counts()); err != nil {
		return fail(err)
	}
	return okJSON(body)
}

func (h *Handler) scorerLock(r *http.Request) (int, any, error) {
	return h.scorerTransition(r, h.svc.DeviceLockStage)
}

func ringCountsJSON(c repository.RingCounts) map[string]any {
	return map[string]any{"x": c.X, "10": c.Ten, "9": c.Nine, "8": c.Eight, "7": c.Seven, "0": c.Zero, "miss": c.Miss}
}

func (h *Handler) scorerAccept(r *http.Request) (int, any, error) {
	return h.scorerTransition(r, h.svc.DeviceAcceptStage)
}

func (h *Handler) scorerDispute(r *http.Request) (int, any, error) {
	return h.scorerTransition(r, h.svc.DeviceDisputeStage)
}

func (h *Handler) scorerTransition(r *http.Request, act func(context.Context, service.Device, int) (repository.StageStatus, error)) (int, any, error) {
	req := in(r)
	stage := req.path("stage")
	if req.err != nil {
		return fail(req.err)
	}
	status, err := act(r.Context(), deviceFrom(r), int(stage))
	if err != nil {
		return fail(err)
	}
	return okJSON(map[string]repository.StageStatus{"status": status})
}

func (h *Handler) createPairing(r *http.Request) (int, any, error) {
	req := in(r)
	compID := req.path("matchID")
	var body struct {
		Lane int `json:"lane"`
	}
	req.body(&body)
	if req.err != nil {
		return fail(req.err)
	}
	id, token, err := h.svc.CreatePairing(r.Context(), compID, body.Lane)
	if err != nil {
		return fail(err)
	}
	return http.StatusCreated, map[string]any{"id": id, "lane": body.Lane, "token": token}, nil
}

func (h *Handler) listPairings(r *http.Request) (int, any, error) {
	req := in(r)
	compID := req.path("matchID")
	if req.err != nil {
		return fail(req.err)
	}
	pairings, err := h.svc.ListPairings(r.Context(), compID)
	if err != nil {
		return fail(err)
	}
	out := make([]map[string]any, 0, len(pairings))
	for _, p := range pairings {
		out = append(out, pairingJSON(p))
	}
	return okJSON(map[string]any{"pairings": out})
}

func pairingJSON(p repository.Pairing) map[string]any {
	return map[string]any{"id": p.ID, "lane": p.Lane, "revoked": p.Revoked}
}

func (h *Handler) revokePairing(r *http.Request) (int, any, error) {
	req := in(r)
	compID, id := req.path("matchID"), req.path("pairingID")
	if req.err != nil {
		return fail(req.err)
	}
	if err := h.svc.RevokePairing(r.Context(), compID, id); err != nil {
		return fail(err)
	}
	return http.StatusNoContent, nil, nil
}

func (h *Handler) lockStage(r *http.Request) (int, any, error) {
	return h.adminTransition(r, h.svc.AdminLockStage)
}

func (h *Handler) acceptStage(r *http.Request) (int, any, error) {
	return h.adminTransition(r, h.svc.AdminAcceptStage)
}

func (h *Handler) unlockStage(r *http.Request) (int, any, error) {
	return h.adminTransition(r, h.svc.UnlockStage)
}

func (h *Handler) adminTransition(r *http.Request, act func(context.Context, int64, int64, int) (repository.StageStatus, error)) (int, any, error) {
	req := in(r)
	compID, entryID, stage := req.path("matchID"), req.path("entryID"), req.path("stage")
	if req.err != nil {
		return fail(req.err)
	}
	status, err := act(r.Context(), compID, entryID, int(stage))
	if err != nil {
		return fail(err)
	}
	return okJSON(map[string]repository.StageStatus{"status": status})
}

type ringCountsBody struct {
	X     int `json:"x"`
	Ten   int `json:"10"`
	Nine  int `json:"9"`
	Eight int `json:"8"`
	Seven int `json:"7"`
	Zero  int `json:"0"`
	Miss  int `json:"miss"`
}

func (b ringCountsBody) counts() repository.RingCounts {
	return repository.RingCounts{X: b.X, Ten: b.Ten, Nine: b.Nine, Eight: b.Eight, Seven: b.Seven, Zero: b.Zero, Miss: b.Miss}
}
