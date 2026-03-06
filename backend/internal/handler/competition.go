package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/sollie/matchmark/internal/auth"
	"github.com/sollie/matchmark/internal/service"
)

type CompetitionHandler struct {
	competitionSvc *service.CompetitionService
}

func NewCompetitionHandler(competitionSvc *service.CompetitionService) *CompetitionHandler {
	return &CompetitionHandler{competitionSvc: competitionSvc}
}

func (h *CompetitionHandler) List(w http.ResponseWriter, r *http.Request) {
	status := strings.ToLower(r.URL.Query().Get("status"))
	isAdmin := auth.RoleFromContext(r.Context()) == "admin"
	competitions, err := h.competitionSvc.ListVisible(r.Context(), status, isAdmin)
	if err != nil {
		if errors.Is(err, service.ErrInvalidStatus) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to list competitions")
		return
	}
	writeJSON(w, http.StatusOK, competitions)
}

func (h *CompetitionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid competition id")
		return
	}
	isAdmin := auth.RoleFromContext(r.Context()) == "admin"
	c, err := h.competitionSvc.GetVisibleByID(r.Context(), id, isAdmin)
	if err != nil {
		if errors.Is(err, service.ErrCompetitionNotFound) {
			writeError(w, http.StatusNotFound, "competition not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get competition")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *CompetitionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name             string `json:"name"`
		Date             string `json:"date"`
		LanesTotal       int    `json:"lanes_total"`
		SpectatorEnabled bool   `json:"spectator_enabled"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	c, err := h.competitionSvc.Create(r.Context(), service.CreateCompetitionParams{
		Name:             body.Name,
		Date:             body.Date,
		LanesTotal:       body.LanesTotal,
		SpectatorEnabled: body.SpectatorEnabled,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *CompetitionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid competition id")
		return
	}

	var body struct {
		Name             *string `json:"name"`
		Date             *string `json:"date"`
		LanesTotal       *int    `json:"lanes_total"`
		Status           *string `json:"status"`
		SpectatorEnabled *bool   `json:"spectator_enabled"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	c, err := h.competitionSvc.Update(r.Context(), id, service.UpdateCompetitionParams{
		Name:             body.Name,
		Date:             body.Date,
		LanesTotal:       body.LanesTotal,
		Status:           body.Status,
		SpectatorEnabled: body.SpectatorEnabled,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCompetitionNotFound):
			writeError(w, http.StatusNotFound, "competition not found")
		case errors.Is(err, service.ErrInvalidStatus), errors.Is(err, service.ErrInvalidStatusTransition):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "failed to update competition")
		}
		return
	}
	writeJSON(w, http.StatusOK, c)
}
