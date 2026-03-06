package handler

import (
	"errors"
	"net/http"

	"github.com/sollie/matchmark/internal/auth"
	"github.com/sollie/matchmark/internal/service"
)

type ShooterHandler struct {
	shooterSvc *service.ShooterService
}

func NewShooterHandler(shooterSvc *service.ShooterService) *ShooterHandler {
	return &ShooterHandler{shooterSvc: shooterSvc}
}

func (h *ShooterHandler) List(w http.ResponseWriter, r *http.Request) {
	compID, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid competition id")
		return
	}
	isAdmin := auth.RoleFromContext(r.Context()) == "admin"
	if isAdmin {
		shooters, err := h.shooterSvc.ListByCompetition(r.Context(), compID)
		if err != nil {
			if errors.Is(err, service.ErrCompetitionNotFound) {
				writeError(w, http.StatusNotFound, "competition not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to list shooters")
			return
		}
		writeJSON(w, http.StatusOK, shooters)
		return
	}

	shooters, err := h.shooterSvc.ListVisibleByCompetition(r.Context(), compID, false)
	if err != nil {
		if errors.Is(err, service.ErrCompetitionNotFound) {
			writeError(w, http.StatusNotFound, "competition not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to list shooters")
		return
	}
	writeJSON(w, http.StatusOK, shooters)
}

func (h *ShooterHandler) Create(w http.ResponseWriter, r *http.Request) {
	compID, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid competition id")
		return
	}

	var body struct {
		Name        string  `json:"name"`
		WAID        string  `json:"wa_id"`
		Club        *string `json:"club"`
		LaneNumber  int     `json:"lane_number"`
		Discipline  string  `json:"discipline"`
		WeaponClass string  `json:"weapon_class"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(body.Name) > maxNameLen {
		writeError(w, http.StatusBadRequest, "name too long")
		return
	}
	if len(body.WAID) > maxWAIDLen {
		writeError(w, http.StatusBadRequest, "wa_id too long")
		return
	}
	if body.Club != nil && len(*body.Club) > maxClubLen {
		writeError(w, http.StatusBadRequest, "club too long")
		return
	}

	shooter, err := h.shooterSvc.Create(r.Context(), service.CreateShooterParams{
		CompetitionID: compID,
		Name:          body.Name,
		WAID:          body.WAID,
		Club:          body.Club,
		LaneNumber:    body.LaneNumber,
		Discipline:    body.Discipline,
		WeaponClass:   body.WeaponClass,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCompetitionNotFound):
			writeError(w, http.StatusNotFound, "competition not found")
		case errors.Is(err, service.ErrLaneOutOfRange):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrLaneTaken):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, service.ErrInvalidWeaponClass):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "failed to create shooter")
		}
		return
	}
	writeJSON(w, http.StatusCreated, shooter)
}

func (h *ShooterHandler) Update(w http.ResponseWriter, r *http.Request) {
	compID, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid competition id")
		return
	}
	shooterID, err := parseIDParam(r, "sid")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid shooter id")
		return
	}

	var body struct {
		Name        *string `json:"name"`
		WAID        *string `json:"wa_id"`
		Club        *string `json:"club"`
		LaneNumber  *int    `json:"lane_number"`
		Discipline  *string `json:"discipline"`
		WeaponClass *string `json:"weapon_class"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Name != nil && len(*body.Name) > maxNameLen {
		writeError(w, http.StatusBadRequest, "name too long")
		return
	}
	if body.WAID != nil && len(*body.WAID) > maxWAIDLen {
		writeError(w, http.StatusBadRequest, "wa_id too long")
		return
	}
	if body.Club != nil && len(*body.Club) > maxClubLen {
		writeError(w, http.StatusBadRequest, "club too long")
		return
	}

	shooter, err := h.shooterSvc.Update(r.Context(), compID, shooterID, service.UpdateShooterParams{
		Name:        body.Name,
		WAID:        body.WAID,
		Club:        body.Club,
		LaneNumber:  body.LaneNumber,
		Discipline:  body.Discipline,
		WeaponClass: body.WeaponClass,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCompetitionNotFound):
			writeError(w, http.StatusNotFound, "competition not found")
		case errors.Is(err, service.ErrShooterNotFound):
			writeError(w, http.StatusNotFound, "shooter not found")
		case errors.Is(err, service.ErrLaneOutOfRange):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrLaneTaken):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, service.ErrInvalidWeaponClass):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "failed to update shooter")
		}
		return
	}
	writeJSON(w, http.StatusOK, shooter)
}

func (h *ShooterHandler) Delete(w http.ResponseWriter, r *http.Request) {
	compID, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid competition id")
		return
	}
	shooterID, err := parseIDParam(r, "sid")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid shooter id")
		return
	}

	if err := h.shooterSvc.Delete(r.Context(), compID, shooterID); err != nil {
		switch {
		case errors.Is(err, service.ErrCompetitionNotFound):
			writeError(w, http.StatusNotFound, "competition not found")
		case errors.Is(err, service.ErrShooterNotFound):
			writeError(w, http.StatusNotFound, "shooter not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete shooter")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
