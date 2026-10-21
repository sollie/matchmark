package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/sollie/matchmark/internal/apitest"
	"github.com/sollie/matchmark/internal/handler"
)

const eventWait = 3 * time.Second

type snapshotBody struct {
	Standings []map[string]any `json:"standings"`
	Stages    []map[string]any `json:"stages"`
}

func streamPath(f *fixture) string {
	return fmt.Sprintf("/api/spectator/competitions/%d/stream", f.comp)
}

func nextSnapshot(t *testing.T, s *apitest.Stream) snapshotBody {
	t.Helper()
	for {
		ev, ok := s.Next(eventWait)
		if !ok {
			t.Fatal("no snapshot received")
		}
		if ev.Name != "snapshot" {
			continue
		}
		var body snapshotBody
		if err := json.Unmarshal([]byte(ev.Data), &body); err != nil {
			t.Fatalf("decode snapshot: %v", err)
		}
		return body
	}
}

func TestSpectatorStreamSendsSnapshotOnConnect(t *testing.T) {
	f := spectatorFixture(t)
	s := f.env.OpenStream(streamPath(f), nil)
	if s.Status != http.StatusOK {
		t.Fatalf("status %d", s.Status)
	}
	if got := s.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content type %q", got)
	}
	if got := s.Header.Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("cache control %q", got)
	}
	if got := s.Header.Get("X-Accel-Buffering"); got != "no" {
		t.Fatalf("x-accel-buffering %q", got)
	}
	snap := nextSnapshot(t, s)
	if len(snap.Standings) != 2 || snap.Standings[0]["last_name"] != "Berg" {
		t.Fatalf("standings %v", snap.Standings)
	}
	if len(snap.Stages) != 10 {
		t.Fatalf("stages %v", snap.Stages)
	}
}

func TestSpectatorStreamPushesSnapshotAfterLock(t *testing.T) {
	f := newFixture(t, "150")
	entry := f.addEntry(t, "Hansen", 1)
	f.score(t, entry, 1, map[string]any{"x": 12})
	f.activate(t)
	s := f.env.OpenStream(streamPath(f), nil)
	first := nextSnapshot(t, s)
	if first.Stages[0]["locked"].(float64) != 0 {
		t.Fatalf("stages %v", first.Stages)
	}
	path := fmt.Sprintf("/api/competitions/%d/entries/%d/stages/1/lock", f.comp, entry)
	if status, out := f.env.Do("POST", path, nil); status != http.StatusOK && status != http.StatusNoContent {
		t.Fatalf("lock: %d %v", status, out)
	}
	second := nextSnapshot(t, s)
	if second.Stages[0]["locked"].(float64) != 1 {
		t.Fatalf("stages %v", second.Stages)
	}
}

func TestSpectatorStreamHonoursVisibilityFlags(t *testing.T) {
	f := spectatorFixture(t)
	s := f.env.OpenStream(streamPath(f), nil)
	if _, ok := nextSnapshot(t, s).Standings[0]["last_name"]; !ok {
		t.Fatal("names should be visible initially")
	}
	visPath := fmt.Sprintf("/api/competitions/%d/visibility", f.comp)
	if status, out := f.env.Do("PUT", visPath, map[string]bool{"show_names": false}); status != http.StatusOK {
		t.Fatalf("visibility: %d %v", status, out)
	}
	if _, ok := nextSnapshot(t, s).Standings[0]["last_name"]; ok {
		t.Fatal("names should be hidden after the change")
	}
}

func TestSpectatorStreamHiddenForDraftCompetition(t *testing.T) {
	f := newFixture(t, "150")
	s := f.env.OpenStream(streamPath(f), nil)
	if s.Status != http.StatusNotFound {
		t.Fatalf("status %d", s.Status)
	}
}

func TestSpectatorStreamSendsHeartbeat(t *testing.T) {
	env := apitest.NewAdmin(t, handler.WithStreamLimits(50*time.Millisecond, time.Minute, 10, 10, 10))
	f := &fixture{env: env, comp: env.ID("POST", "/api/competitions", map[string]any{"name": "M", "date": "2026-05-01", "course": "150"})}
	f.addEntry(t, "Hansen", 1)
	f.activate(t)
	s := env.OpenStream(streamPath(f), nil)
	for {
		ev, ok := s.Next(eventWait)
		if !ok {
			t.Fatal("no heartbeat")
		}
		if ev.Comment == "heartbeat" {
			return
		}
	}
}

func TestSpectatorStreamEndsAtLifetime(t *testing.T) {
	env := apitest.NewAdmin(t, handler.WithStreamLimits(time.Minute, 100*time.Millisecond, 10, 10, 10))
	f := &fixture{env: env, comp: env.ID("POST", "/api/competitions", map[string]any{"name": "M", "date": "2026-05-01", "course": "150"})}
	f.addEntry(t, "Hansen", 1)
	f.activate(t)
	s := env.OpenStream(streamPath(f), nil)
	if !s.Closed(eventWait) {
		t.Fatal("stream should end after its lifetime")
	}
}

func TestSpectatorStreamCapsConcurrentStreams(t *testing.T) {
	env := apitest.NewAdmin(t, handler.WithStreamLimits(time.Minute, time.Minute, 1, 1, 1))
	f := &fixture{env: env, comp: env.ID("POST", "/api/competitions", map[string]any{"name": "M", "date": "2026-05-01", "course": "150"})}
	f.addEntry(t, "Hansen", 1)
	f.activate(t)
	first := env.OpenStream(streamPath(f), nil)
	nextSnapshot(t, first)
	second := env.OpenStream(streamPath(f), nil)
	if second.Status != http.StatusServiceUnavailable {
		t.Fatalf("status %d", second.Status)
	}
	first.Close()
	deadline := time.Now().Add(eventWait)
	for time.Now().Before(deadline) {
		if env.OpenStream(streamPath(f), nil).Status == http.StatusOK {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("slot was not released after the stream closed")
}

func TestSpectatorFloodDoesNotBlockAdminOrScorerStreams(t *testing.T) {
	env := apitest.NewAdmin(t, handler.WithStreamLimits(time.Minute, time.Minute, 1, 1, 1))
	f := &scoringFixture{fixture: &fixture{env: env, comp: env.ID("POST", "/api/competitions", map[string]any{"name": "M", "date": "2026-05-01", "course": "150"})}, entries: map[int]int{}}
	f.entries[1] = f.addEntry(t, "Hansen", 1)
	f.entries[2] = f.addEntry(t, "Berg", 2)
	f.activate(t)
	token := f.pair(t, 1)
	nextSnapshot(t, env.OpenStream(streamPath(f.fixture), nil))
	second := env.OpenStream(streamPath(f.fixture), nil)
	if second.Status != http.StatusServiceUnavailable {
		t.Fatalf("spectator status %d", second.Status)
	}
	if got := second.Header.Get("Retry-After"); got == "" {
		t.Fatal("missing Retry-After")
	}
	admin := env.OpenStream(fmt.Sprintf("/api/competitions/%d/stream", f.comp), nil)
	if admin.Status != http.StatusOK {
		t.Fatalf("admin status %d", admin.Status)
	}
	if s := openScorerStream(f, token); s.Status != http.StatusOK {
		t.Fatalf("scorer status %d", s.Status)
	}
}

func TestRejectedStreamRequestsConsumeNoCapacity(t *testing.T) {
	env := apitest.NewAdmin(t, handler.WithStreamLimits(time.Minute, time.Minute, 1, 1, 1))
	f := &fixture{env: env, comp: env.ID("POST", "/api/competitions", map[string]any{"name": "M", "date": "2026-05-01", "course": "150"})}
	for i := 0; i < 3; i++ {
		if s := env.OpenStream(streamPath(f), nil); s.Status != http.StatusNotFound {
			t.Fatalf("draft status %d", s.Status)
		}
		if s := env.OpenStream("/api/spectator/competitions/9999/stream", nil); s.Status != http.StatusNotFound {
			t.Fatalf("unknown status %d", s.Status)
		}
		if s := env.OpenStream("/api/competitions/9999/stream", nil); s.Status != http.StatusNotFound {
			t.Fatalf("unknown admin status %d", s.Status)
		}
	}
	f.addEntry(t, "Hansen", 1)
	f.activate(t)
	if s := env.OpenStream(streamPath(f), nil); s.Status != http.StatusOK {
		t.Fatalf("status %d", s.Status)
	}
}
