package handler_test

import (
	"fmt"
	"net/http"
	"testing"
)

type scoringFixture struct {
	*fixture
	entries map[int]int
}

func newScoringFixture(t *testing.T, lanes ...int) *scoringFixture {
	t.Helper()
	f := &scoringFixture{fixture: newFixture(t, "150"), entries: map[int]int{}}
	for _, l := range lanes {
		f.entries[l] = f.addEntry(t, fmt.Sprintf("SHOOTER%d", l), l)
	}
	f.env.Do("POST", fmt.Sprintf("/api/matches/%d/status", f.comp), map[string]any{"status": "active"})
	return f
}

func (f *scoringFixture) pair(t *testing.T, lane int) string {
	t.Helper()
	status, out := f.env.Do("POST", fmt.Sprintf("/api/matches/%d/pairings", f.comp), map[string]any{"lane": lane})
	if status != http.StatusCreated {
		t.Fatalf("pair lane %d: %d %v", lane, status, out)
	}
	return out["token"].(string)
}

func (f *scoringFixture) segmentLocked(t *testing.T, lane, segment int) bool {
	t.Helper()
	_, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d/entries", f.comp), nil)
	for _, e := range out["entries"].([]any) {
		m := e.(map[string]any)
		if int(m["lane"].(float64)) == lane {
			return m["segments"].([]any)[segment-1].(map[string]any)["status"].(string) != "open"
		}
	}
	t.Fatalf("lane %d not found", lane)
	return false
}

func counts(x, ten int) map[string]any {
	return map[string]any{"x": x, "10": ten, "miss": 24 - x - ten}
}

func TestDeviceScoresOnlyScoreLane(t *testing.T) {
	f := newScoringFixture(t, 1, 3, 5)
	token := f.pair(t, 1)

	status, out := f.env.DoToken(token, "GET", "/api/scorer/session", nil)
	if status != http.StatusOK || int(out["lane"].(float64)) != 1 || int(out["score_lane"].(float64)) != 3 {
		t.Fatalf("session: %d %v", status, out)
	}
	if out["shooter_name"] != "A SHOOTER3" {
		t.Fatalf("shooter_name: %v", out["shooter_name"])
	}

	if status, out := f.env.DoToken(token, "PUT", "/api/scorer/segments/1", counts(10, 2)); status != http.StatusOK {
		t.Fatalf("set counts: %d %v", status, out)
	}
	if status, out := f.env.DoToken(token, "POST", "/api/scorer/segments/1/lock", nil); status != http.StatusOK {
		t.Fatalf("lock: %d %v", status, out)
	}
	if !f.segmentLocked(t, 3, 1) {
		t.Fatal("score lane segment should be locked")
	}
	if f.segmentLocked(t, 1, 1) || f.segmentLocked(t, 5, 1) {
		t.Fatal("other lanes must stay untouched")
	}

	t.Run("locked segment rejects further writes", func(t *testing.T) {
		if status, _ := f.env.DoToken(token, "PUT", "/api/scorer/segments/1", counts(12, 0)); status != http.StatusConflict {
			t.Fatalf("status %d", status)
		}
	})
	t.Run("counts must sum to expected shots", func(t *testing.T) {
		if status, _ := f.env.DoToken(f.pair(t, 3), "PUT", "/api/scorer/segments/1", map[string]any{"x": 5}); status != http.StatusUnprocessableEntity {
			t.Fatalf("status %d", status)
		}
	})
	t.Run("unknown token is rejected", func(t *testing.T) {
		if status, _ := f.env.DoToken("nope", "GET", "/api/scorer/session", nil); status != http.StatusUnauthorized {
			t.Fatalf("status %d", status)
		}
	})
}

func sessionNames(t *testing.T, f *scoringFixture, token string) []any {
	t.Helper()
	status, out := f.env.DoToken(token, "GET", "/api/scorer/session", nil)
	if status != http.StatusOK {
		t.Fatalf("session: %d %v", status, out)
	}
	names := []any{}
	for _, st := range out["segments"].([]any) {
		names = append(names, st.(map[string]any)["name"])
	}
	return names
}

func TestSessionCarriesSegmentNames(t *testing.T) {
	f := newScoringFixture(t, 1, 3)
	names := sessionNames(t, f, f.pair(t, 1))
	if len(names) != 6 || names[0] != "Match 1" || names[3] != "Match 4" || names[5] != "Match 5 · Stage 3+4" {
		t.Fatalf("150 names: %v", names)
	}

	plain := &scoringFixture{fixture: newFixture(t, "60"), entries: map[int]int{}}
	plain.entries[1] = plain.addEntry(t, "SHOOTER1", 1)
	plain.entries[3] = plain.addEntry(t, "SHOOTER3", 3)
	plain.env.Do("POST", fmt.Sprintf("/api/matches/%d/status", plain.comp), map[string]any{"status": "active"})
	if names := sessionNames(t, plain, plain.pair(t, 1)); len(names) != 2 || names[0] != "Stage 1+2" || names[1] != "Stage 3+4" {
		t.Fatalf("60 names: %v", names)
	}
}

func TestDevicePairedInDraftWaitsForStart(t *testing.T) {
	f := &scoringFixture{fixture: newFixture(t, "150"), entries: map[int]int{}}
	f.entries[1] = f.addEntry(t, "SHOOTER1", 1)
	f.entries[3] = f.addEntry(t, "SHOOTER3", 3)
	token := f.pair(t, 1)
	status, out := f.env.DoToken(token, "GET", "/api/scorer/session", nil)
	if status != http.StatusOK || out["phase"] != "not_started" || out["shooter_name"] != "" {
		t.Fatalf("draft session: %d %v", status, out)
	}
	if status, _ := f.env.DoToken(token, "PUT", "/api/scorer/segments/1", counts(5, 0)); status != http.StatusConflict {
		t.Fatalf("draft scoring status %d", status)
	}
	f.env.Do("POST", fmt.Sprintf("/api/matches/%d/status", f.comp), map[string]any{"status": "active"})
	status, out = f.env.DoToken(token, "GET", "/api/scorer/session", nil)
	if status != http.StatusOK || out["phase"] == "not_started" || int(out["score_lane"].(float64)) != 3 {
		t.Fatalf("active session: %d %v", status, out)
	}
}

func TestSessionFollowsLaneChanges(t *testing.T) {
	f := newScoringFixture(t, 1, 3, 5)
	token := f.pair(t, 1)
	f.acceptAll(t, 1, 1, 3, 5)
	status, _ := f.env.Do("POST", fmt.Sprintf("/api/matches/%d/entries/%d/withdraw", f.comp, f.entries[3]), nil)
	if status != http.StatusOK {
		t.Fatalf("withdraw status %d", status)
	}
	status, out := f.env.DoToken(token, "GET", "/api/scorer/session", nil)
	if status != http.StatusOK || int(out["score_lane"].(float64)) != 5 || out["shooter_name"] != "A SHOOTER5" {
		t.Fatalf("session after withdraw: %d %v", status, out)
	}

}

func TestAdminLocksCompleteSegment(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	p := fmt.Sprintf("/api/matches/%d/entries/%d/segments/1", f.comp, f.entries[2])
	if status, _ := f.env.Do("POST", p+"/lock", nil); status != http.StatusUnprocessableEntity {
		t.Fatalf("lock incomplete: %d", status)
	}
	f.env.Do("PUT", p, counts(12, 0))
	if status, out := f.env.Do("POST", p+"/lock", nil); status != http.StatusOK {
		t.Fatalf("lock: %d %v", status, out)
	}
	if !f.segmentLocked(t, 2, 1) {
		t.Fatal("not locked")
	}
	if status, _ := f.env.Do("POST", p+"/lock", nil); status != http.StatusConflict {
		t.Fatalf("lock twice: %d", status)
	}
}

func TestLockRecordsScorerAndAdminUnlock(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	token := f.pair(t, 1)
	f.env.DoToken(token, "PUT", "/api/scorer/segments/1", counts(12, 0))
	f.env.DoToken(token, "POST", "/api/scorer/segments/1/lock", nil)

	_, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d/entries", f.comp), nil)
	var segment map[string]any
	for _, e := range out["entries"].([]any) {
		m := e.(map[string]any)
		if int(m["lane"].(float64)) == 2 {
			segment = m["segments"].([]any)[0].(map[string]any)
		}
	}
	if segment["scorer_lane"] != float64(1) || segment["scorer_name"] != "A SHOOTER1" {
		t.Fatalf("lock record: %v", segment)
	}

	p := fmt.Sprintf("/api/matches/%d/entries/%d/segments/1", f.comp, f.entries[2])
	if status, _ := f.env.DoToken(token, "POST", "/api/scorer/segments/1/unlock", nil); status == http.StatusOK {
		t.Fatal("device must not unlock")
	}
	if status, _ := f.env.Do("PUT", p, counts(11, 1)); status != http.StatusOK {
		t.Fatalf("admin edit while locked: %d", status)
	}
	if !f.segmentLocked(t, 2, 1) {
		t.Fatal("admin edit must keep the segment locked")
	}
	if status, out := f.env.Do("POST", p+"/unlock", nil); status != http.StatusOK {
		t.Fatalf("unlock: %d %v", status, out)
	}
	if f.segmentLocked(t, 2, 1) {
		t.Fatal("still locked")
	}
	if status, out := f.env.Do("PUT", p, counts(11, 1)); status != http.StatusOK {
		t.Fatalf("admin direct scoring: %d %v", status, out)
	}
}

func TestPairingLifecycle(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	base := fmt.Sprintf("/api/matches/%d/pairings", f.comp)

	t.Run("empty lane cannot be paired", func(t *testing.T) {
		if status, _ := f.env.Do("POST", base, map[string]any{"lane": 7}); status != http.StatusConflict {
			t.Fatalf("status %d", status)
		}
	})

	first := f.pair(t, 1)
	second := f.pair(t, 1)
	t.Run("re-pairing revokes the previous device", func(t *testing.T) {
		if status, _ := f.env.DoToken(first, "GET", "/api/scorer/session", nil); status != http.StatusUnauthorized {
			t.Fatalf("old token status %d", status)
		}
		if status, _ := f.env.DoToken(second, "GET", "/api/scorer/session", nil); status != http.StatusOK {
			t.Fatalf("new token status %d", status)
		}
	})

	t.Run("revoke", func(t *testing.T) {
		_, out := f.env.Do("GET", base, nil)
		var id int
		for _, p := range out["pairings"].([]any) {
			m := p.(map[string]any)
			if !m["revoked"].(bool) {
				id = int(m["id"].(float64))
			}
		}
		if status, _ := f.env.Do("DELETE", fmt.Sprintf("%s/%d", base, id), nil); status != http.StatusNoContent {
			t.Fatalf("status %d", status)
		}
		if status, _ := f.env.DoToken(second, "GET", "/api/scorer/session", nil); status != http.StatusUnauthorized {
			t.Fatalf("status %d", status)
		}
	})

	t.Run("finish leaves devices a finished screen only", func(t *testing.T) {
		token := f.pair(t, 2)
		f.env.Do("POST", fmt.Sprintf("/api/matches/%d/status", f.comp), map[string]any{"status": "finished"})
		status, out := f.env.DoToken(token, "GET", "/api/scorer/session", nil)
		if status != http.StatusOK || out["phase"] != "finished" {
			t.Fatalf("status %d %v", status, out)
		}
		if out["match_id"] != float64(f.comp) || out["lane"] != float64(2) {
			t.Fatalf("finished session must identify the scorecard: %v", out)
		}
		if status, _ := f.env.DoToken(token, "PUT", "/api/scorer/segments/1", counts(12, 0)); status != http.StatusConflict {
			t.Fatalf("write after finish: %d", status)
		}
	})
}

func TestSingleLaneRequiresAdminScoring(t *testing.T) {
	f := newScoringFixture(t, 4)
	token := f.pair(t, 4)
	status, out := f.env.DoToken(token, "PUT", "/api/scorer/segments/1", counts(12, 0))
	if status != http.StatusForbidden || out["error"] != "admin scoring required" {
		t.Fatalf("got %d %v", status, out)
	}
	p := fmt.Sprintf("/api/matches/%d/entries/%d/segments/1", f.comp, f.entries[4])
	if status, _ := f.env.Do("PUT", p, counts(12, 0)); status != http.StatusOK {
		t.Fatalf("admin scoring status %d", status)
	}
}

func TestPairingRequiresAdmin(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	token := f.pair(t, 1)
	if status, _ := f.env.DoToken(token, "POST", fmt.Sprintf("/api/matches/%d/pairings", f.comp), map[string]any{"lane": 2}); status != http.StatusUnauthorized {
		t.Fatalf("status %d", status)
	}
}

func TestOutOfRangeHitsAreRejected(t *testing.T) {
	f := newScoringFixture(t, 1, 3)
	token := f.pair(t, 1)
	adminPath := fmt.Sprintf("/api/matches/%d/entries/%d/segments/1", f.comp, f.entries[3])
	const maxInt = 9223372036854775807

	oversized := map[string]any{"x": 13, "10": -1}
	wrapping := map[string]any{"x": maxInt, "10": maxInt, "9": 14}
	if status, out := f.env.DoToken(token, "PUT", "/api/scorer/segments/1", counts(7, 5)); status != http.StatusOK {
		t.Fatalf("seed: %d %v", status, out)
	}

	for name, body := range map[string]map[string]any{
		"oversized": {"x": 13},
		"negative":  oversized,
		"wrapping":  wrapping,
	} {
		t.Run(name, func(t *testing.T) {
			if status, _ := f.env.DoToken(token, "PUT", "/api/scorer/segments/1", body); status != http.StatusUnprocessableEntity {
				t.Fatalf("device status %d", status)
			}
			if status, _ := f.env.Do("PUT", adminPath, body); status != http.StatusUnprocessableEntity {
				t.Fatalf("admin status %d", status)
			}
		})
	}

	_, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d/entries", f.comp), nil)
	for _, e := range out["entries"].([]any) {
		m := e.(map[string]any)
		if int(m["lane"].(float64)) == 3 {
			s := m["segments"].([]any)[0].(map[string]any)
			if s["score"].(float64) != 120 {
				t.Fatalf("stored hits changed: %v", s)
			}
		}
	}
	if status, out := f.env.Do("PUT", adminPath, counts(12, 0)); status != http.StatusOK {
		t.Fatalf("valid admin: %d %v", status, out)
	}
}
