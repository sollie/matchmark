package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/sollie/matchmark/internal/auth"
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

func (h *ScoringHandler) GetStage(w http.ResponseWriter, r *http.Request) {
	compID, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid competition id")
		return
	}
	lane, err := parseLaneParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid lane")
		return
	}
	stageNum, err := parseStageParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stage")
		return
	}

	callerID := auth.UserIDFromContext(r.Context())
	isAdmin := auth.RoleFromContext(r.Context()) == "admin"

	detail, err := h.scoringSvc.GetStageDetail(r.Context(), compID, lane, stageNum, callerID, isAdmin)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCompetitionNotFound):
			writeError(w, http.StatusNotFound, "competition not found")
		case errors.Is(err, service.ErrStageLaneNotFound):
			writeError(w, http.StatusNotFound, "no shooter on that lane")
		case errors.Is(err, service.ErrShooterNotFound):
			writeError(w, http.StatusNotFound, "stage not found")
		case errors.Is(err, service.ErrNotAssignedToLane):
			writeError(w, http.StatusForbidden, "you are not assigned to score this lane")
		default:
			writeError(w, http.StatusInternalServerError, "failed to get stage")
		}
		return
	}

	type shooterResp struct {
		Name string  `json:"name"`
		WAID string  `json:"wa_id"`
		Club *string `json:"club"`
	}
	type hitResp struct {
		Ring   string `json:"ring"`
		Points int    `json:"points"`
	}
	hits := make([]hitResp, len(detail.Hits))
	for i, h := range detail.Hits {
		hits[i] = hitResp{Ring: h.Ring, Points: h.Points}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"stage_number":       detail.StageNumber,
		"expected_shots":     detail.ExpectedShots,
		"distance":           detail.Distance,
		"time_seconds":       detail.TimeSeconds,
		"position":           detail.Position,
		"discipline":         detail.Discipline,
		"total_stages":       detail.TotalStages,
		"shooter":            shooterResp{Name: detail.Shooter.Name, WAID: detail.Shooter.WAID, Club: detail.Shooter.Club},
		"hits":               hits,
		"locked_by_scorer":   detail.LockedByScorer,
		"fully_locked":       detail.FullyLocked,
		"stage_score":        detail.StageScore,
		"stage_x_count":      detail.StageXCount,
		"cumulative_score":   detail.CumulativeScore,
		"cumulative_x_count": detail.CumulativeXCount,
	})
}

func (h *ScoringHandler) SubmitHits(w http.ResponseWriter, r *http.Request) {
	compID, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid competition id")
		return
	}
	lane, err := parseLaneParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid lane")
		return
	}
	stageNum, err := parseStageParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stage")
		return
	}

	var body struct {
		Hits []string `json:"hits"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	callerID := auth.UserIDFromContext(r.Context())
	isAdmin := auth.RoleFromContext(r.Context()) == "admin"

	result, err := h.scoringSvc.SubmitHits(r.Context(), compID, lane, stageNum, body.Hits, callerID, isAdmin)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCompetitionNotFound):
			writeError(w, http.StatusNotFound, "competition not found")
		case errors.Is(err, service.ErrStageLaneNotFound):
			writeError(w, http.StatusNotFound, "no shooter on that lane")
		case errors.Is(err, service.ErrShooterNotFound):
			writeError(w, http.StatusNotFound, "stage not found")
		case errors.Is(err, service.ErrStageLocked):
			writeError(w, http.StatusConflict, err.Error())
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

	type hitResp struct {
		Ring   string `json:"ring"`
		Points int    `json:"points"`
	}
	hits := make([]hitResp, len(result.Hits))
	for i, h := range result.Hits {
		hits[i] = hitResp{Ring: h.Ring, Points: h.Points}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"hits":        hits,
		"stage_score": result.StageScore,
		"x_count":     result.XCount,
	})
}

func (h *ScoringHandler) LockStage(w http.ResponseWriter, r *http.Request) {
	compID, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid competition id")
		return
	}
	lane, err := parseLaneParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid lane")
		return
	}
	stageNum, err := parseStageParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stage")
		return
	}

	callerID := auth.UserIDFromContext(r.Context())
	isAdmin := auth.RoleFromContext(r.Context()) == "admin"

	if err := h.scoringSvc.LockStage(r.Context(), compID, lane, stageNum, callerID, isAdmin); err != nil {
		switch {
		case errors.Is(err, service.ErrCompetitionNotFound):
			writeError(w, http.StatusNotFound, "competition not found")
		case errors.Is(err, service.ErrStageLaneNotFound):
			writeError(w, http.StatusNotFound, "no shooter on that lane")
		case errors.Is(err, service.ErrShooterNotFound):
			writeError(w, http.StatusNotFound, "stage not found")
		case errors.Is(err, service.ErrNotAssignedToLane):
			writeError(w, http.StatusForbidden, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "failed to lock stage")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"fully_locked": true})
}

func (h *ScoringHandler) UnlockStage(w http.ResponseWriter, r *http.Request) {
	compID, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid competition id")
		return
	}
	lane, err := parseLaneParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid lane")
		return
	}
	stageNum, err := parseStageParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stage")
		return
	}

	isAdmin := auth.RoleFromContext(r.Context()) == "admin"

	if err := h.scoringSvc.UnlockStage(r.Context(), compID, lane, stageNum, isAdmin); err != nil {
		switch {
		case errors.Is(err, service.ErrCompetitionNotFound):
			writeError(w, http.StatusNotFound, "competition not found")
		case errors.Is(err, service.ErrStageLaneNotFound):
			writeError(w, http.StatusNotFound, "no shooter on that lane")
		case errors.Is(err, service.ErrShooterNotFound):
			writeError(w, http.StatusNotFound, "stage not found")
		case errors.Is(err, service.ErrNotAssignedToLane):
			writeError(w, http.StatusForbidden, "admin access required")
		default:
			writeError(w, http.StatusInternalServerError, "failed to unlock stage")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"fully_locked": false})
}
