package handler

import (
	"errors"
	"net/http"

	"github.com/sollie/matchmark/internal/auth"
	"github.com/sollie/matchmark/internal/model"
	"github.com/sollie/matchmark/internal/service"
)

type AssignmentHandler struct {
	assignmentSvc *service.AssignmentService
}

func NewAssignmentHandler(assignmentSvc *service.AssignmentService) *AssignmentHandler {
	return &AssignmentHandler{assignmentSvc: assignmentSvc}
}

func (h *AssignmentHandler) SelectShooter(w http.ResponseWriter, r *http.Request) {
	compID, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid competition id")
		return
	}

	var body struct {
		ShooterID int64 `json:"shooter_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	callerID := auth.UserIDFromContext(r.Context())

	assignment, err := h.assignmentSvc.SelectShooter(r.Context(), compID, body.ShooterID, callerID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCompetitionNotFound):
			writeError(w, http.StatusNotFound, "competition not found")
		case errors.Is(err, service.ErrShooterNotFound):
			writeError(w, http.StatusNotFound, "shooter not found")
		case errors.Is(err, service.ErrShooterAlreadyClaimed):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, service.ErrUserAlreadyClaimed):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "failed to assign shooter")
		}
		return
	}
	writeJSON(w, http.StatusOK, assignment)
}

func (h *AssignmentHandler) ReleaseAssignment(w http.ResponseWriter, r *http.Request) {
	compID, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid competition id")
		return
	}

	var body struct {
		ShooterID int64 `json:"shooter_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.assignmentSvc.ReleaseAssignment(r.Context(), compID, body.ShooterID); err != nil {
		switch {
		case errors.Is(err, service.ErrCompetitionNotFound):
			writeError(w, http.StatusNotFound, "competition not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to release assignment")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AssignmentHandler) GetMyAssignment(w http.ResponseWriter, r *http.Request) {
	compID, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid competition id")
		return
	}

	callerID := auth.UserIDFromContext(r.Context())

	assignment, err := h.assignmentSvc.GetMyAssignment(r.Context(), compID, callerID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAssignmentNotFound):
			writeError(w, http.StatusNotFound, "no assignment found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to get assignment")
		}
		return
	}
	writeJSON(w, http.StatusOK, assignment)
}

func (h *AssignmentHandler) ListAssignments(w http.ResponseWriter, r *http.Request) {
	compID, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid competition id")
		return
	}

	assignments, err := h.assignmentSvc.ListAssignments(r.Context(), compID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCompetitionNotFound):
			writeError(w, http.StatusNotFound, "competition not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to list assignments")
		}
		return
	}
	if assignments == nil {
		assignments = []model.ScorerAssignment{}
	}
	writeJSON(w, http.StatusOK, assignments)
}
