package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/sollie/matchmark/internal/service"
)

const maxAuditPage = 1_000_000

type auditEntryResponse struct {
	ID       int64  `json:"id"`
	Time     string `json:"time"`
	UserID   *int   `json:"user_id"`
	Username string `json:"username"`
	Action   string `json:"action"`
	Target   string `json:"target"`
	Details  string `json:"details"`
	IP       string `json:"ip"`
}

func queryInt(r *http.Request, name string, fallback int) (int, error) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > maxAuditPage {
		return 0, service.ErrInvalid
	}
	return n, nil
}

func queryDate(r *http.Request, name string) (time.Time, error) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.DateOnly, v)
	if err != nil {
		return time.Time{}, service.ErrInvalid
	}
	return t, nil
}

func (h *Handler) listAudit(w http.ResponseWriter, r *http.Request) {
	page, err := queryInt(r, "page", 1)
	if err != nil {
		writeError(w, err)
		return
	}
	pageSize, err := queryInt(r, "page_size", service.AuditDefaultPage)
	if err != nil {
		writeError(w, err)
		return
	}
	from, err := queryDate(r, "from")
	if err != nil {
		writeError(w, err)
		return
	}
	to, err := queryDate(r, "to")
	if err != nil {
		writeError(w, err)
		return
	}
	if !to.IsZero() {
		to = to.AddDate(0, 0, 1)
	}
	result, err := h.audit.List(r.Context(), service.AuditQuery{
		Username: r.URL.Query().Get("user"),
		Action:   r.URL.Query().Get("action"),
		From:     from,
		To:       to,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	entries := make([]auditEntryResponse, 0, len(result.Entries))
	for _, e := range result.Entries {
		entries = append(entries, auditEntryResponse{
			ID: e.ID, Time: e.CreatedAt.Format(time.RFC3339), UserID: e.UserID, Username: e.Username,
			Action: e.Action, Target: e.Target, Details: e.Details, IP: e.IP,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"entries": entries, "total": result.Total, "page": result.Page, "page_size": result.PageSize,
	})
}
