package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/sollie/matchmark/internal/apitest"
)

func lastScoreLine(t *testing.T, env *apitest.Env) map[string]any {
	t.Helper()
	lines := env.ScoreLogLines()
	if len(lines) == 0 {
		t.Fatal("no score log lines")
	}
	return lines[len(lines)-1]
}

func TestScoreLogNamesUserForUserLocksAndLaneForDeviceLocks(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	stage := fmt.Sprintf("/api/matches/%d/entries/%d/stages/1", f.comp, f.entries[2])

	token := f.pair(t, 1)
	if status, out := f.env.DoToken(token, "PUT", "/api/scorer/stages/1", counts(10, 2)); status != http.StatusOK {
		t.Fatalf("set counts: %d %v", status, out)
	}
	if status, out := f.env.DoToken(token, "POST", "/api/scorer/stages/1/lock", nil); status != http.StatusOK {
		t.Fatalf("device lock: %d %v", status, out)
	}
	if line := lastScoreLine(t, f.env); line["locked_by"] != float64(1) {
		t.Fatalf("device lock line: %v", line)
	}

	if status, out := f.env.Do("PUT", stage, counts(11, 1)); status != http.StatusOK {
		t.Fatalf("user edit: %d %v", status, out)
	}
	line := lastScoreLine(t, f.env)
	if line["locked_by"] != apitest.AdminUsername || line["scorer_name"] != apitest.AdminUsername {
		t.Fatalf("user edit line: %v", line)
	}

	f.env.Do("POST", stage+"/unlock", nil)
	if status, out := f.env.Do("POST", stage+"/lock", nil); status != http.StatusOK && status != http.StatusNoContent {
		t.Fatalf("user lock: %d %v", status, out)
	}
	line = lastScoreLine(t, f.env)
	if line["event"] != "stage_status" || line["locked_by"] != apitest.AdminUsername || line["scorer_name"] != apitest.AdminUsername {
		t.Fatalf("user lock line: %v", line)
	}

	_, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d/entries", f.comp), nil)
	for _, e := range out["entries"].([]any) {
		m := e.(map[string]any)
		if int(m["lane"].(float64)) == 2 {
			st := m["stages"].([]any)[0].(map[string]any)
			if st["scorer_name"] != apitest.AdminUsername {
				t.Fatalf("entries stage: %v", st)
			}
		}
	}
}

func TestResultsCSVNamesUserWhoLockedStage(t *testing.T) {
	f := newFixture(t, "150")
	entry := f.addEntry(t, "One", 1)
	stage := fmt.Sprintf("/api/matches/%d/entries/%d/stages/1", f.comp, entry)
	f.env.Do("POST", fmt.Sprintf("/api/matches/%d/status", f.comp), map[string]any{"status": "active"})
	f.env.Do("PUT", stage, map[string]any{"x": 12})
	f.env.Do("POST", stage+"/lock", nil)
	_, _, body := f.env.Raw("GET", fmt.Sprintf("/api/matches/%d/results.csv", f.comp))
	want := ",true,," + apitest.AdminUsername + "\n"
	if !strings.Contains(string(body), want) {
		t.Fatalf("csv lacks %q: %s", want, body)
	}
}
