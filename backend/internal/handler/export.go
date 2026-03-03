package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/sollie/matchmark/internal/service"
)

type ExportHandler struct {
	exportSvc *service.ExportService
}

func NewExportHandler(exportSvc *service.ExportService) *ExportHandler {
	return &ExportHandler{exportSvc: exportSvc}
}

func (h *ExportHandler) competitionID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("invalid id")
	}
	return id, nil
}

func (h *ExportHandler) ExportJSON(w http.ResponseWriter, r *http.Request) {
	id, err := h.competitionID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid competition id")
		return
	}
	data, err := h.exportSvc.ExportJSON(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrCompetitionNotFound) {
			writeError(w, http.StatusNotFound, "competition not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "export failed")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"competition-%d.json\"", id))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *ExportHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	id, err := h.competitionID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid competition id")
		return
	}
	data, err := h.exportSvc.ExportCSV(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrCompetitionNotFound) {
			writeError(w, http.StatusNotFound, "competition not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "export failed")
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"competition-%d.csv\"", id))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *ExportHandler) ExportPDF(w http.ResponseWriter, r *http.Request) {
	id, err := h.competitionID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid competition id")
		return
	}
	data, err := h.exportSvc.ExportPDF(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrCompetitionNotFound) {
			writeError(w, http.StatusNotFound, "competition not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "export failed")
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"competition-%d.pdf\"", id))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
