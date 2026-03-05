package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/sollie/matchmark/internal/service"
)

type ExportHandler struct {
	exportSvc *service.ExportService
}

func NewExportHandler(exportSvc *service.ExportService) *ExportHandler {
	return &ExportHandler{exportSvc: exportSvc}
}

func (h *ExportHandler) ExportJSON(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r, "id")
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
	id, err := parseIDParam(r, "id")
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
	id, err := parseIDParam(r, "id")
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
