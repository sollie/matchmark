package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func (f *fixture) putEntry(entry int, body map[string]any) (int, map[string]any) {
	return f.env.Do("PUT", fmt.Sprintf("/api/matches/%d/entries/%d", f.comp, entry), body)
}

func resultRemarks(t *testing.T, f *fixture) []any {
	t.Helper()
	status, out := publicGet(t, f, resultsPath(f))
	if status != http.StatusOK {
		t.Fatalf("results: %d %v", status, out)
	}
	var remarks []any
	for _, c := range out["categories"].([]any) {
		for _, r := range c.(map[string]any)["rows"].([]any) {
			remarks = append(remarks, r.(map[string]any)["remarks"])
		}
	}
	return remarks
}

func TestEntryRemarksAreTrimmedSetAndCleared(t *testing.T) {
	f := newFixture(t, "150")
	entry := f.addEntry(t, "One", 1)
	status, out := f.putEntry(entry, map[string]any{"category": "Revolver 1500", "lane": 1, "remarks": "  Late arrival  "})
	if status != http.StatusOK || out["remarks"] != "Late arrival" {
		t.Fatalf("set: %d %v", status, out)
	}
	status, out = f.putEntry(entry, map[string]any{"category": "Revolver 1500", "lane": 1})
	if status != http.StatusOK || out["remarks"] != "Late arrival" {
		t.Fatalf("omitted keeps: %d %v", status, out)
	}
	status, out = f.putEntry(entry, map[string]any{"category": "Revolver 1500", "lane": 1, "remarks": "   "})
	if status != http.StatusOK || out["remarks"] != nil {
		t.Fatalf("clear: %d %v", status, out)
	}
}

func TestEntryRemarksLimitIs200Characters(t *testing.T) {
	f := newFixture(t, "150")
	entry := f.addEntry(t, "One", 1)
	ok := strings.Repeat("å", 200)
	if status, out := f.putEntry(entry, map[string]any{"category": "Revolver 1500", "lane": 1, "remarks": ok}); status != http.StatusOK {
		t.Fatalf("200: %d %v", status, out)
	}
	if status, _ := f.putEntry(entry, map[string]any{"category": "Revolver 1500", "lane": 1, "remarks": ok + "x"}); status != http.StatusUnprocessableEntity {
		t.Fatalf("201: %d", status)
	}
	shooter := f.env.ID("POST", "/api/shooters", map[string]any{"last_name": "TWO", "first_name": "A", "country": "NOR", "club": "C"})
	status, _ := f.env.Do("POST", fmt.Sprintf("/api/matches/%d/entries", f.comp), map[string]any{"shooter_id": shooter, "category": "Revolver 1500", "lane": 2, "remarks": ok + "x"})
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("create 201: %d", status)
	}
}

func TestEntryRemarksCanBeEditedAfterFinishAndShowInResultsWhenNamesVisible(t *testing.T) {
	f := resultsFixture(t)
	status, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d/entries", f.comp), nil)
	if status != http.StatusOK {
		t.Fatalf("entries: %d %v", status, out)
	}
	first := out["entries"].([]any)[0].(map[string]any)
	id := int(first["id"].(float64))
	status, out = f.putEntry(id, map[string]any{"category": first["category"], "lane": first["lane"], "remarks": "Pistol change"})
	if status != http.StatusOK || out["remarks"] != "Pistol change" {
		t.Fatalf("finished edit: %d %v", status, out)
	}
	found := false
	for _, r := range resultRemarks(t, f) {
		found = found || r == "Pistol change"
	}
	if !found {
		t.Fatal("remarks missing from results")
	}
	off := map[string]any{"show_names": false, "show_clubs": true, "show_countries": true, "show_segment_scores": true, "show_ring_counts": true}
	if status, out := f.env.Do("PUT", fmt.Sprintf("/api/matches/%d/visibility", f.comp), off); status != http.StatusOK {
		t.Fatalf("visibility: %d %v", status, out)
	}
	for _, r := range resultRemarks(t, f) {
		if r != nil {
			t.Fatalf("remarks leaked: %v", r)
		}
	}
}
