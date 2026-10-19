package handler

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func WithBackupDir(dir string) Option {
	return func(h *Handler) { h.backupDir = dir }
}

func (h *Handler) backup(w http.ResponseWriter, r *http.Request) {
	path, cleanup, err := h.svc.Backup(r.Context(), h.backupDir)
	if err != nil {
		writeError(w, err)
		return
	}
	defer cleanup()
	f, err := os.Open(path)
	if err != nil {
		writeError(w, err)
		return
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.sqlite3")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="matchmark-%s.db"`, time.Now().UTC().Format("2006-01-02T150405Z")))
	http.ServeContent(w, r, "", info.ModTime(), f)
}
