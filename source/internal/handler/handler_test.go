package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/sollie/matchmark/internal/apitest"
)

type fixture struct {
	env  *apitest.Env
	comp int
}

func newFixture(t *testing.T, course string) *fixture {
	t.Helper()
	env := apitest.NewAdmin(t)
	comp := env.ID("POST", "/api/competitions", map[string]any{"name": "Club Match", "date": "2026-05-01", "course": course})
	return &fixture{env: env, comp: comp}
}

func (f *fixture) addEntry(t *testing.T, name string, lane int) int {
	t.Helper()
	shooter := f.env.ID("POST", "/api/shooters", map[string]any{"last_name": name, "first_name": "A", "sex": "M", "country": "NOR", "club": "Club"})
	return f.env.ID("POST", fmt.Sprintf("/api/competitions/%d/entries", f.comp), map[string]any{"shooter_id": shooter, "category": "Revolver", "lane": lane})
}

func (f *fixture) scoreLane(lane int) (int, map[string]any) {
	return f.env.Do("GET", fmt.Sprintf("/api/competitions/%d/lanes/%d/score-lane", f.comp, lane), nil)
}

func TestScoreLane(t *testing.T) {
	f := newFixture(t, "150")
	f.addEntry(t, "One", 1)
	f.addEntry(t, "Three", 3)
	e5 := f.addEntry(t, "Five", 5)

	tests := []struct {
		name string
		own  int
		want int
	}{
		{"skips empty lane", 1, 3},
		{"next manned", 3, 5},
		{"wraps around", 5, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, out := f.scoreLane(tt.own)
			if status != http.StatusOK || int(out["score_lane"].(float64)) != tt.want {
				t.Fatalf("got %d %v, want lane %d", status, out, tt.want)
			}
		})
	}

	t.Run("own lane must be manned", func(t *testing.T) {
		if status, _ := f.scoreLane(2); status != http.StatusConflict {
			t.Fatalf("status %d", status)
		}
	})

	t.Run("withdrawn entry is skipped", func(t *testing.T) {
		status, _ := f.env.Do("POST", fmt.Sprintf("/api/competitions/%d/entries/%d/withdraw", f.comp, e5), nil)
		if status != http.StatusOK {
			t.Fatalf("withdraw status %d", status)
		}
		_, out := f.scoreLane(3)
		if int(out["score_lane"].(float64)) != 1 {
			t.Fatalf("got %v", out)
		}
	})
}

func TestScoreLaneSingleLaneRequiresAdmin(t *testing.T) {
	f := newFixture(t, "150")
	f.addEntry(t, "Solo", 4)
	status, out := f.scoreLane(4)
	if status != http.StatusForbidden || out["error"] != "admin scoring required" {
		t.Fatalf("got %d %v", status, out)
	}
}

func TestLaneIsExclusive(t *testing.T) {
	f := newFixture(t, "150")
	f.addEntry(t, "One", 1)
	shooter := f.env.ID("POST", "/api/shooters", map[string]any{"last_name": "Two", "first_name": "B", "sex": "F", "country": "NOR", "club": "Club"})
	status, _ := f.env.Do("POST", fmt.Sprintf("/api/competitions/%d/entries", f.comp), map[string]any{"shooter_id": shooter, "category": "Pistol", "lane": 1})
	if status != http.StatusConflict {
		t.Fatalf("status %d", status)
	}
}

func TestRingCounts(t *testing.T) {
	f := newFixture(t, "150")
	entry := f.addEntry(t, "One", 1)
	path := fmt.Sprintf("/api/competitions/%d/entries/%d/stages/1", f.comp, entry)

	tests := []struct {
		name string
		body map[string]any
		want int
	}{
		{"sums to expected shots", map[string]any{"x": 3, "10": 4, "9": 3, "8": 1, "7": 1, "0": 0, "miss": 0}, http.StatusOK},
		{"too few", map[string]any{"x": 3, "10": 4, "9": 3, "8": 1, "7": 0, "0": 0, "miss": 0}, http.StatusUnprocessableEntity},
		{"too many", map[string]any{"x": 13}, http.StatusUnprocessableEntity},
		{"negative count", map[string]any{"x": 14, "10": -2}, http.StatusUnprocessableEntity},
		{"all misses", map[string]any{"miss": 12}, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if status, out := f.env.Do("PUT", path, tt.body); status != tt.want {
				t.Fatalf("got %d %v", status, out)
			}
		})
	}

	t.Run("stage number outside course", func(t *testing.T) {
		status, _ := f.env.Do("PUT", fmt.Sprintf("/api/competitions/%d/entries/%d/stages/11", f.comp, entry), map[string]any{"x": 12})
		if status != http.StatusNotFound {
			t.Fatalf("status %d", status)
		}
	})
}

func TestStageExpectedShotsFollowCourse(t *testing.T) {
	f := newFixture(t, "40")
	entry := f.addEntry(t, "Five", 1)
	path := fmt.Sprintf("/api/competitions/%d/entries/%d/stages/1", f.comp, entry)
	if status, out := f.env.Do("PUT", path, map[string]any{"x": 5}); status != http.StatusOK {
		t.Fatalf("got %d %v", status, out)
	}
	if status, _ := f.env.Do("PUT", path, map[string]any{"x": 6}); status != http.StatusUnprocessableEntity {
		t.Fatalf("status %d", status)
	}
}

func TestCreateCompetitionRejectsNonISODate(t *testing.T) {
	env := apitest.NewAdmin(t)
	for _, date := range []string{"19/10/2026", "2026-13-01", "2026-2-1", ""} {
		status, _ := env.Do("POST", "/api/competitions", map[string]any{"name": "X", "date": date, "course": "150"})
		if status != http.StatusUnprocessableEntity {
			t.Fatalf("date %q: status %d", date, status)
		}
	}
}

func TestCreateCompetitionRejectsUnknownCourse(t *testing.T) {
	env := apitest.NewAdmin(t)
	status, _ := env.Do("POST", "/api/competitions", map[string]any{"name": "X", "date": "2026-05-01", "course": "99"})
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("status %d", status)
	}
}
