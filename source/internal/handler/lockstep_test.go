package handler_test

import (
	"fmt"
	"net/http"
	"testing"
)

func (f *scoringFixture) startNext(comp int) (int, map[string]any) {
	return f.env.Do("POST", fmt.Sprintf("/api/matches/%d/next-segment", comp), nil)
}

func (f *scoringFixture) fullSegment(t *testing.T, lane, segment int) map[string]any {
	t.Helper()
	_, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d/entries", f.comp), nil)
	for _, e := range out["entries"].([]any) {
		m := e.(map[string]any)
		if int(m["lane"].(float64)) == lane {
			return map[string]any{"x": m["segments"].([]any)[segment-1].(map[string]any)["expected_shots"]}
		}
	}
	t.Fatalf("lane %d not found", lane)
	return nil
}

func (f *scoringFixture) acceptAll(t *testing.T, segment int, lanes ...int) {
	t.Helper()
	for _, l := range lanes {
		f.env.Do("PUT", f.adminSegment(l, segment, ""), f.fullSegment(t, l, segment))
		if status, out := f.env.Do("POST", f.adminSegment(l, segment, "/lock"), nil); status != http.StatusOK {
			t.Fatalf("accept lane %d segment %d: %d %v", l, segment, status, out)
		}
	}
}

func (f *scoringFixture) scoreAndLock(t *testing.T, token string, lane, segment int) {
	t.Helper()
	if status, out := f.env.DoToken(token, "PUT", fmt.Sprintf("/api/scorer/segments/%d", segment), f.fullSegment(t, lane, segment)); status != http.StatusOK {
		t.Fatalf("set counts: %d %v", status, out)
	}
	if status, out := f.deviceAction(token, segment, "lock"); status != http.StatusOK {
		t.Fatalf("lock: %d %v", status, out)
	}
}

func (f *scoringFixture) currentSegment(t *testing.T) int {
	t.Helper()
	_, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d", f.comp), nil)
	return int(out["current_segment"].(float64))
}

func TestActivationStartsAtSegmentOneAndDevicesScoreOnlyTheCurrentSegment(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	if got := f.currentSegment(t); got != 1 {
		t.Fatalf("current segment %d", got)
	}
	token := f.pair(t, 1)
	if got := f.session(t, token)["current_segment"]; got != float64(1) {
		t.Fatalf("session current segment %v", got)
	}
	if status, _ := f.env.DoToken(token, "PUT", "/api/scorer/segments/2", counts(10, 2)); status != http.StatusConflict {
		t.Fatalf("set counts on a later segment: %d", status)
	}
	if status, _ := f.deviceAction(token, 2, "lock"); status != http.StatusConflict {
		t.Fatalf("lock a later segment: %d", status)
	}
	if status, _ := f.env.DoToken(token, "PUT", "/api/scorer/segments/1", counts(10, 2)); status != http.StatusOK {
		t.Fatalf("set counts on current segment: %d", status)
	}
}

func TestStartNextSegmentIsRejectedUntilEveryMannedLaneHasAccepted(t *testing.T) {
	f := newScoringFixture(t, 1, 2, 3)
	if status, _ := f.startNext(f.comp); status != http.StatusConflict {
		t.Fatalf("nothing accepted: %d", status)
	}
	f.acceptAll(t, 1, 1, 2)
	if status, _ := f.startNext(f.comp); status != http.StatusConflict {
		t.Fatalf("one lane missing: %d", status)
	}
	f.deviceLock(t, 2, 1)
	if status, _ := f.startNext(f.comp); status != http.StatusConflict {
		t.Fatalf("awaiting acceptance is not accepted: %d", status)
	}
	f.deviceAction(f.pair(t, 3), 1, "accept")
	status, out := f.startNext(f.comp)
	if status != http.StatusOK || out["current_segment"] != float64(2) {
		t.Fatalf("start next: %d %v", status, out)
	}
	if got := f.currentSegment(t); got != 2 {
		t.Fatalf("current segment %d", got)
	}
}

func TestWithdrawnLaneDoesNotBlockTheGate(t *testing.T) {
	f := newScoringFixture(t, 1, 2, 3)
	f.acceptAll(t, 1, 1, 2, 3)
	if status, _ := f.env.Do("POST", fmt.Sprintf("/api/matches/%d/entries/%d/withdraw", f.comp, f.entries[3]), nil); status != http.StatusOK && status != http.StatusNoContent {
		t.Fatalf("withdraw at the gate: %d", status)
	}
	f.env.Do("POST", f.adminSegment(3, 1, "/unlock"), nil)
	if status, out := f.startNext(f.comp); status != http.StatusOK {
		t.Fatalf("start next: %d %v", status, out)
	}
}

func TestPhasesAcrossTheGate(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	lane1, lane2 := f.pair(t, 1), f.pair(t, 2)
	f.lockOnDevice(t, lane1)
	f.lockOnDevice(t, lane2)
	f.deviceAction(lane1, 1, "accept")
	f.deviceAction(lane2, 1, "accept")
	if got := f.session(t, lane1)["phase"]; got != "waiting" {
		t.Fatalf("phase at the gate: %v", got)
	}
	if status, out := f.startNext(f.comp); status != http.StatusOK {
		t.Fatalf("start next: %d %v", status, out)
	}
	if got := f.session(t, lane1)["phase"]; got != "next_segment" {
		t.Fatalf("phase after next segment started: %v", got)
	}
	if status, _ := f.env.DoToken(lane1, "PUT", "/api/scorer/segments/2", f.fullSegment(t, 2, 2)); status != http.StatusOK {
		t.Fatalf("score segment 2: %d", status)
	}
	if got := f.session(t, lane1)["phase"]; got != "scoring" {
		t.Fatalf("phase once scoring: %v", got)
	}
}

func TestNextSegmentBannerClearsOnceTheScorerSaves(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	lane1, lane2 := f.pair(t, 1), f.pair(t, 2)
	f.acceptAll(t, 1, 1, 2)
	if status, out := f.startNext(f.comp); status != http.StatusOK {
		t.Fatalf("start next: %d %v", status, out)
	}
	if status, _ := f.env.DoToken(lane1, "PUT", "/api/scorer/segments/2", f.fullSegment(t, 2, 2)); status != http.StatusOK {
		t.Fatalf("save hits: %d", status)
	}
	if got := f.session(t, lane1)["phase"]; got != "scoring" {
		t.Fatalf("phase after the scorer saved: %v", got)
	}
	if got := f.session(t, f.pair(t, 1))["phase"]; got != "scoring" {
		t.Fatalf("phase after re-pairing: %v", got)
	}
	if got := f.session(t, lane2)["phase"]; got != "next_segment" {
		t.Fatalf("other lane phase: %v", got)
	}
}

func TestLastAcceptanceFinishesTheMatchAndRevokesPairings(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	lane1, lane2 := f.pair(t, 1), f.pair(t, 2)
	for segment := 1; segment <= 6; segment++ {
		if segment > 1 {
			if status, out := f.startNext(f.comp); status != http.StatusOK {
				t.Fatalf("start segment %d: %d %v", segment, status, out)
			}
		}
		for _, l := range []int{1, 2} {
			f.scoreAndLock(t, map[int]string{1: lane1, 2: lane2}[l], 3-l, segment)
		}
		f.deviceAction(lane1, segment, "accept")
		if segment < 6 {
			f.deviceAction(lane2, segment, "accept")
		}
	}
	if status, _ := f.startNext(f.comp); status != http.StatusConflict {
		t.Fatalf("start next while last segment is incomplete: %d", status)
	}
	if _, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d", f.comp), nil); out["status"] != "active" {
		t.Fatalf("finished too early: %v", out["status"])
	}
	if st, o := f.deviceAction(lane2, 6, "accept"); st != 200 {
		t.Fatalf("final accept %d %v", st, o)
	}
	_, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d", f.comp), nil)
	if out["status"] != "finished" {
		t.Fatalf("status %v", out["status"])
	}
	for _, p := range pairingsOf(t, f) {
		if !p {
			t.Fatal("pairing not revoked")
		}
	}
	status, sess := f.env.DoToken(lane1, "GET", "/api/scorer/session", nil)
	if status != http.StatusOK || sess["phase"] != "finished" {
		t.Fatalf("finished session: %d %v", status, sess)
	}
	if segments, _ := sess["segments"].([]any); len(segments) != 0 || sess["shooter_name"] != "" {
		t.Fatalf("finished session leaks scores: %v", sess)
	}
	if status, _ := f.env.DoToken(lane1, "GET", "/api/scorer/stream", nil); status != http.StatusGone {
		t.Fatalf("finished stream: %d", status)
	}
	if status, _ := f.startNext(f.comp); status != http.StatusConflict {
		t.Fatalf("start next on finished: %d", status)
	}
}

func pairingsOf(t *testing.T, f *scoringFixture) []bool {
	t.Helper()
	_, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d/pairings", f.comp), nil)
	var revoked []bool
	for _, p := range out["pairings"].([]any) {
		revoked = append(revoked, p.(map[string]any)["revoked"].(bool))
	}
	return revoked
}

func TestEntryChangesAreFrozenOnceActiveAndWithdrawalOnlyAtTheGate(t *testing.T) {
	f := newScoringFixture(t, 1, 2, 3)
	base := fmt.Sprintf("/api/matches/%d", f.comp)
	shooter := f.env.ID("POST", "/api/shooters", shooterBody("LATE"))
	if status, _ := f.env.Do("POST", base+"/entries", map[string]any{"shooter_id": shooter, "category": "Pistol", "lane": 9}); status != http.StatusConflict {
		t.Fatalf("add entry: %d", status)
	}
	update := fmt.Sprintf("%s/entries/%d", base, f.entries[2])
	if status, _ := f.env.Do("PUT", update, map[string]any{"category": "Pistol", "lane": 8}); status != http.StatusConflict {
		t.Fatalf("lane change: %d", status)
	}
	withdraw := fmt.Sprintf("%s/entries/%d/withdraw", base, f.entries[3])
	if status, _ := f.env.Do("POST", withdraw, nil); status != http.StatusConflict {
		t.Fatalf("withdraw mid segment: %d", status)
	}
	f.acceptAll(t, 1, 1, 2, 3)
	if status, _ := f.env.Do("POST", withdraw, nil); status != http.StatusOK && status != http.StatusNoContent {
		t.Fatalf("withdraw at the gate: %d", status)
	}
	if status, out := f.startNext(f.comp); status != http.StatusOK {
		t.Fatalf("start next after withdrawal: %d %v", status, out)
	}
	if status, _ := f.env.Do("POST", fmt.Sprintf("%s/entries/%d/withdraw", base, f.entries[2]), nil); status != http.StatusConflict {
		t.Fatalf("withdraw after next segment started: %d", status)
	}
	if got := f.session(t, f.pair(t, 1))["score_lane"]; got != float64(2) {
		t.Fatalf("score lane %v", got)
	}
}

func TestActiveMatchesKeepIndependentPointers(t *testing.T) {
	a := newScoringFixture(t, 1, 2)
	b := newScoringFixture(t, 1, 2)
	a.acceptAll(t, 1, 1, 2)
	if status, _ := a.startNext(a.comp); status != http.StatusOK {
		t.Fatalf("start next: %d", status)
	}
	if a.currentSegment(t) != 2 || b.currentSegment(t) != 1 {
		t.Fatalf("pointers %d %d", a.currentSegment(t), b.currentSegment(t))
	}
}

func TestCorrectionsAfterThePointerMovedOnAreAdminOnlyAndDoNotMoveThePointer(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	f.acceptAll(t, 1, 1, 2)
	f.startNext(f.comp)
	lane1 := f.pair(t, 1)

	if status, out := f.env.Do("POST", f.adminSegment(2, 1, "/unlock"), nil); status != http.StatusOK {
		t.Fatalf("unlock: %d %v", status, out)
	}
	if status, _ := f.env.DoToken(lane1, "PUT", "/api/scorer/segments/1", counts(9, 3)); status != http.StatusConflict {
		t.Fatalf("device edit of an earlier segment: %d", status)
	}
	if status, _ := f.deviceAction(lane1, 1, "lock"); status != http.StatusConflict {
		t.Fatalf("device lock of an earlier segment: %d", status)
	}
	if status, _ := f.env.Do("PUT", f.adminSegment(2, 1, ""), counts(9, 3)); status != http.StatusOK {
		t.Fatalf("admin edit: %d", status)
	}
	if status, out := f.env.Do("POST", f.adminSegment(2, 1, "/lock"), nil); status != http.StatusOK || out["status"] != "accepted" {
		t.Fatalf("admin relock: %d %v", status, out)
	}
	if got := f.currentSegment(t); got != 2 {
		t.Fatalf("pointer moved to %d", got)
	}
}

func TestUnlockOnAFinishedMatchLeavesItFinished(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	f.env.Do("POST", fmt.Sprintf("/api/matches/%d/status", f.comp), map[string]any{"status": "finished"})
	f.acceptAll(t, 1, 1, 2)
	if status, out := f.env.Do("POST", f.adminSegment(1, 1, "/unlock"), nil); status != http.StatusOK {
		t.Fatalf("unlock: %d %v", status, out)
	}
	if _, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d", f.comp), nil); out["status"] != "finished" {
		t.Fatalf("status %v", out["status"])
	}
	if got := f.segmentStatus(t, 1, 1); got != "open" {
		t.Fatalf("segment %q", got)
	}
}

func TestAdminAcceptanceOfTheLastSegmentFinishesTheMatch(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	for segment := 1; segment <= 6; segment++ {
		if segment > 1 {
			f.startNext(f.comp)
		}
		f.acceptAll(t, segment, 1, 2)
	}
	if _, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d", f.comp), nil); out["status"] != "finished" {
		t.Fatalf("status %v", out["status"])
	}
}

func TestCategoryCanBeEditedOnceActiveButTheLaneStaysFrozen(t *testing.T) {
	f := newScoringFixture(t, 1, 2, 3)
	update := fmt.Sprintf("/api/matches/%d/entries/%d", f.comp, f.entries[2])
	if status, out := f.env.Do("PUT", update, map[string]any{"category": "Revolver", "lane": 2}); status != http.StatusOK {
		t.Fatalf("category change: %d %v", status, out)
	}
	if status, _ := f.env.Do("PUT", update, map[string]any{"category": "Revolver", "lane": 8}); status != http.StatusConflict {
		t.Fatalf("lane change: %d", status)
	}
}
