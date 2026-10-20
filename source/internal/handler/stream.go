package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sollie/matchmark/internal/service"
)

const (
	defaultHeartbeat    = 25 * time.Second
	defaultStreamLife   = 30 * time.Minute
	defaultMaxStreams   = 500
	streamRetryAfterSec = "5"
)

type streamLimits struct {
	heartbeat time.Duration
	lifetime  time.Duration
	max       int64
}

func WithStreamLimits(heartbeat, lifetime time.Duration, maxStreams int) Option {
	return func(h *Handler) {
		h.limits = streamLimits{heartbeat: heartbeat, lifetime: lifetime, max: int64(maxStreams)}
	}
}

type streamEvent struct {
	Name  string
	Data  any
	Final bool
}

type streamSnapshot func(ctx context.Context) (streamEvent, error)

func (h *Handler) serveStream(w http.ResponseWriter, r *http.Request, sub *service.Subscription, snapshot streamSnapshot) {
	defer sub.Close()
	if h.streams.Add(1) > h.limits.max {
		h.streams.Add(-1)
		w.Header().Set("Retry-After", streamRetryAfterSec)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "too many streams"})
		return
	}
	defer h.streams.Add(-1)

	first, err := snapshot(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	hdr := w.Header()
	hdr.Set("Content-Type", "text/event-stream")
	hdr.Set("Cache-Control", "no-cache")
	hdr.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	rc := http.NewResponseController(w)
	if !writeEvent(w, rc, first) || first.Final {
		return
	}

	heartbeat := time.NewTicker(h.limits.heartbeat)
	defer heartbeat.Stop()
	lifetime := time.NewTimer(h.limits.lifetime)
	defer lifetime.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-lifetime.C:
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil || rc.Flush() != nil {
				return
			}
		case <-sub.C:
			ev, err := snapshot(r.Context())
			if err != nil || !writeEvent(w, rc, ev) || ev.Final {
				return
			}
		}
	}
}

func writeEvent(w http.ResponseWriter, rc *http.ResponseController, ev streamEvent) bool {
	data, err := json.Marshal(ev.Data)
	if err != nil {
		return false
	}
	name := strings.NewReplacer("\n", "", "\r", "").Replace(ev.Name)
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, data); err != nil {
		return false
	}
	return rc.Flush() == nil
}
