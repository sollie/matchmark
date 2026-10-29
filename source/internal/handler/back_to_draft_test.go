package handler_test

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/sollie/matchmark/internal/apitest"
)

func (f *fixture) backToDraft() int {
	status, _ := f.env.Do("POST", fmt.Sprintf("/api/matches/%d/back-to-draft", f.comp), nil)
	return status
}

func (f *fixture) activatedAt(t *testing.T) string {
	t.Helper()
	_, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d", f.comp), nil)
	return out["activated_at"].(string)
}

func TestBackToDraftRefusedWhenAnyEntryHasHits(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, f *scoringFixture)
	}{
		{"unlocked hits", func(t *testing.T, f *scoringFixture) {
			f.score(t, f.entries[1], 1, map[string]any{"x": 24})
		}},
		{"locked hits", func(t *testing.T, f *scoringFixture) {
			f.scoreAccepted(t, f.entries[1], 1, map[string]any{"x": 24})
		}},
		{"withdrawn entry with hits", func(t *testing.T, f *scoringFixture) {
			f.score(t, f.entries[1], 1, map[string]any{"x": 24})
			f.env.Do("POST", fmt.Sprintf("/api/matches/%d/entries/%d/withdraw", f.comp, f.entries[1]), nil)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newScoringFixture(t, 1, 2)
			tt.setup(t, f)
			if status := f.backToDraft(); status != http.StatusConflict {
				t.Fatalf("status %d", status)
			}
			_, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d", f.comp), nil)
			if out["status"] != "active" {
				t.Fatalf("status changed: %v", out["status"])
			}
		})
	}
}

func TestBackToDraftRequiresActiveMatchAndAdmin(t *testing.T) {
	f := newFixture(t, "150")
	f.addEntry(t, "One", 1)
	if status := f.backToDraft(); status != http.StatusConflict {
		t.Fatalf("draft: %d", status)
	}
	f.activate(t)
	f.env.Do("POST", fmt.Sprintf("/api/matches/%d/status", f.comp), map[string]any{"status": "finished"})
	if status := f.backToDraft(); status != http.StatusConflict {
		t.Fatalf("finished: %d", status)
	}
	if status, _ := apitest.New(t).Do("POST", fmt.Sprintf("/api/matches/%d/back-to-draft", f.comp), nil); status != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", status)
	}
}

func TestBackToDraftBehavesLikeDraft(t *testing.T) {
	f := newScoringFixture(t, 1, 3)
	token := f.pair(t, 1)
	first := f.activatedAt(t)
	audited := auditEntries(t, f.env, "?action=match.back_to_draft")
	if len(audited) != 0 {
		t.Fatalf("audit before: %v", audited)
	}

	if status := f.backToDraft(); status != http.StatusOK {
		t.Fatalf("back to draft: %d", status)
	}

	if status, out := f.env.DoToken(token, "GET", "/api/scorer/session", nil); status != http.StatusOK || out["phase"] != "not_started" {
		t.Fatalf("scorer session: %d %v", status, out)
	}
	if status, _ := f.env.DoToken(token, "PUT", "/api/scorer/segments/1", counts(12, 0)); status != http.StatusConflict {
		t.Fatalf("scorer hits status %d", status)
	}
	if _, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d/pairings", f.comp), nil); len(out["pairings"].([]any)) != 1 {
		t.Fatalf("pairings: %v", out)
	}
	if status, _ := publicGet(t, f.fixture, standingsPath(f.fixture)); status != http.StatusNotFound {
		t.Fatalf("spectator status %d", status)
	}

	f.entries[2] = f.addEntry(t, "LATE", 2)
	if status, _ := f.env.Do("PUT", fmt.Sprintf("/api/matches/%d/entries/%d", f.comp, f.entries[2]), map[string]any{"category": "Revolver", "lane": 4}); status != http.StatusOK {
		t.Fatalf("lane edit: %d", status)
	}
	if status, _ := f.env.Do("PUT", fmt.Sprintf("/api/matches/%d", f.comp), map[string]any{"name": "Club Match", "date": "2026-05-01", "course": "60"}); status != http.StatusConflict {
		t.Fatalf("course change status %d", status)
	}

	if got := auditEntries(t, f.env, "?action=match.back_to_draft"); len(got) != 1 {
		t.Fatalf("audit: %v", got)
	}

	time.Sleep(5 * time.Millisecond)
	f.activate(t)
	if second := f.activatedAt(t); second <= first {
		t.Fatalf("restart time %s not after %s", second, first)
	}
	if status, out := f.env.DoToken(token, "GET", "/api/scorer/session", nil); status != http.StatusOK || out["phase"] == "not_started" || int(out["score_lane"].(float64)) != 3 {
		t.Fatalf("restarted session: %d %v", status, out)
	}
}

func TestBackToDraftKeepsScreenAssignment(t *testing.T) {
	env := apitest.NewAdmin(t)
	token, code := registerScreen(t, env)
	_, screen := env.Do("POST", "/api/screens/approve", map[string]any{"code": code})
	id := strconv.Itoa(int(screen["id"].(float64)))
	match := env.ID("POST", "/api/matches", map[string]any{"name": "Cup", "date": env.Today(), "course": "40"})
	env.Do("PUT", "/api/screens/"+id, map[string]any{"match_id": match})
	env.Do("POST", "/api/matches/"+strconv.Itoa(match)+"/status", map[string]any{"status": "active"})

	s := openScreenStream(env, token)
	if state, mid, _ := nextScreenState(t, s); state != "match" || mid != float64(match) {
		t.Fatalf("active: %s %v", state, mid)
	}
	if status, _ := env.Do("POST", "/api/matches/"+strconv.Itoa(match)+"/back-to-draft", nil); status != http.StatusOK {
		t.Fatalf("back to draft: %d", status)
	}
	if state, mid, _ := nextScreenState(t, s); state != "waiting" || mid != nil {
		t.Fatalf("draft: %s %v", state, mid)
	}
	_, got := env.Do("GET", "/api/screens", nil)
	if list := got["screens"].([]any); len(list) != 1 || list[0].(map[string]any)["match_id"] != float64(match) {
		t.Fatalf("assignment lost: %v", got)
	}
	env.Do("POST", "/api/matches/"+strconv.Itoa(match)+"/status", map[string]any{"status": "active"})
	if state, mid, _ := nextScreenState(t, s); state != "match" || mid != float64(match) {
		t.Fatalf("restarted: %s %v", state, mid)
	}
}

func TestBackToDraftRaceWithScorerHit(t *testing.T) {
	for i := 0; i < 20; i++ {
		f := newScoringFixture(t, 1, 3)
		token := f.pair(t, 1)
		var hit, back int
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			hit, _ = f.env.DoToken(token, "PUT", "/api/scorer/segments/1", counts(12, 0))
		}()
		go func() {
			defer wg.Done()
			back = f.backToDraft()
		}()
		wg.Wait()

		_, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d", f.comp), nil)
		switch {
		case back == http.StatusOK && hit == http.StatusConflict && out["status"] == "draft":
		case back == http.StatusConflict && hit == http.StatusOK && out["status"] == "active":
		default:
			t.Fatalf("iteration %d: back %d hit %d status %v", i, back, hit, out["status"])
		}
	}
}
