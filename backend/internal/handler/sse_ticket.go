package handler

import (
	"net/http"

	"github.com/sollie/matchmark/internal/auth"
	"github.com/sollie/matchmark/internal/service"
)

type SSETicketHandler struct {
	svc *service.SSETicketService
}

func NewSSETicketHandler(svc *service.SSETicketService) *SSETicketHandler {
	return &SSETicketHandler{svc: svc}
}

func (h *SSETicketHandler) Issue(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	ticket, err := h.svc.Issue(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue ticket")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ticket": ticket})
}
