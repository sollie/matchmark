package handler_test

import (
	"fmt"
	"net/http"
	"testing"
)

func (f *fixture) setClass(t *testing.T, entry, classification, class int) {
	t.Helper()
	path := fmt.Sprintf("/api/matches/%d/entries/%d/classification", f.comp, entry)
	if status, out := f.env.Do("PUT", path, map[string]any{"classification_at_match": classification, "class": class}); status != http.StatusOK {
		t.Fatalf("class: %d %v", status, out)
	}
}

func (f *fixture) setClassified(t *testing.T, on bool) {
	t.Helper()
	path := fmt.Sprintf("/api/matches/%d", f.comp)
	status, out := f.env.Do("PUT", path, map[string]any{"name": "Club Match", "date": "2026-05-01", "course": "150", "classified": on})
	if status != http.StatusOK || out["classified"] != on {
		t.Fatalf("classified: %d %v", status, out)
	}
}

func classedFixture(t *testing.T) *fixture {
	f := newFixture(t, "150")
	alpha := f.addShooterEntry(t, "ALPHA", "1001", "Revolver 1500", 1)
	bravo := f.addShooterEntry(t, "BRAVO", "1002", "Revolver 1500", 2)
	charlie := f.addShooterEntry(t, "CHARLIE", "1003", "Revolver 1500", 3)
	delta := f.addShooterEntry(t, "DELTA", "1004", "Revolver 1500", 4)
	echo := f.addShooterEntry(t, "ECHO", "1005", "Pistol 1500", 5)
	f.activate(t)
	f.setClass(t, alpha, 4, 4)
	f.setClass(t, bravo, 1, 1)
	f.setClass(t, charlie, 4, 4)
	f.setClass(t, delta, 0, 4)
	f.setClass(t, echo, 5, 5)
	f.scoreAccepted(t, alpha, 1, map[string]any{"x": 6, "10": 14, "9": 4})
	f.scoreAccepted(t, bravo, 1, map[string]any{"x": 24})
	f.scoreAccepted(t, charlie, 1, map[string]any{"x": 2, "10": 16, "9": 6})
	f.scoreAccepted(t, delta, 1, map[string]any{"x": 1, "10": 17, "9": 6})
	f.scoreAccepted(t, echo, 1, map[string]any{"x": 4, "10": 20})
	f.finishActive(t)
	return f
}

func groupSummary(t *testing.T, f *fixture) (flat []string, groups map[string][]string) {
	t.Helper()
	status, out := publicGet(t, f, resultsPath(f))
	if status != http.StatusOK {
		t.Fatalf("results: %d %v", status, out)
	}
	cat := out["categories"].([]any)[0].(map[string]any)
	summarize := func(rows []any) []string {
		var s []string
		for _, r := range rows {
			m := r.(map[string]any)
			s = append(s, fmt.Sprintf("%v:%v", m["rank"], m["total"]))
		}
		return s
	}
	flat = summarize(cat["rows"].([]any))
	groups = map[string][]string{}
	for _, g := range cat["groups"].([]any) {
		m := g.(map[string]any)
		groups[m["class"].(map[string]any)["name"].(string)] = summarize(m["rows"].([]any))
	}
	return flat, groups
}

func TestResultsGroupByClassWithRankRestart(t *testing.T) {
	f := classedFixture(t)
	flat, groups := groupSummary(t, f)
	if len(flat) != 0 || len(groups) != 2 {
		t.Fatalf("flat %v groups %v", flat, groups)
	}
	if got := fmt.Sprint(groups["Master"]); got != "[1:236 2:234 3:234]" {
		t.Fatalf("master %s", got)
	}
	if got := fmt.Sprint(groups["Marksman"]); got != "[1:240]" {
		t.Fatalf("marksman %s", got)
	}
	_, out := publicGet(t, f, resultsPath(f))
	order := out["categories"].([]any)[0].(map[string]any)["groups"].([]any)
	if order[0].(map[string]any)["class"].(map[string]any)["name"] != "Master" {
		t.Fatalf("group order %v", order)
	}
	pistol := out["categories"].([]any)[1].(map[string]any)
	if len(pistol["groups"].([]any)) != 1 {
		t.Fatalf("pistol groups %v", pistol["groups"])
	}
}

func TestResultsFlatWhenNotClassified(t *testing.T) {
	f := classedFixture(t)
	f.setClassified(t, false)
	flat, groups := groupSummary(t, f)
	if fmt.Sprint(flat) != "[1:240 2:236 3:234 4:234]" || len(groups) != 0 {
		t.Fatalf("flat %v groups %v", flat, groups)
	}
}

func TestResultsFlatForNonClassifiedCategory(t *testing.T) {
	f := newFixture(t, "40")
	f.addShooterEntry(t, "ONE", "1001", "Standard Revolver 2,75\" 5 Shot", 1)
	f.activate(t)
	f.finishActive(t)
	flat, groups := groupSummary(t, f)
	if len(flat) != 1 || len(groups) != 0 {
		t.Fatalf("flat %v groups %v", flat, groups)
	}
}

func TestResultsGroupClasslessEntryWithCompeteAsClass(t *testing.T) {
	f := newFixture(t, "150")
	one := f.addShooterEntry(t, "ONE", "1001", "Revolver 1500", 1)
	f.addShooterEntry(t, "TWO", "", "Revolver 1500", 2)
	f.activate(t)
	f.setClass(t, one, 4, 4)
	f.scoreAccepted(t, one, 1, map[string]any{"x": 24})
	f.finishActive(t)
	flat, groups := groupSummary(t, f)
	if len(flat) != 0 || len(groups) != 1 || len(groups["Master"]) != 2 {
		t.Fatalf("flat %v groups %v", flat, groups)
	}
}

func TestStandingsCarryClassGroup(t *testing.T) {
	f := classedFixture(t)
	standings := func() []any {
		_, out := publicGet(t, f, fmt.Sprintf("/api/spectator/matches/%d/standings", f.comp))
		return out["standings"].([]any)
	}
	for _, s := range standings() {
		if s.(map[string]any)["class_group"] == nil {
			t.Fatalf("missing class_group: %v", s)
		}
	}
	f.setClassified(t, false)
	for _, s := range standings() {
		if s.(map[string]any)["class_group"] != nil {
			t.Fatalf("class_group when off: %v", s)
		}
	}
}
