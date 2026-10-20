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
	r.Get("/scorer/session", h.scorerSession)
	r.Get("/scorer/stream", h.scorerStream)
	r.Put("/scorer/stages/{stage}", h.scorerSetCounts)
	r.Post("/scorer/stages/{stage}/lock", h.scorerLock)
}

func (h *Handler) RequireDevice(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			writeUnauthorized(w)
			return
		}
		d, err := h.svc.AuthenticateDevice(r.Context(), token)
		if err != nil {
			writeError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), deviceKey{}, d)))
	})
}

func deviceFrom(r *http.Request) service.Device {
	d, _ := r.Context().Value(deviceKey{}).(service.Device)
	return d
}

func scorerSessionJSON(sess service.ScorerSession) map[string]any {
	stages := make([]map[string]any, 0, len(sess.Stages))
	for _, s := range sess.Stages {
		stages = append(stages, map[string]any{"stage_number": s.Number, "expected_shots": s.ExpectedShots, "locked": s.Locked})
	}
	return map[string]any{
		"lane": sess.Lane, "score_lane": sess.ScoreLane, "entry_id": sess.EntryID,
		"shooter_name": sess.ShooterName, "stages": stages,
	}
}

func (h *Handler) scorerSession(w http.ResponseWriter, r *http.Request) {
	sess, err := h.svc.DeviceSession(r.Context(), deviceFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, scorerSessionJSON(sess))
}

func (h *Handler) scorerStream(w http.ResponseWriter, r *http.Request) {
	token, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	h.serveStream(w, r, h.svc.Subscribe(deviceFrom(r).CompetitionID), func(ctx context.Context) (streamEvent, error) {
		sess, err := h.svc.DeviceSnapshot(ctx, token)
		if errors.Is(err, service.ErrUnauthenticated) {
			return streamEvent{Name: "revoked", Data: struct{}{}, Final: true}, nil
		}
		if err != nil {
			return streamEvent{}, err
		}
		return streamEvent{Name: "snapshot", Data: scorerSessionJSON(sess)}, nil
	})
}

func (h *Handler) scorerSetCounts(w http.ResponseWriter, r *http.Request) {
	stage, err := pathInt(r, "stage")
	if err != nil {
		writeError(w, err)
		return
	}
	var body ringCountsBody
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	if err := h.svc.DeviceSetRingCounts(r.Context(), deviceFrom(r), int(stage), body.counts()); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *Handler) scorerLock(w http.ResponseWriter, r *http.Request) {
	stage, err := pathInt(r, "stage")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := h.svc.DeviceLockStage(r.Context(), deviceFrom(r), int(stage)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"locked": true})
}

func (h *Handler) createPairing(w http.ResponseWriter, r *http.Request) {
	compID, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	var body struct {
		Lane int `json:"lane"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	id, token, err := h.svc.CreatePairing(r.Context(), compID, body.Lane)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "lane": body.Lane, "token": token})
}

func (h *Handler) listPairings(w http.ResponseWriter, r *http.Request) {
	compID, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	pairings, err := h.svc.ListPairings(r.Context(), compID)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(pairings))
	for _, p := range pairings {
		out = append(out, pairingJSON(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"pairings": out})
}

func pairingJSON(p repository.Pairing) map[string]any {
	return map[string]any{"id": p.ID, "lane": p.Lane, "revoked": p.Revoked}
}

func (h *Handler) revokePairing(w http.ResponseWriter, r *http.Request) {
	compID, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	id, err := pathInt(r, "pairingID")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := h.svc.RevokePairing(r.Context(), compID, id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) lockStage(w http.ResponseWriter, r *http.Request) {
	compID, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	entryID, err := pathInt(r, "entryID")
	if err != nil {
		writeError(w, err)
		return
	}
	stage, err := pathInt(r, "stage")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := h.svc.AdminLockStage(r.Context(), compID, entryID, int(stage)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"locked": true})
}

func (h *Handler) unlockStage(w http.ResponseWriter, r *http.Request) {
	compID, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	entryID, err := pathInt(r, "entryID")
	if err != nil {
		writeError(w, err)
		return
	}
	stage, err := pathInt(r, "stage")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := h.svc.UnlockStage(r.Context(), compID, entryID, int(stage)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"locked": false})
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
