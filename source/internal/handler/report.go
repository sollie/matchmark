package handler

import (
	"context"
	"fmt"
	"net/http"
)

func (h *Handler) resultsWorkbook(w http.ResponseWriter, r *http.Request) {
	h.download(w, r, "xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", h.svc.ResultsWorkbook)
}

func (h *Handler) resultsCSV(w http.ResponseWriter, r *http.Request) {
	h.download(w, r, "csv", "text/csv; charset=utf-8", h.svc.ResultsCSV)
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request, ext, contentType string, build func(ctx context.Context, id int64) ([]byte, error)) {
	id, err := pathInt(r, "competitionID")
	if err != nil {
		writeError(w, err)
		return
	}
	data, err := build(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="results-%d.%s"`, id, ext))
	_, _ = w.Write(data)
}
