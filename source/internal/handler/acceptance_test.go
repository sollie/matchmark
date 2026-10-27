package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/sollie/matchmark/internal/apitest"
)

func (f *scoringFixture) stageStatus(t *testing.T, lane, stage int) string {
	t.Helper()
	_, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d/entries", f.comp), nil)
	for _, e := range out["entries"].([]any) {
		m := e.(map[string]any)
		if int(m["lane"].(float64)) == lane {
			return m["stages"].([]any)[stage-1].(map[string]any)["status"].(string)
		}
	}
	t.Fatalf("lane %d not found", lane)
	return ""
}

func (f *scoringFixture) adminStage(lane, stage int, action string) string {
	return fmt.Sprintf("/api/matches/%d/entries/%d/stages/%d%s", f.comp, f.entries[lane], stage, action)
}

func (f *scoringFixture) deviceLock(t *testing.T, scorerLane, stage int) {
	t.Helper()
	token := f.pair(t, scorerLane)
	if status, out := f.env.DoToken(token, "PUT", fmt.Sprintf("/api/scorer/stages/%d", stage), counts(10, 2)); status != http.StatusOK {
		t.Fatalf("set counts: %d %v", status, out)
	}
	status, out := f.env.DoToken(token, "POST", fmt.Sprintf("/api/scorer/stages/%d/lock", stage), nil)
	if status != http.StatusOK || out["status"] != "awaiting_acceptance" {
		t.Fatalf("device lock: %d %v", status, out)
	}
}

func (f *scoringFixture) standingTotals(t *testing.T) []float64 {
	t.Helper()
	status, out := publicGet(t, f.fixture, standingsPath(f.fixture))
	if status != http.StatusOK {
		t.Fatalf("standings: %d %v", status, out)
	}
	var totals []float64
	for _, r := range out["standings"].([]any) {
		totals = append(totals, r.(map[string]any)["total"].(float64))
	}
	return totals
}

func TestDeviceLockAwaitsAcceptanceAndCountsNowhereUntilAccepted(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	f.deviceLock(t, 1, 1)

	if got := f.stageStatus(t, 2, 1); got != "awaiting_acceptance" {
		t.Fatalf("status %q", got)
	}
	if status, out := f.env.DoToken(f.pair(t, 1), "PUT", "/api/scorer/stages/1", counts(9, 3)); status != http.StatusConflict {
		t.Fatalf("hits must be frozen: %d %v", status, out)
	}
}

func TestAwaitingStageIsNotCountedUntilAdminAccepts(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	f.deviceLock(t, 1, 1)

	for _, total := range f.standingTotals(t) {
		if total != 0 {
			t.Fatalf("awaiting stage counted: %v", f.standingTotals(t))
		}
	}

	if status, out := f.env.Do("POST", f.adminStage(2, 1, "/accept"), nil); status != http.StatusOK || out["status"] != "accepted" {
		t.Fatalf("accept: %d %v", status, out)
	}
	if totals := f.standingTotals(t); totals[0] != 120 {
		t.Fatalf("accepted stage not counted: %v", totals)
	}
}

func TestAdminAcceptRequiresAwaitingAcceptance(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	f.env.Do("PUT", f.adminStage(2, 1, ""), counts(12, 0))
	if status, _ := f.env.Do("POST", f.adminStage(2, 1, "/accept"), nil); status != http.StatusConflict {
		t.Fatalf("accept open stage: %d", status)
	}
	f.env.Do("POST", f.adminStage(2, 1, "/lock"), nil)
	if status, _ := f.env.Do("POST", f.adminStage(2, 1, "/accept"), nil); status != http.StatusConflict {
		t.Fatalf("accept accepted stage: %d", status)
	}
}

func TestAdminLockAcceptsDirectlyAndUnlockReopensKeepingHits(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	f.env.Do("PUT", f.adminStage(2, 1, ""), counts(12, 0))
	if status, out := f.env.Do("POST", f.adminStage(2, 1, "/lock"), nil); status != http.StatusOK || out["status"] != "accepted" {
		t.Fatalf("admin lock: %d %v", status, out)
	}
	if status, out := f.env.Do("POST", f.adminStage(2, 1, "/unlock"), nil); status != http.StatusOK || out["status"] != "open" {
		t.Fatalf("unlock: %d %v", status, out)
	}
	if got := f.stageStatus(t, 2, 1); got != "open" {
		t.Fatalf("status %q", got)
	}
	_, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d/entries", f.comp), nil)
	for _, e := range out["entries"].([]any) {
		m := e.(map[string]any)
		if int(m["lane"].(float64)) == 2 {
			if st := m["stages"].([]any)[0].(map[string]any); st["shots_entered"] != float64(12) {
				t.Fatalf("hits lost: %v", st)
			}
		}
	}
}

func TestAdminUnlockFromAwaitingAcceptanceAndOverwriteEndsAccepted(t *testing.T) {
	f := newScoringFixture(t, 1, 2, 3)
	f.deviceLock(t, 1, 1)
	if status, _ := f.env.Do("POST", f.adminStage(2, 1, "/unlock"), nil); status != http.StatusOK {
		t.Fatalf("unlock awaiting: %d", status)
	}
	if status, _ := f.env.Do("POST", f.adminStage(2, 1, "/unlock"), nil); status != http.StatusConflict {
		t.Fatalf("unlock open: %d", status)
	}

	f.deviceLock(t, 2, 1)
	if status, _ := f.env.Do("PUT", f.adminStage(3, 1, ""), counts(11, 1)); status != http.StatusOK {
		t.Fatalf("overwrite: %d", status)
	}
	if got := f.stageStatus(t, 3, 1); got != "accepted" {
		t.Fatalf("overwrite of awaiting stage: %q", got)
	}
}

func TestStatusChangesWriteStageStatusLinesAndAuditActions(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	f.deviceLock(t, 1, 1)
	line := lastScoreLine(t, f.env)
	if line["event"] != "stage_status" || line["msg"] != "stage_status" || line["schema_version"] != float64(3) ||
		line["status"] != "awaiting_acceptance" || line["locked_by"] != float64(1) {
		t.Fatalf("lock line: %v", line)
	}
	if _, ok := line["accepted_by"]; ok && line["accepted_by"] != "" {
		t.Fatalf("accepted_by on awaiting line: %v", line)
	}

	f.env.Do("POST", f.adminStage(2, 1, "/accept"), nil)
	line = lastScoreLine(t, f.env)
	if line["status"] != "accepted" || line["accepted_by"] != apitest.AdminUsername || line["accepted_at"] == "" {
		t.Fatalf("accept line: %v", line)
	}

	f.env.Do("POST", f.adminStage(2, 1, "/unlock"), nil)
	line = lastScoreLine(t, f.env)
	if line["status"] != "open" || line["ring_counts"].([]any)[0] != float64(10) {
		t.Fatalf("unlock line: %v", line)
	}

	actions := map[string]bool{}
	for _, a := range f.env.AuditLines() {
		actions[fmt.Sprint(a["action"])] = true
	}
	for _, want := range []string{"stage.lock", "stage.accept", "stage.unlock"} {
		if !actions[want] {
			t.Fatalf("audit lacks %s: %v", want, actions)
		}
	}
}

func TestStageEventsRecordEveryStatusChange(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	f.deviceLock(t, 1, 1)
	f.env.Do("POST", f.adminStage(2, 1, "/accept"), nil)
	f.env.Do("POST", f.adminStage(2, 1, "/unlock"), nil)

	events := f.env.StageEvents(f.comp)
	var got []string
	for _, e := range events {
		got = append(got, string(e.Event)+":"+e.ActorName)
	}
	want := []string{"locked:device", "accepted:" + apitest.AdminUsername, "unlocked:" + apitest.AdminUsername}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("events %v, want %v", got, want)
	}
	if events[0].ScorerName == "" || events[0].ShooterName == "" {
		t.Fatalf("device event lacks names: %+v", events[0])
	}
}

func TestReportAndCSVCountOnlyAcceptedStages(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	f.deviceLock(t, 1, 1)
	_, _, body := f.env.Raw("GET", fmt.Sprintf("/api/matches/%d/results.csv", f.comp))
	if strings.Contains(string(body), ",120,") {
		t.Fatalf("awaiting stage in csv: %s", body)
	}
	f.env.Do("POST", f.adminStage(2, 1, "/accept"), nil)
	_, _, body = f.env.Raw("GET", fmt.Sprintf("/api/matches/%d/results.csv", f.comp))
	if !strings.Contains(string(body), ",120,") {
		t.Fatalf("accepted stage missing from csv: %s", body)
	}
}
