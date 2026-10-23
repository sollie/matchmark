package handler

import (
	"errors"
	"net/http"

	"github.com/sollie/matchmark/internal/service"
)

type shooterImportBody struct {
	shooterBody
	UpdateShooterID int64 `json:"update_shooter_id"`
	CreateNew       bool  `json:"create_new"`
}

func (h *Handler) importShooter(w http.ResponseWriter, r *http.Request) {
	var body shooterImportBody
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	result, err := h.svc.ImportShooter(r.Context(), service.ShooterImport{
		Shooter: body.shooter(0), UpdateShooterID: body.UpdateShooterID, CreateNew: body.CreateNew,
	})
	var conflict *service.NameConflictError
	if errors.As(err, &conflict) {
		candidates := make([]map[string]any, 0, len(conflict.Candidates))
		for _, c := range conflict.Candidates {
			candidates = append(candidates, shooterJSON(c))
		}
		writeJSON(w, http.StatusConflict, map[string]any{"error": "name_conflict", "candidates": candidates})
		return
	}
	if err != nil {
		writeError(w, err)
		return
	}
	status, action := http.StatusOK, "updated"
	if result.Created {
		status, action = http.StatusCreated, "created"
	}
	writeJSON(w, status, map[string]any{"id": result.ID, "action": action})
}
