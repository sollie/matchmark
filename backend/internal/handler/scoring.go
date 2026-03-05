package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sollie/matchmark/internal/auth"
	"github.com/sollie/matchmark/internal/model"
	"github.com/sollie/matchmark/internal/service"
)

type ScoringHandler struct {
	scoringSvc *service.ScoringService
}

func NewScoringHandler(scoringSvc *service.ScoringService) *ScoringHandler {
	return &ScoringHandler{scoringSvc: scoringSvc}
}

func parseLaneParam(r *http.Request) (int, error) {
	return strconv.Atoi(chi.URLParam(r, "lane"))
}

func parseStageParam(r *http.Request) (int, error) {
	return strconv.Atoi(chi.URLParam(r, "stage"))
}

func isAdminFromCtx(ctx context.Context) bool {
	return auth.RoleFromContext(ctx) == "admin"
}

type hitResp struct {
	Ring   string `json:"ring"`
	Points int    `json:"points"`
}

func hitsToResp(hits []model.Hit) []hitResp {
	out := make([]hitResp, len(hits))
	for i, h := range hits {
		out[i] = hitResp{Ring: h.Ring, Points: h.Points}
	}
	return out
}

func parseScoringParams(r *http.Request) (int64, int, int, error) {
	compID, err := parseIDParam(r, "id")
	if err != nil {
		return 0, 0, 0, err
	}
	lane, err := parseLaneParam(r)
	if err != nil {
		return 0, 0, 0, err
	}
	stage, err := parseStageParam(r)
	if err != nil {
		return 0, 0, 0, err
	}
	return compID, lane, stage, nil
}

func writeScoringError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, service.ErrCompetitionNotFound):
		writeError(w, http.StatusNotFound, "competition not found")
	case errors.Is(err, service.ErrStageLaneNotFound):
		writeError(w, http.StatusNotFound, "no shooter on that lane")
	case errors.Is(err, service.ErrShooterNotFound):
		writeError(w, http.StatusNotFound, "stage not found")
	default:
		return false
	}
	return true
}

type shooterResp struct {
	Name string  `json:"name"`
	WAID string  `json:"wa_id"`
	Club *string `json:"club"`
}

type stageResp struct {
	StageNumber      int         `json:"stage_number"`
	ExpectedShots    int         `json:"expected_shots"`
	Distance         string      `json:"distance"`
	TimeSeconds      int         `json:"time_seconds"`
	Position         string      `json:"position"`
	Discipline       string      `json:"discipline"`
	TotalStages      int         `json:"total_stages"`
	Shooter          shooterResp `json:"shooter"`
	Hits             []hitResp   `json:"hits"`
	LockedByScorer   bool        `json:"locked_by_scorer"`
	FullyLocked      bool        `json:"fully_locked"`
	StageScore       int         `json:"stage_score"`
	StageXCount      int         `json:"stage_x_count"`
	CumulativeScore  int         `json:"cumulative_score"`
	CumulativeXCount int         `json:"cumulative_x_count"`
}

type submitHitsResp struct {
	Hits        []hitResp `json:"hits"`
	StageScore  int       `json:"stage_score"`
	StageXCount int       `json:"stage_x_count"`
}

type lockResp struct {
	FullyLocked bool `json:"fully_locked"`
}

type streamEventResp struct {
	StageScore  int       `json:"stage_score"`
	StageXCount int       `json:"stage_x_count"`
	Hits        []hitResp `json:"hits"`
	FullyLocked bool      `json:"fully_locked"`
}

func (h *ScoringHandler) GetStage(w http.ResponseWriter, r *http.Request) {
	compID, lane, stageNum, err := parseScoringParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid path parameters")
		return
	}

	callerID := auth.UserIDFromContext(r.Context())

	detail, err := h.scoringSvc.GetStageDetail(r.Context(), compID, lane, stageNum, callerID, isAdminFromCtx(r.Context()))
	if err != nil {
		if writeScoringError(w, err) {
			return
		}
		switch {
		case errors.Is(err, service.ErrNotAssignedToLane):
			writeError(w, http.StatusForbidden, "you are not assigned to score this lane")
		default:
			writeError(w, http.StatusInternalServerError, "failed to get stage")
		}
		return
	}

	writeJSON(w, http.StatusOK, stageResp{
		StageNumber:      detail.StageNumber,
		ExpectedShots:    detail.ExpectedShots,
		Distance:         detail.Distance,
		TimeSeconds:      detail.TimeSeconds,
		Position:         detail.Position,
		Discipline:       detail.Discipline,
		TotalStages:      detail.TotalStages,
		Shooter:          shooterResp{Name: detail.Shooter.Name, WAID: detail.Shooter.WAID, Club: detail.Shooter.Club},
		Hits:             hitsToResp(detail.Hits),
		LockedByScorer:   detail.LockedByScorer,
		FullyLocked:      detail.FullyLocked,
		StageScore:       detail.StageScore,
		StageXCount:      detail.StageXCount,
		CumulativeScore:  detail.CumulativeScore,
		CumulativeXCount: detail.CumulativeXCount,
	})
}

func (h *ScoringHandler) SubmitHits(w http.ResponseWriter, r *http.Request) {
	compID, lane, stageNum, err := parseScoringParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid path parameters")
		return
	}

	var body struct {
		Hits []string `json:"hits"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(body.Hits) > 200 {
		writeError(w, http.StatusBadRequest, "too many hits")
		return
	}

	callerID := auth.UserIDFromContext(r.Context())

	result, err := h.scoringSvc.SubmitHits(r.Context(), compID, lane, stageNum, body.Hits, callerID, isAdminFromCtx(r.Context()))
	if err != nil {
		if writeScoringError(w, err) {
			return
		}
		switch {
		case errors.Is(err, service.ErrStageLocked):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, service.ErrSingleLaneAdminOnly):
			writeError(w, http.StatusForbidden, "admin scoring required")
		case errors.Is(err, service.ErrNotAssignedToLane):
			writeError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, service.ErrWrongShotCount):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrInvalidRing):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "failed to submit hits")
		}
		return
	}

	writeJSON(w, http.StatusOK, submitHitsResp{
		Hits:        hitsToResp(result.Hits),
		StageScore:  result.StageScore,
		StageXCount: result.XCount,
	})
}

func (h *ScoringHandler) LockStage(w http.ResponseWriter, r *http.Request) {
	compID, lane, stageNum, err := parseScoringParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid path parameters")
		return
	}

	callerID := auth.UserIDFromContext(r.Context())

	if err := h.scoringSvc.LockStage(r.Context(), compID, lane, stageNum, callerID, isAdminFromCtx(r.Context())); err != nil {
		if writeScoringError(w, err) {
			return
		}
		switch {
		case errors.Is(err, service.ErrSingleLaneAdminOnly):
			writeError(w, http.StatusForbidden, "admin scoring required")
		case errors.Is(err, service.ErrNotAssignedToLane):
			writeError(w, http.StatusForbidden, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "failed to lock stage")
		}
		return
	}

	writeJSON(w, http.StatusOK, lockResp{FullyLocked: true})
}

func (h *ScoringHandler) UnlockStage(w http.ResponseWriter, r *http.Request) {
	compID, lane, stageNum, err := parseScoringParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid path parameters")
		return
	}

	if err := h.scoringSvc.UnlockStage(r.Context(), compID, lane, stageNum, isAdminFromCtx(r.Context())); err != nil {
		if writeScoringError(w, err) {
			return
		}
		switch {
		case errors.Is(err, service.ErrAdminRequired):
			writeError(w, http.StatusForbidden, "admin access required")
		default:
			writeError(w, http.StatusInternalServerError, "failed to unlock stage")
		}
		return
	}

	writeJSON(w, http.StatusOK, lockResp{FullyLocked: false})
}

func (h *ScoringHandler) StreamStage(w http.ResponseWriter, r *http.Request) {
	compID, lane, stageNum, err := parseScoringParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid path parameters")
		return
	}

	userID := auth.UserIDFromContext(r.Context())

	_, verifyErr := h.scoringSvc.GetStageDetail(r.Context(), compID, lane, stageNum, userID, false)
	if verifyErr != nil {
		if writeScoringError(w, verifyErr) {
			return
		}
		if errors.Is(verifyErr, service.ErrNotAssignedToLane) {
			writeError(w, http.StatusForbidden, "you are not assigned to score this lane")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to verify lane assignment")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	emit := func() bool {
		detail, err := h.scoringSvc.GetStageDetail(r.Context(), compID, lane, stageNum, userID, false)
		if err != nil {
			_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
			flusher.Flush()
			return true
		}
		data, err := json.Marshal(streamEventResp{
			StageScore:  detail.StageScore,
			StageXCount: detail.StageXCount,
			Hits:        hitsToResp(detail.Hits),
			FullyLocked: detail.FullyLocked,
		})
		if err != nil {
			return true
		}
		_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return detail.FullyLocked
	}

	if done := emit(); done {
		return
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if done := emit(); done {
				return
			}
		}
	}
}
