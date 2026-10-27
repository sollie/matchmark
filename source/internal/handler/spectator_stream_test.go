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
	CurrentStage int              `json:"current_stage"`
	Standings    []map[string]any `json:"standings"`
	TotalStages  int              `json:"total_stages"`
	Status       string           `json:"status"`
}

func streamPath(f *fixture) string {
	return fmt.Sprintf("/api/spectator/matches/%d/stream", f.comp)
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
	if len(snap.Standings) != 2 || snap.Standings[0]["last_name"] != "BERG" {
		t.Fatalf("standings %v", snap.Standings)
	}
}

func TestSpectatorStreamPushesSnapshotAfterLock(t *testing.T) {
	f := newFixture(t, "150")
	entry := f.addEntry(t, "HANSEN", 1)
	f.score(t, entry, 1, map[string]any{"x": 12})
	f.activate(t)
	s := f.env.OpenStream(streamPath(f), nil)
	first := nextSnapshot(t, s)
	if first.Standings[0]["stages"].([]any)[0].(map[string]any)["accepted"] != false {
		t.Fatalf("standings %v", first.Standings)
	}
	path := fmt.Sprintf("/api/matches/%d/entries/%d/stages/1/lock", f.comp, entry)
	if status, out := f.env.Do("POST", path, nil); status != http.StatusOK && status != http.StatusNoContent {
		t.Fatalf("lock: %d %v", status, out)
	}
	second := nextSnapshot(t, s)
	if second.Standings[0]["stages"].([]any)[0].(map[string]any)["accepted"] != true {
		t.Fatalf("standings %v", second.Standings)
	}
}

func TestSpectatorStreamHonoursVisibilityFlags(t *testing.T) {
	f := spectatorFixture(t)
	s := f.env.OpenStream(streamPath(f), nil)
	if _, ok := nextSnapshot(t, s).Standings[0]["last_name"]; !ok {
		t.Fatal("names should be visible initially")
	}
	visPath := fmt.Sprintf("/api/matches/%d/visibility", f.comp)
	if status, out := f.env.Do("PUT", visPath, map[string]bool{"show_names": false}); status != http.StatusOK {
		t.Fatalf("visibility: %d %v", status, out)
	}
	if _, ok := nextSnapshot(t, s).Standings[0]["last_name"]; ok {
		t.Fatal("names should be hidden after the change")
	}
}

func TestSpectatorStreamHiddenForDraftMatch(t *testing.T) {
	f := newFixture(t, "150")
	s := f.env.OpenStream(streamPath(f), nil)
	if s.Status != http.StatusNotFound {
		t.Fatalf("status %d", s.Status)
	}
}

func TestSpectatorStreamSendsHeartbeat(t *testing.T) {
	env := apitest.NewAdmin(t, handler.WithStreamLimits(50*time.Millisecond, time.Minute, 10, 10, 10))
	f := &fixture{env: env, comp: env.ID("POST", "/api/matches", map[string]any{"name": "M", "date": "2026-05-01", "course": "150"})}
	f.addEntry(t, "HANSEN", 1)
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
	f := &fixture{env: env, comp: env.ID("POST", "/api/matches", map[string]any{"name": "M", "date": "2026-05-01", "course": "150"})}
	f.addEntry(t, "HANSEN", 1)
	f.activate(t)
	s := env.OpenStream(streamPath(f), nil)
	if !s.Closed(eventWait) {
		t.Fatal("stream should end after its lifetime")
	}
}

func TestSpectatorStreamCapsConcurrentStreams(t *testing.T) {
	env := apitest.NewAdmin(t, handler.WithStreamLimits(time.Minute, time.Minute, 1, 1, 1))
	f := &fixture{env: env, comp: env.ID("POST", "/api/matches", map[string]any{"name": "M", "date": "2026-05-01", "course": "150"})}
	f.addEntry(t, "HANSEN", 1)
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
	f := &scoringFixture{fixture: &fixture{env: env, comp: env.ID("POST", "/api/matches", map[string]any{"name": "M", "date": "2026-05-01", "course": "150"})}, entries: map[int]int{}}
	f.entries[1] = f.addEntry(t, "HANSEN", 1)
	f.entries[2] = f.addEntry(t, "BERG", 2)
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
	admin := env.OpenStream(fmt.Sprintf("/api/matches/%d/stream", f.comp), nil)
	if admin.Status != http.StatusOK {
		t.Fatalf("admin status %d", admin.Status)
	}
	if s := openScorerStream(f, token); s.Status != http.StatusOK {
		t.Fatalf("scorer status %d", s.Status)
	}
}

func TestRejectedStreamRequestsConsumeNoCapacity(t *testing.T) {
	env := apitest.NewAdmin(t, handler.WithStreamLimits(time.Minute, time.Minute, 1, 1, 1))
	f := &fixture{env: env, comp: env.ID("POST", "/api/matches", map[string]any{"name": "M", "date": "2026-05-01", "course": "150"})}
	for i := 0; i < 3; i++ {
		if s := env.OpenStream(streamPath(f), nil); s.Status != http.StatusNotFound {
			t.Fatalf("draft status %d", s.Status)
		}
		if s := env.OpenStream("/api/spectator/matches/9999/stream", nil); s.Status != http.StatusNotFound {
			t.Fatalf("unknown status %d", s.Status)
		}
		if s := env.OpenStream("/api/matches/9999/stream", nil); s.Status != http.StatusNotFound {
			t.Fatalf("unknown admin status %d", s.Status)
		}
	}
	f.addEntry(t, "HANSEN", 1)
	f.activate(t)
	if s := env.OpenStream(streamPath(f), nil); s.Status != http.StatusOK {
		t.Fatalf("status %d", s.Status)
	}
}

func TestRejectedStreamRequestsDoNotConsumeCapAtCapacity(t *testing.T) {
	env := apitest.NewAdmin(t, handler.WithStreamLimits(time.Minute, time.Minute, 1, 1, 1))
	f := &fixture{env: env, comp: env.ID("POST", "/api/matches", map[string]any{"name": "M", "date": "2026-05-01", "course": "150"})}
	f.addEntry(t, "HANSEN", 1)
	f.activate(t)
	nextSnapshot(t, env.OpenStream(streamPath(f), nil))
	adminOpen := env.OpenStream(fmt.Sprintf("/api/matches/%d/stream", f.comp), nil)
	if adminOpen.Status != http.StatusOK {
		t.Fatalf("admin status %d", adminOpen.Status)
	}
	if s := env.OpenStream("/api/spectator/matches/9999/stream", nil); s.Status != http.StatusNotFound {
		t.Fatalf("unknown spectator status %d, want 404", s.Status)
	}
	if s := env.OpenStream("/api/matches/9999/stream", nil); s.Status != http.StatusNotFound {
		t.Fatalf("unknown admin status %d, want 404", s.Status)
	}
}

func TestSpectatorSnapshotStageStates(t *testing.T) {
	f := newFixture(t, "150")
	one := f.addEntry(t, "HANSEN", 1)
	two := f.addEntry(t, "BERG", 2)
	f.env.Do("PUT", fmt.Sprintf("/api/matches/%d/visibility", f.comp), map[string]any{"show_names": true, "show_stage_scores": true})
	f.scoreAccepted(t, one, 1, map[string]any{"x": 12})
	f.score(t, two, 1, map[string]any{"x": 12})
	f.activate(t)
	snap := nextSnapshot(t, f.env.OpenStream(streamPath(f), nil))
	if snap.TotalStages != 10 {
		t.Fatalf("total stages %d", snap.TotalStages)
	}
	if snap.CurrentStage != 1 {
		t.Fatalf("current stage %d", snap.CurrentStage)
	}
	if snap.Status != "active" {
		t.Fatalf("status %q", snap.Status)
	}
	states := map[string][]string{}
	for _, row := range snap.Standings {
		var list []string
		for _, st := range row["stages"].([]any) {
			list = append(list, st.(map[string]any)["state"].(string))
		}
		states[row["last_name"].(string)] = list
	}
	if got := states["HANSEN"]; got[0] != "accepted" || got[1] != "not_started" {
		t.Fatalf("hansen %v", got)
	}
	if got := states["BERG"]; got[0] != "in_progress" || got[1] != "not_started" {
		t.Fatalf("berg %v", got)
	}
}
