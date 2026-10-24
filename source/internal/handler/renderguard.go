package handler

import (
	"bytes"
	"context"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	defaultRenderConcurrent  = 4
	defaultRenderQueue       = 8
	defaultRenderQueueWait   = 2 * time.Second
	defaultRenderPerResource = 20
	defaultRenderWindow      = time.Minute
	maxRenderRateKeys        = 100000
	renderTimeout            = 30 * time.Second
)

type RenderLimits struct {
	Concurrent  int
	Queue       int
	QueueWait   time.Duration
	PerResource int
	Window      time.Duration
	Now         func() time.Time
}

func DefaultRenderLimits() RenderLimits {
	return RenderLimits{
		Concurrent:  defaultRenderConcurrent,
		Queue:       defaultRenderQueue,
		QueueWait:   defaultRenderQueueWait,
		PerResource: defaultRenderPerResource,
		Window:      defaultRenderWindow,
	}
}

type RenderGuard struct {
	limits  RenderLimits
	slots   chan struct{}
	waiting atomic.Int64
	flight  singleflight.Group

	mu        sync.Mutex
	rates     map[string]*rateWindow
	lastSweep time.Time
}

type rateWindow struct {
	start time.Time
	count int
}

func NewRenderGuard(limits RenderLimits) *RenderGuard {
	d := DefaultRenderLimits()
	if limits.Concurrent < 1 {
		limits.Concurrent = d.Concurrent
	}
	if limits.Queue < 0 {
		limits.Queue = d.Queue
	}
	if limits.QueueWait <= 0 {
		limits.QueueWait = d.QueueWait
	}
	if limits.PerResource < 1 {
		limits.PerResource = d.PerResource
	}
	if limits.Window <= 0 {
		limits.Window = d.Window
	}
	if limits.Now == nil {
		limits.Now = time.Now
	}
	return &RenderGuard{
		limits: limits,
		slots:  make(chan struct{}, limits.Concurrent),
		rates:  map[string]*rateWindow{},
	}
}

func ResourceByPath(r *http.Request) string {
	return r.URL.Path
}

func (g *RenderGuard) Middleware(resource func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if retry, ok := g.allow(clientKey(r) + "|" + resource(r)); !ok {
				writeRetry(w, http.StatusTooManyRequests, "too many requests, try again shortly", retry)
				return
			}
			rendered := g.coalesce(r, next)
			rendered.writeTo(w)
		})
	}
}

func (g *RenderGuard) coalesce(r *http.Request, next http.Handler) *renderedResponse {
	v, _, _ := g.flight.Do(r.URL.RequestURI(), func() (any, error) {
		if !g.acquire(r.Context()) {
			return busyResponse(g.limits.QueueWait), nil
		}
		defer g.release()
		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), renderTimeout)
		defer cancel()
		rec := &renderRecorder{header: http.Header{}, status: http.StatusOK}
		next.ServeHTTP(rec, r.Clone(ctx))
		return &renderedResponse{status: rec.status, header: rec.header, body: rec.body.Bytes()}, nil
	})
	return v.(*renderedResponse)
}

func (g *RenderGuard) acquire(ctx context.Context) bool {
	select {
	case g.slots <- struct{}{}:
		return true
	default:
	}
	if g.waiting.Add(1) > int64(g.limits.Queue) {
		g.waiting.Add(-1)
		return false
	}
	defer g.waiting.Add(-1)
	timer := time.NewTimer(g.limits.QueueWait)
	defer timer.Stop()
	select {
	case g.slots <- struct{}{}:
		return true
	case <-timer.C:
		return false
	case <-ctx.Done():
		return false
	}
}

func (g *RenderGuard) release() {
	<-g.slots
}

func (g *RenderGuard) allow(key string) (time.Duration, bool) {
	now := g.limits.Now()
	g.mu.Lock()
	defer g.mu.Unlock()
	if now.Sub(g.lastSweep) >= g.limits.Window {
		for k, w := range g.rates {
			if now.Sub(w.start) >= g.limits.Window {
				delete(g.rates, k)
			}
		}
		g.lastSweep = now
	}
	w, ok := g.rates[key]
	if !ok || now.Sub(w.start) >= g.limits.Window {
		if !ok && len(g.rates) >= maxRenderRateKeys {
			return g.limits.Window, false
		}
		w = &rateWindow{start: now}
		g.rates[key] = w
	}
	if w.count >= g.limits.PerResource {
		return w.start.Add(g.limits.Window).Sub(now), false
	}
	w.count++
	return 0, true
}

func writeRetry(w http.ResponseWriter, status int, msg string, retry time.Duration) {
	seconds := int(retry.Seconds() + 0.999)
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	writeJSON(w, status, map[string]string{"error": msg})
}

type renderedResponse struct {
	status int
	header http.Header
	body   []byte
}

func busyResponse(retry time.Duration) *renderedResponse {
	seconds := int(retry.Seconds() + 0.999)
	if seconds < 1 {
		seconds = 1
	}
	return &renderedResponse{
		status: http.StatusServiceUnavailable,
		header: http.Header{"Content-Type": {"application/json"}, "Retry-After": {strconv.Itoa(seconds)}},
		body:   []byte(`{"error":"server busy, try again shortly"}` + "\n"),
	}
}

func (r *renderedResponse) writeTo(w http.ResponseWriter) {
	for k, v := range r.header {
		w.Header()[k] = append([]string(nil), v...)
	}
	w.WriteHeader(r.status)
	_, _ = w.Write(r.body)
}

type renderRecorder struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (r *renderRecorder) Header() http.Header         { return r.header }
func (r *renderRecorder) WriteHeader(status int)      { r.status = status }
func (r *renderRecorder) Write(p []byte) (int, error) { return r.body.Write(p) }
