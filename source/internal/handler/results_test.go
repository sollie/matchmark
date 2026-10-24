package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/sollie/matchmark/internal/handler"
)

func (f *fixture) addShooterEntry(t *testing.T, last, waid, category string, lane int) int {
	t.Helper()
	body := map[string]any{"last_name": last, "first_name": "A", "sex": "M", "country": "NOR", "club": "Club " + last}
	if waid != "" {
		body["waid"] = waid
	}
	shooter := f.env.ID("POST", "/api/shooters", body)
	return f.env.ID("POST", fmt.Sprintf("/api/competitions/%d/entries", f.comp), map[string]any{"shooter_id": shooter, "category": category, "lane": lane})
}

func resultsPath(f *fixture) string {
	return fmt.Sprintf("/api/spectator/competitions/%d/results", f.comp)
}

func resultsFixture(t *testing.T, opts ...handler.Option) *fixture {
	f := newFixture(t, "150", opts...)
	alpha := f.addShooterEntry(t, "Alpha", "1001", "Revolver", 1)
	bravo := f.addShooterEntry(t, "Bravo", "1002", "Revolver", 2)
	charlie := f.addShooterEntry(t, "Charlie", "", "Revolver", 3)
	delta := f.addShooterEntry(t, "Delta", "1004", "Revolver", 4)
	echo := f.addShooterEntry(t, "Echo", "1005", "Pistol", 5)
	f.activate(t)
	f.scoreAccepted(t, alpha, 1, map[string]any{"x": 3, "10": 7, "9": 2})
	f.scoreAccepted(t, bravo, 1, map[string]any{"x": 3, "10": 7, "9": 2})
	f.scoreAccepted(t, charlie, 1, map[string]any{"x": 1, "10": 8, "9": 3})
	f.scoreAccepted(t, delta, 1, map[string]any{"x": 12})
	f.scoreAccepted(t, echo, 1, map[string]any{"x": 2, "10": 10})
	if status, out := f.env.Do("POST", fmt.Sprintf("/api/competitions/%d/entries/%d/withdraw", f.comp, delta), nil); status != http.StatusOK {
		t.Fatalf("withdraw: %d %v", status, out)
	}
	f.finishActive(t)
	on := map[string]any{"show_names": true, "show_clubs": true, "show_countries": true, "show_stage_scores": true, "show_ring_counts": true}
	if status, out := f.env.Do("PUT", fmt.Sprintf("/api/competitions/%d/visibility", f.comp), on); status != http.StatusOK {
		t.Fatalf("visibility: %d %v", status, out)
	}
	return f
}

func TestResultsOnlyForFinishedCompetitions(t *testing.T) {
	f := newFixture(t, "150")
	f.addEntry(t, "One", 1)
	if status, _ := publicGet(t, f, resultsPath(f)); status != http.StatusNotFound {
		t.Fatalf("draft: %d", status)
	}
	f.activate(t)
	if status, _ := publicGet(t, f, resultsPath(f)); status != http.StatusNotFound {
		t.Fatalf("active: %d", status)
	}
	f.finishActive(t)
	if status, _ := publicGet(t, f, resultsPath(f)); status != http.StatusOK {
		t.Fatalf("finished: %d", status)
	}
}

func (f *fixture) finishActive(t *testing.T) {
	t.Helper()
	if status, out := f.env.Do("POST", fmt.Sprintf("/api/competitions/%d/status", f.comp), map[string]string{"status": "finished"}); status != http.StatusOK {
		t.Fatalf("finish: %d %v", status, out)
	}
}

func TestResultsRankingAndGrouping(t *testing.T) {
	f := resultsFixture(t)
	status, out := publicGet(t, f, resultsPath(f))
	if status != http.StatusOK {
		t.Fatalf("status %d %v", status, out)
	}
	comp := out["competition"].(map[string]any)
	if comp["name"] != "Club Match" || comp["date"] != "2026-05-01" || comp["course_label"] == "" {
		t.Fatalf("competition %v", comp)
	}
	cats := out["categories"].([]any)
	if len(cats) != 2 || cats[0].(map[string]any)["category"] != "Revolver" || cats[1].(map[string]any)["category"] != "Pistol" {
		t.Fatalf("categories %v", cats)
	}
	rows := cats[0].(map[string]any)["rows"].([]any)
	var names []string
	var ranks []float64
	for _, r := range rows {
		m := r.(map[string]any)
		names = append(names, m["last_name"].(string))
		ranks = append(ranks, m["rank"].(float64))
	}
	if fmt.Sprint(names) != "[Alpha Bravo Charlie]" || fmt.Sprint(ranks) != "[1 1 3]" {
		t.Fatalf("names %v ranks %v", names, ranks)
	}
	first := rows[0].(map[string]any)
	if first["waid"] != "1001" || first["total"].(float64) != 118 || first["x_count"].(float64) != 3 {
		t.Fatalf("first %v", first)
	}
	stages := first["stages"].([]any)
	if stages[0].(map[string]any)["score"].(float64) != 118 {
		t.Fatalf("stages %v", stages)
	}
	fresh := out["new_shooters"].([]any)
	if len(fresh) != 1 || fresh[0].(map[string]any)["last_name"] != "Charlie" || fresh[0].(map[string]any)["category"] != "Revolver" {
		t.Fatalf("new shooters %v", fresh)
	}
}

func TestResultsRankBreaksTieOnXCount(t *testing.T) {
	f := newFixture(t, "150")
	a := f.addShooterEntry(t, "Alpha", "1", "Revolver", 1)
	b := f.addShooterEntry(t, "Bravo", "2", "Revolver", 2)
	f.activate(t)
	f.scoreAccepted(t, a, 1, map[string]any{"x": 2, "10": 8, "9": 2})
	f.scoreAccepted(t, b, 1, map[string]any{"x": 5, "10": 5, "9": 2})
	f.finishActive(t)
	_, out := publicGet(t, f, resultsPath(f))
	rows := out["categories"].([]any)[0].(map[string]any)["rows"].([]any)
	if rows[0].(map[string]any)["last_name"] != "Bravo" || rows[1].(map[string]any)["rank"].(float64) != 2 {
		t.Fatalf("rows %v", rows)
	}
}

func TestResultsHonoursVisibilityFlags(t *testing.T) {
	f := resultsFixture(t)
	visPath := fmt.Sprintf("/api/competitions/%d/visibility", f.comp)
	off := map[string]any{"show_names": false, "show_clubs": false, "show_countries": false, "show_stage_scores": false, "show_ring_counts": false}
	if status, out := f.env.Do("PUT", visPath, off); status != http.StatusOK {
		t.Fatalf("put %d %v", status, out)
	}
	_, out := publicGet(t, f, resultsPath(f))
	row := out["categories"].([]any)[0].(map[string]any)["rows"].([]any)[0].(map[string]any)
	for _, k := range []string{"last_name", "first_name", "waid", "club", "country", "x_count"} {
		if _, ok := row[k]; ok {
			t.Fatalf("%s present: %v", k, row)
		}
	}
	for _, st := range row["stages"].([]any) {
		if _, ok := st.(map[string]any)["score"]; ok {
			t.Fatalf("stage score present: %v", st)
		}
	}
	if row["total"] == nil || row["rank"] == nil {
		t.Fatalf("always-visible fields missing: %v", row)
	}
	if fresh := out["new_shooters"].([]any); len(fresh) != 0 {
		t.Fatalf("new shooters listed without names: %v", fresh)
	}
}
