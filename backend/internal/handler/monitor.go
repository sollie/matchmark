package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sollie/matchmark/internal/service"
)

type MonitorHandler struct {
	monitorSvc *service.MonitorService
	ticketSvc  *service.SSETicketService
}

func NewMonitorHandler(monitorSvc *service.MonitorService, ticketSvc *service.SSETicketService) *MonitorHandler {
	return &MonitorHandler{monitorSvc: monitorSvc, ticketSvc: ticketSvc}
}

func (h *MonitorHandler) Stream(w http.ResponseWriter, r *http.Request) {
	ticket := r.URL.Query().Get("ticket")
	if ticket == "" {
		writeError(w, http.StatusUnauthorized, "missing ticket")
		return
	}
	if _, err := h.ticketSvc.Consume(r.Context(), ticket); err != nil {
		if errors.Is(err, service.ErrInvalidTicket) {
			writeError(w, http.StatusUnauthorized, "invalid or expired ticket")
			return
		}
		writeError(w, http.StatusInternalServerError, "ticket validation failed")
		return
	}

	idStr := r.URL.Query().Get("competition_id")
	if idStr == "" {
		writeError(w, http.StatusBadRequest, "competition_id query parameter is required")
		return
	}
	competitionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || competitionID < 1 {
		writeError(w, http.StatusBadRequest, "invalid competition_id")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	emit := func() error {
		snapshot, err := h.monitorSvc.GetSnapshot(r.Context(), competitionID)
		if err != nil {
			if errors.Is(err, service.ErrCompetitionNotFound) {
				_, _ = fmt.Fprintf(w, "event: error\ndata: competition not found\n\n")
				flusher.Flush()
				return err
			}
			return err
		}
		data, err := json.Marshal(snapshot)
		if err != nil {
			return fmt.Errorf("marshal snapshot: %w", err)
		}
		_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return nil
	}

	if err := emit(); err != nil {
		return
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if err := emit(); err != nil {
				return
			}
		}
	}
}
