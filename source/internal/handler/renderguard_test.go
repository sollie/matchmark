package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sollie/matchmark/internal/handler"
)

func guardRequest(guard http.Handler, target, remote string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.RemoteAddr = remote + ":1234"
	rec := httptest.NewRecorder()
	guard.ServeHTTP(rec, req)
	return rec
}

func TestRenderGuardLimitsPerIPAndResource(t *testing.T) {
	now := time.Unix(1000, 0)
	g := handler.NewRenderGuard(handler.RenderLimits{Concurrent: 4, Queue: 4, QueueWait: time.Second, PerResource: 2, Window: time.Minute, Now: func() time.Time { return now }})
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("png")) })
	guard := g.Middleware(handler.ResourceByPath)(ok)

	for i := 0; i < 2; i++ {
		if rec := guardRequest(guard, "/r/1", "10.0.0.1"); rec.Code != http.StatusOK {
			t.Fatalf("request %d: %d", i, rec.Code)
		}
	}
	rec := guardRequest(guard, "/r/1", "10.0.0.1")
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("over limit: %d %v", rec.Code, rec.Header())
	}
	if rec := guardRequest(guard, "/r/2", "10.0.0.1"); rec.Code != http.StatusOK {
		t.Fatalf("other resource, same ip: %d", rec.Code)
	}
	if rec := guardRequest(guard, "/r/1", "10.0.0.2"); rec.Code != http.StatusOK {
		t.Fatalf("same resource, other ip: %d", rec.Code)
	}
	now = now.Add(time.Minute + time.Second)
	if rec := guardRequest(guard, "/r/1", "10.0.0.1"); rec.Code != http.StatusOK {
		t.Fatalf("after window: %d", rec.Code)
	}
}

func TestRenderGuardCoalescesIdenticalRequests(t *testing.T) {
	g := handler.NewRenderGuard(handler.RenderLimits{Concurrent: 4, Queue: 4, QueueWait: time.Second, PerResource: 100, Window: time.Minute})
	var calls atomic.Int32
	release := make(chan struct{})
	slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		<-release
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("shared"))
	})
	guard := g.Middleware(handler.ResourceByPath)(slow)

	var wg sync.WaitGroup
	recs := make([]*httptest.ResponseRecorder, 3)
	for i := range recs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			recs[i] = guardRequest(guard, "/r/1?format=png", "10.0.0."+strconv.Itoa(i+1))
		}()
	}
	time.Sleep(100 * time.Millisecond)
	close(release)
	wg.Wait()
	if n := calls.Load(); n != 1 {
		t.Fatalf("handler ran %d times", n)
	}
	for _, rec := range recs {
		if rec.Code != http.StatusOK || rec.Body.String() != "shared" || rec.Header().Get("Content-Type") != "image/png" {
			t.Fatalf("response %d %q %v", rec.Code, rec.Body.String(), rec.Header())
		}
	}
}

func TestRenderGuardShedsLoadWhenSaturated(t *testing.T) {
	g := handler.NewRenderGuard(handler.RenderLimits{Concurrent: 1, Queue: 0, QueueWait: 10 * time.Millisecond, PerResource: 100, Window: time.Minute})
	started := make(chan struct{})
	release := make(chan struct{})
	slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/r/1" {
			close(started)
			<-release
		}
		_, _ = w.Write([]byte("ok"))
	})
	guard := g.Middleware(handler.ResourceByPath)(slow)

	done := make(chan *httptest.ResponseRecorder)
	go func() { done <- guardRequest(guard, "/r/1", "10.0.0.1") }()
	<-started
	rec := guardRequest(guard, "/r/2", "10.0.0.2")
	if rec.Code != http.StatusServiceUnavailable || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("saturated: %d %v", rec.Code, rec.Header())
	}
	close(release)
	if first := <-done; first.Code != http.StatusOK {
		t.Fatalf("first: %d", first.Code)
	}
	if rec := guardRequest(guard, "/r/2", "10.0.0.2"); rec.Code != http.StatusOK {
		t.Fatalf("after release: %d", rec.Code)
	}
}

func TestRenderGuardWithoutCoalescingRunsEveryRequestAndSharesLimits(t *testing.T) {
	g := handler.NewRenderGuard(handler.RenderLimits{Concurrent: 2, Queue: 0, QueueWait: 10 * time.Millisecond, PerResource: 100, Window: time.Minute})
	var calls atomic.Int32
	release := make(chan struct{})
	slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		<-release
		_, _ = w.Write([]byte("own"))
	})
	guard := g.MiddlewareUncoalesced(handler.ResourceByPath)(slow)

	var wg sync.WaitGroup
	recs := make([]*httptest.ResponseRecorder, 2)
	for i := range recs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			recs[i] = guardRequest(guard, "/r/1", "10.0.0."+strconv.Itoa(i+1))
		}()
	}
	for calls.Load() < 2 {
		time.Sleep(5 * time.Millisecond)
	}
	if rec := guardRequest(guard, "/r/1", "10.0.0.9"); rec.Code != http.StatusServiceUnavailable || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("saturated: %d %v", rec.Code, rec.Header())
	}
	close(release)
	wg.Wait()
	if n := calls.Load(); n != 2 {
		t.Fatalf("handler ran %d times", n)
	}
	for _, rec := range recs {
		if rec.Code != http.StatusOK || rec.Body.String() != "own" {
			t.Fatalf("response %d %q", rec.Code, rec.Body.String())
		}
	}
}
