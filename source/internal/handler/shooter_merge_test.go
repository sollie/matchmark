package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/sollie/matchmark/internal/apitest"
)

type mergeFixture struct {
	env *apitest.Env
}

func (m *mergeFixture) shooter(t *testing.T, last, club, waid string) int {
	t.Helper()
	body := map[string]any{"last_name": last, "first_name": "Ann", "country": "NOR", "club": club}
	if waid != "" {
		body["waid"] = waid
	}
	return m.env.ID("POST", "/api/shooters", body)
}

func (m *mergeFixture) match(t *testing.T, name string) int {
	t.Helper()
	return m.env.ID("POST", "/api/matches", map[string]any{"name": name, "date": "2026-05-01", "course": "150"})
}

func (m *mergeFixture) enter(t *testing.T, match, shooter, lane int) int {
	t.Helper()
	return m.env.ID("POST", fmt.Sprintf("/api/matches/%d/entries", match), map[string]any{"shooter_id": shooter, "category": "Revolver", "lane": lane})
}

func mergePaths(source int) (preview, execute string) {
	return fmt.Sprintf("/api/shooters/%d/merge-preview", source), fmt.Sprintf("/api/shooters/%d/merge", source)
}

func newMergeFixture(t *testing.T) *mergeFixture {
	return &mergeFixture{env: apitest.NewAdmin(t)}
}

func TestMergeMovesEntriesAndDeletesSource(t *testing.T) {
	m := newMergeFixture(t)
	source := m.shooter(t, "DUPE", "Old Club", "W900")
	target := m.shooter(t, "REAL", "Real Club", "")
	spring, autumn := m.match(t, "Spring"), m.match(t, "Autumn")
	sourceEntry := m.enter(t, spring, source, 1)
	m.enter(t, autumn, target, 1)

	preview, execute := mergePaths(source)
	status, out := m.env.Do("GET", fmt.Sprintf("%s?target_id=%d", preview, target), nil)
	if status != http.StatusOK || out["can_merge"] != true || out["resulting_waid"] != "W900" ||
		len(out["entries"].([]any)) != 1 || len(out["conflicts"].([]any)) != 0 {
		t.Fatalf("preview %d %v", status, out)
	}

	status, out = m.env.Do("POST", execute, map[string]any{"target_id": target})
	if status != http.StatusOK || out["moved_entries"] != float64(1) {
		t.Fatalf("merge %d %v", status, out)
	}

	if status, _ := m.env.Do("GET", fmt.Sprintf("/api/shooters/%d", source), nil); status != http.StatusNotFound {
		t.Fatalf("source still exists: %d", status)
	}
	_, detail := m.env.Do("GET", fmt.Sprintf("/api/shooters/%d", target), nil)
	if detail["entry_count"] != float64(2) || detail["last_name"] != "REAL" || detail["club"] != "Real Club" || detail["waid"] != "W900" {
		t.Fatalf("target %v", detail)
	}
	_, entries := m.env.Do("GET", fmt.Sprintf("/api/matches/%d/entries", spring), nil)
	if got := entries["entries"].([]any)[0].(map[string]any); got["id"] != float64(sourceEntry) {
		t.Fatalf("entry %v", got)
	}

	row := findAudit(auditEntries(t, m.env, ""), "shooter.merge")
	if row == nil || row["target"] != fmt.Sprintf("shooter:%d", source) || row["details"] != fmt.Sprintf("into shooter:%d", target) {
		t.Fatalf("audit %v", row)
	}
}

func TestMergeKeepsTargetWAID(t *testing.T) {
	m := newMergeFixture(t)
	source := m.shooter(t, "DUPE", "Club", "")
	target := m.shooter(t, "REAL", "Club", "W1")
	_, execute := mergePaths(source)
	if status, out := m.env.Do("POST", execute, map[string]any{"target_id": target}); status != http.StatusOK {
		t.Fatalf("merge %d %v", status, out)
	}
	_, detail := m.env.Do("GET", fmt.Sprintf("/api/shooters/%d", target), nil)
	if detail["waid"] != "W1" {
		t.Fatalf("target %v", detail)
	}
}

func TestMergeRefusedForDifferentWAIDs(t *testing.T) {
	m := newMergeFixture(t)
	source := m.shooter(t, "ONE", "Club", "W1")
	target := m.shooter(t, "TWO", "Club", "W2")
	preview, execute := mergePaths(source)

	_, out := m.env.Do("GET", fmt.Sprintf("%s?target_id=%d", preview, target), nil)
	if out["can_merge"] != false || out["waid_conflict"] != true {
		t.Fatalf("preview %v", out)
	}
	status, out := m.env.Do("POST", execute, map[string]any{"target_id": target})
	if status != http.StatusConflict || out["error"] != "merge_blocked" {
		t.Fatalf("merge %d %v", status, out)
	}
	if status, _ := m.env.Do("GET", fmt.Sprintf("/api/shooters/%d", source), nil); status != http.StatusOK {
		t.Fatalf("source gone: %d", status)
	}
}

func TestMergeRefusedWhenBothEnteredInSameMatch(t *testing.T) {
	m := newMergeFixture(t)
	source := m.shooter(t, "DUPE", "Club", "")
	target := m.shooter(t, "REAL", "Club", "")
	spring := m.match(t, "Spring")
	m.enter(t, spring, source, 1)
	m.enter(t, spring, target, 2)
	preview, execute := mergePaths(source)

	_, out := m.env.Do("GET", fmt.Sprintf("%s?target_id=%d", preview, target), nil)
	conflicts := out["conflicts"].([]any)
	if out["can_merge"] != false || len(conflicts) != 1 || conflicts[0].(map[string]any)["match_id"] != float64(spring) {
		t.Fatalf("preview %v", out)
	}
	status, out := m.env.Do("POST", execute, map[string]any{"target_id": target})
	if status != http.StatusConflict || out["error"] != "merge_blocked" || len(out["conflicts"].([]any)) != 1 {
		t.Fatalf("merge %d %v", status, out)
	}
	_, detail := m.env.Do("GET", fmt.Sprintf("/api/shooters/%d", target), nil)
	if detail["entry_count"] != float64(1) {
		t.Fatalf("target changed %v", detail)
	}
}

func TestMergeRejectsSelfAndUnknownTarget(t *testing.T) {
	m := newMergeFixture(t)
	source := m.shooter(t, "DUPE", "Club", "")
	_, execute := mergePaths(source)
	if status, _ := m.env.Do("POST", execute, map[string]any{"target_id": source}); status != http.StatusUnprocessableEntity {
		t.Fatalf("self merge: %d", status)
	}
	if status, _ := m.env.Do("POST", execute, map[string]any{"target_id": 9999}); status != http.StatusNotFound {
		t.Fatalf("unknown target: %d", status)
	}
}

func TestMergeRefreshesStoredNamesOnFinishedMatch(t *testing.T) {
	f := newFixture(t, "150")
	source := f.env.ID("POST", "/api/shooters", map[string]any{"last_name": "DUPE", "first_name": "Ann", "country": "NOR", "club": "Club"})
	target := f.env.ID("POST", "/api/shooters", map[string]any{"last_name": "REAL", "first_name": "Bea", "country": "NOR", "club": "Club"})
	entry := f.env.ID("POST", fmt.Sprintf("/api/matches/%d/entries", f.comp), map[string]any{"shooter_id": source, "category": "Revolver", "lane": 1})
	f.activate(t)
	f.scoreAccepted(t, entry, 1, counts(10, 2))
	f.finishActive(t)

	if status, out := f.env.Do("POST", fmt.Sprintf("/api/shooters/%d/merge", source), map[string]any{"target_id": target}); status != http.StatusOK {
		t.Fatalf("merge %d %v", status, out)
	}
	events := f.env.StageEvents(f.comp)
	if len(events) == 0 {
		t.Fatal("no stage events")
	}
	for _, e := range events {
		if e.ShooterName != "Bea REAL" {
			t.Fatalf("stored name %q", e.ShooterName)
		}
	}
}
