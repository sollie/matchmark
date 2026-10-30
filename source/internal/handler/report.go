package handler

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/sollie/matchmark/internal/service"
)

const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

func (h *Handler) resultsWorkbook(w http.ResponseWriter, r *http.Request) {
	h.download(w, r, "xlsx", xlsxContentType, h.svc.ResultsWorkbook)
}

func (h *Handler) resultsCSV(w http.ResponseWriter, r *http.Request) {
	h.download(w, r, "csv", "text/csv; charset=utf-8", h.svc.ResultsCSV)
}

func contentDisposition(baseName, ext string) string {
	name := baseName + "." + ext
	var ascii strings.Builder
	for _, r := range name {
		if r < 0x80 {
			ascii.WriteRune(r)
		} else {
			ascii.WriteByte('_')
		}
	}
	header := `attachment; filename="` + ascii.String() + `"`
	if ascii.String() != name {
		header += "; filename*=UTF-8''" + url.PathEscape(name)
	}
	return header
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request, ext, contentType string, build func(ctx context.Context, id int64) (service.Download, error)) {
	id, err := pathInt(r, "matchID")
	if err != nil {
		writeError(w, err)
		return
	}
	file, err := build(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", contentDisposition(file.BaseName, ext))
	_, _ = w.Write(file.Data)
}

func requireFormat(r *http.Request, want string) error {
	if r.URL.Query().Has("format") && r.URL.Query().Get("format") != want {
		return service.ErrInvalid
	}
	return nil
}

func writeRendered(w http.ResponseWriter, contentType, ext string, file service.Download, attachment bool) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if attachment {
		w.Header().Set("Content-Disposition", contentDisposition(file.BaseName, ext))
	}
	_, _ = w.Write(file.Data)
}
