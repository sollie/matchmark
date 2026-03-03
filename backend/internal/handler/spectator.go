package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sollie/matchmark/internal/service"
)

type SpectatorHandler struct {
	spectatorSvc *service.SpectatorService
}

func NewSpectatorHandler(spectatorSvc *service.SpectatorService) *SpectatorHandler {
	return &SpectatorHandler{spectatorSvc: spectatorSvc}
}

func (h *SpectatorHandler) ListCompetitions(w http.ResponseWriter, r *http.Request) {
	comps, err := h.spectatorSvc.GetActiveCompetitions(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list competitions")
		return
	}
	writeJSON(w, http.StatusOK, comps)
}

func (h *SpectatorHandler) Stream(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	competitionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || competitionID < 1 {
		writeError(w, http.StatusBadRequest, "invalid competition id")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	snapshot, err := h.spectatorSvc.GetSnapshot(r.Context(), competitionID)
	if err != nil {
		if errors.Is(err, service.ErrSpectatorNotAvailable) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.Header().Set("X-Accel-Buffering", "no")
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprintf(w, "event: error\ndata: competition not found or not available for public viewing\n\n")
			flusher.Flush()
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load competition")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	emit := func() error {
		snap, err := h.spectatorSvc.GetSnapshot(r.Context(), competitionID)
		if err != nil {
			if errors.Is(err, service.ErrSpectatorNotAvailable) {
				_, _ = fmt.Fprintf(w, "event: error\ndata: competition not found or not available for public viewing\n\n")
				flusher.Flush()
				return err
			}
			return err
		}
		data, err := json.Marshal(snap)
		if err != nil {
			return fmt.Errorf("marshal snapshot: %w", err)
		}
		_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return nil
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()

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
