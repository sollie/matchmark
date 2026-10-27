package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func publicGet(t *testing.T, f *fixture, path string) (int, map[string]any) {
	t.Helper()
	resp, err := http.Get(f.env.Server.URL + path)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func (f *fixture) score(t *testing.T, entry, stage int, counts map[string]any) {
	t.Helper()
	path := fmt.Sprintf("/api/competitions/%d/entries/%d/stages/%d", f.comp, entry, stage)
	if status, out := f.env.Do("PUT", path, counts); status != http.StatusOK {
		t.Fatalf("score: %d %v", status, out)
	}
}

func (f *fixture) scoreAccepted(t *testing.T, entry, stage int, counts map[string]any) {
	t.Helper()
	f.score(t, entry, stage, counts)
	path := fmt.Sprintf("/api/competitions/%d/entries/%d/stages/%d/lock", f.comp, entry, stage)
	if status, out := f.env.Do("POST", path, nil); status != http.StatusOK {
		t.Fatalf("lock: %d %v", status, out)
	}
}

func (f *fixture) activate(t *testing.T) {
	t.Helper()
	if status, out := f.env.Do("POST", fmt.Sprintf("/api/competitions/%d/status", f.comp), map[string]string{"status": "active"}); status != http.StatusOK {
		t.Fatalf("activate: %d %v", status, out)
	}
}

func spectatorFixture(t *testing.T) *fixture {
	f := newFixture(t, "150")
	one := f.addEntry(t, "HANSEN", 1)
	two := f.addEntry(t, "BERG", 2)
	f.scoreAccepted(t, one, 1, map[string]any{"x": 3, "10": 4, "9": 3, "8": 1, "7": 1})
	f.scoreAccepted(t, two, 1, map[string]any{"x": 12})
	f.activate(t)
	return f
}

func standingsPath(f *fixture) string {
	return fmt.Sprintf("/api/spectator/competitions/%d/standings", f.comp)
}

func TestSpectatorStandings(t *testing.T) {
	f := spectatorFixture(t)
	status, out := publicGet(t, f, standingsPath(f))
	if status != http.StatusOK {
		t.Fatalf("status %d %v", status, out)
	}
	rows := out["standings"].([]any)
	if len(rows) != 2 {
		t.Fatalf("rows %v", rows)
	}
	first := rows[0].(map[string]any)
	if first["last_name"] != "BERG" || first["total"].(float64) != 120 {
		t.Fatalf("first %v", first)
	}
	second := rows[1].(map[string]any)
	if second["last_name"] != "HANSEN" || second["total"].(float64) != 112 {
		t.Fatalf("second %v", second)
	}
}

func TestSpectatorDraftCompetitionIsHidden(t *testing.T) {
	f := newFixture(t, "150")
	if status, _ := publicGet(t, f, standingsPath(f)); status != http.StatusNotFound {
		t.Fatalf("status %d", status)
	}
}

func TestSpectatorVisibilityFlags(t *testing.T) {
	f := spectatorFixture(t)
	visPath := fmt.Sprintf("/api/competitions/%d/visibility", f.comp)

	t.Run("defaults are readable by admin", func(t *testing.T) {
		status, out := f.env.Do("GET", visPath, nil)
		if status != http.StatusOK || out["show_names"] != true || out["show_ring_counts"] != false {
			t.Fatalf("got %d %v", status, out)
		}
	})
	t.Run("ring counts hidden by default", func(t *testing.T) {
		_, out := publicGet(t, f, standingsPath(f))
		row := out["standings"].([]any)[0].(map[string]any)
		if _, ok := row["ring_counts"]; ok {
			t.Fatalf("ring_counts present: %v", row)
		}
	})
	t.Run("hidden fields are omitted", func(t *testing.T) {
		body := map[string]any{
			"show_names": false, "show_clubs": false, "show_countries": false,
			"show_stage_scores": false, "show_ring_counts": false,
		}
		if status, out := f.env.Do("PUT", visPath, body); status != http.StatusOK {
			t.Fatalf("put %d %v", status, out)
		}
		_, out := publicGet(t, f, standingsPath(f))
		row := out["standings"].([]any)[0].(map[string]any)
		for _, k := range []string{"last_name", "first_name", "club", "country", "ring_counts", "x_count"} {
			if _, ok := row[k]; ok {
				t.Fatalf("%s present: %v", k, row)
			}
		}
		stages := row["stages"].([]any)
		if len(stages) != 10 {
			t.Fatalf("stages %v", stages)
		}
		for _, st := range stages {
			m := st.(map[string]any)
			if _, ok := m["score"]; ok || m["state"] == nil {
				t.Fatalf("stage %v", m)
			}
		}
		if row["lane"] == nil || row["total"] == nil || row["category"] == nil {
			t.Fatalf("always-visible fields missing: %v", row)
		}
	})
	t.Run("enabled fields appear", func(t *testing.T) {
		body := map[string]any{
			"show_names": true, "show_clubs": true, "show_countries": true,
			"show_stage_scores": true, "show_ring_counts": true,
		}
		if status, out := f.env.Do("PUT", visPath, body); status != http.StatusOK {
			t.Fatalf("put %d %v", status, out)
		}
		_, out := publicGet(t, f, standingsPath(f))
		row := out["standings"].([]any)[0].(map[string]any)
		for _, k := range []string{"last_name", "first_name", "club", "country", "stages", "ring_counts"} {
			if _, ok := row[k]; !ok {
				t.Fatalf("%s missing: %v", k, row)
			}
		}
		stages := row["stages"].([]any)
		if len(stages) != 10 || stages[0].(map[string]any)["score"].(float64) != 120 {
			t.Fatalf("stages %v", stages)
		}
		if _, ok := stages[1].(map[string]any)["score"]; ok {
			t.Fatalf("unaccepted stage has score: %v", stages[1])
		}
	})
	t.Run("visibility requires admin", func(t *testing.T) {
		if status, _ := publicGet(t, f, visPath); status != http.StatusUnauthorized {
			t.Fatalf("status %d", status)
		}
	})
}

func TestSpectatorCompetitionList(t *testing.T) {
	f := spectatorFixture(t)
	f.env.ID("POST", "/api/competitions", map[string]any{"name": "Draft", "date": "2026-06-01", "course": "60"})
	status, out := publicGet(t, f, "/api/spectator/competitions")
	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	list := out["competitions"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["name"] != "Club Match" {
		t.Fatalf("list %v", list)
	}
}

func TestVersionIsPublic(t *testing.T) {
	f := newFixture(t, "150")
	status, out := publicGet(t, f, "/api/version")
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	if _, ok := out["version"].(string); !ok {
		t.Errorf("version = %v", out["version"])
	}
}

func TestRulebookIsPublic(t *testing.T) {
	f := newFixture(t, "150")
	status, out := publicGet(t, f, "/api/rulebook")
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	courses, _ := out["courses"].([]any)
	keys := []string{"150", "60", "48", "40"}
	shots := []float64{150, 60, 48, 40}
	if len(courses) != len(keys) {
		t.Fatalf("courses = %d, want %d", len(courses), len(keys))
	}
	for i, c := range courses {
		course := c.(map[string]any)
		if course["key"] != keys[i] || course["total_shots"] != shots[i] || course["label"] == "" {
			t.Errorf("course %d = %v", i, course)
		}
		sum := 0.0
		for _, st := range course["stages"].([]any) {
			sum += st.(map[string]any)["expected_shots"].(float64)
		}
		if sum != shots[i] {
			t.Errorf("course %s stage shots = %v, want %v", keys[i], sum, shots[i])
		}
	}
	wantCats := map[string]int{"150": 5, "60": 4, "48": 2, "40": 1}
	for _, c := range courses {
		course := c.(map[string]any)
		if got := len(course["categories"].([]any)); got != wantCats[course["key"].(string)] {
			t.Errorf("course %v categories = %v", course["key"], course["categories"])
		}
	}
	classes := out["weapon_classes"].(map[string]any)
	if len(classes["1500"].([]any)) != 4 || len(classes["60"].([]any)) != 3 || len(classes["48"].([]any)) != 6 {
		t.Errorf("weapon_classes = %v", classes)
	}
	if rings := out["rings"].([]any); len(rings) != 7 {
		t.Errorf("rings = %v", rings)
	}
}
