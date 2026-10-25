package handler_test

import (
	"fmt"
	"net/http"
	"testing"
)

func (f *scoringFixture) session(t *testing.T, token string) map[string]any {
	t.Helper()
	status, out := f.env.DoToken(token, "GET", "/api/scorer/session", nil)
	if status != http.StatusOK {
		t.Fatalf("session: %d %v", status, out)
	}
	return out
}

func (f *scoringFixture) deviceAction(token string, stage int, action string) (int, map[string]any) {
	return f.env.DoToken(token, "POST", fmt.Sprintf("/api/scorer/stages/%d/%s", stage, action), nil)
}

func TestShooterAcceptsOwnLockedStage(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	f.deviceLock(t, 1, 1)

	status, out := f.deviceAction(f.pair(t, 2), 1, "accept")
	if status != http.StatusOK || out["status"] != "accepted" {
		t.Fatalf("accept: %d %v", status, out)
	}
	if got := f.stageStatus(t, 2, 1); got != "accepted" {
		t.Fatalf("status %q", got)
	}
	if totals := f.standingTotals(t); totals[0] != 120 {
		t.Fatalf("accepted stage not counted: %v", totals)
	}
	if status, _ := f.deviceAction(f.pair(t, 2), 1, "accept"); status != http.StatusConflict {
		t.Fatalf("second accept: %d", status)
	}
}

func TestDisputeReopensStageKeepingHitsAndNoticePersistsUntilRelock(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	f.deviceLock(t, 1, 1)
	shooter, scorer := f.pair(t, 2), f.pair(t, 1)

	if status, out := f.deviceAction(shooter, 1, "dispute"); status != http.StatusOK || out["status"] != "open" {
		t.Fatalf("dispute: %d %v", status, out)
	}
	sess := f.session(t, scorer)
	st := sess["stages"].([]any)[0].(map[string]any)
	if st["status"] != "open" || st["disputed"] != true || st["10"] != float64(2) || st["x"] != float64(10) {
		t.Fatalf("disputed stage: %v", st)
	}
	if sess["phase"] != "scoring" {
		t.Fatalf("phase %v", sess["phase"])
	}
	if status, _ := f.env.DoToken(scorer, "PUT", "/api/scorer/stages/1", counts(9, 3)); status != http.StatusOK {
		t.Fatalf("rescore: %d", status)
	}
	if st := f.session(t, scorer)["stages"].([]any)[0].(map[string]any); st["disputed"] != true {
		t.Fatalf("notice lost on edit: %v", st)
	}
	if status, _ := f.env.DoToken(scorer, "POST", "/api/scorer/stages/1/lock", nil); status != http.StatusOK {
		t.Fatalf("relock: %d", status)
	}
	if st := f.session(t, scorer)["stages"].([]any)[0].(map[string]any); st["disputed"] != false {
		t.Fatalf("notice not cleared by lock: %v", st)
	}
}

func TestDeviceCannotActOnAnotherLanesStage(t *testing.T) {
	f := newScoringFixture(t, 1, 2, 3)
	f.deviceLock(t, 1, 1)

	for _, action := range []string{"accept", "dispute"} {
		if status, _ := f.deviceAction(f.pair(t, 1), 1, action); status != http.StatusConflict {
			t.Fatalf("scorer %s: %d", action, status)
		}
		if status, _ := f.deviceAction(f.pair(t, 3), 1, action); status != http.StatusConflict {
			t.Fatalf("other lane %s: %d", action, status)
		}
	}
	if got := f.stageStatus(t, 2, 1); got != "awaiting_acceptance" {
		t.Fatalf("status %q", got)
	}
}

func (f *scoringFixture) lockOnDevice(t *testing.T, token string) {
	t.Helper()
	f.env.DoToken(token, "PUT", "/api/scorer/stages/1", counts(10, 2))
	if status, out := f.deviceAction(token, 1, "lock"); status != http.StatusOK {
		t.Fatalf("lock: %d %v", status, out)
	}
}

func TestPhaseFollowsLockAndAcceptanceAndSurvivesReload(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	lane1, lane2 := f.pair(t, 1), f.pair(t, 2)

	if got := f.session(t, lane1)["phase"]; got != "scoring" {
		t.Fatalf("initial phase %v", got)
	}
	f.lockOnDevice(t, lane1)
	if got := f.session(t, lane1)["phase"]; got != "waiting" {
		t.Fatalf("lane 1 phase after own lock: %v", got)
	}
	if got := f.session(t, lane2)["phase"]; got != "scoring" {
		t.Fatalf("acceptance must not interrupt scoring: %v", got)
	}

	f.lockOnDevice(t, lane2)
	if got := f.session(t, lane1)["phase"]; got != "needs_acceptance" {
		t.Fatalf("lane 1 phase once lane 2 locked: %v", got)
	}
	sess := f.session(t, lane2)
	acc := sess["acceptance"].(map[string]any)
	if acc["stage_number"] != float64(1) || acc["x"] != float64(10) || acc["10"] != float64(2) || acc["total"] != float64(12) ||
		acc["scorer_name"] != "A SHOOTER1" {
		t.Fatalf("acceptance: %v", acc)
	}

	f.deviceAction(lane2, 1, "accept")
	if got := f.session(t, lane2)["phase"]; got != "waiting" {
		t.Fatalf("phase after accepting while own lock awaits: %v", got)
	}
	f.deviceAction(lane1, 1, "accept")
	if got := f.session(t, lane1)["phase"]; got != "waiting" {
		t.Fatalf("lane 1 waits at the gate once both stages are accepted: %v", got)
	}
}

func TestShooterWhoScoresSeesRequestOnlyAfterLockingOwnScoring(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	f.deviceLock(t, 2, 1)
	lane1 := f.pair(t, 1)
	if got := f.session(t, lane1)["phase"]; got != "scoring" {
		t.Fatalf("phase before own lock: %v", got)
	}
	if _, ok := f.session(t, lane1)["acceptance"]; ok {
		t.Fatal("acceptance exposed while scoring")
	}
}

func TestAcceptAndDisputeWriteEventsLogLinesAndAuditActions(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	f.deviceLock(t, 1, 1)
	shooter := f.pair(t, 2)

	f.deviceAction(shooter, 1, "dispute")
	line := lastScoreLine(t, f.env)
	if line["status"] != "open" || line["disputed_at"] == "" || line["ring_counts"].([]any)[0] != float64(10) {
		t.Fatalf("dispute line: %v", line)
	}
	f.env.DoToken(f.pair(t, 1), "POST", "/api/scorer/stages/1/lock", nil)
	f.deviceAction(shooter, 1, "accept")
	line = lastScoreLine(t, f.env)
	if line["status"] != "accepted" || line["accepted_by"] != "2" || line["accepted_at"] == "" || line["disputed_at"] != "" {
		t.Fatalf("accept line: %v", line)
	}

	var got []string
	for _, e := range f.env.StageEvents(f.comp) {
		got = append(got, string(e.Event)+":"+e.ActorName)
	}
	want := []string{"locked:device", "disputed:device", "locked:device", "accepted:device"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("events %v, want %v", got, want)
	}
	actions := map[string]bool{}
	for _, a := range f.env.AuditLines() {
		actions[fmt.Sprint(a["action"])] = true
	}
	for _, w := range []string{"stage.accept", "stage.dispute"} {
		if !actions[w] {
			t.Fatalf("audit lacks %s: %v", w, actions)
		}
	}
}
