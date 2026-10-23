package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/sollie/matchmark/internal/apitest"
)

type sessionBody struct {
	Lane      int `json:"lane"`
	ScoreLane int `json:"score_lane"`
	Stages    []struct {
		Status string `json:"status"`
	} `json:"stages"`
}

func openScorerStream(f *scoringFixture, token string) *apitest.Stream {
	return f.env.OpenStream("/api/scorer/stream", http.Header{"Authorization": {"Bearer " + token}})
}

func nextSession(t *testing.T, s *apitest.Stream) sessionBody {
	t.Helper()
	for {
		ev, ok := s.Next(eventWait)
		if !ok {
			t.Fatal("no snapshot received")
		}
		if ev.Name != "snapshot" {
			continue
		}
		var body sessionBody
		if err := json.Unmarshal([]byte(ev.Data), &body); err != nil {
			t.Fatalf("decode snapshot: %v", err)
		}
		return body
	}
}

func TestScorerStreamRequiresDeviceToken(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	if s := f.env.OpenStream("/api/scorer/stream", nil); s.Status != http.StatusUnauthorized {
		t.Fatalf("no token status %d", s.Status)
	}
	if s := openScorerStream(f, "nope"); s.Status != http.StatusUnauthorized {
		t.Fatalf("bad token status %d", s.Status)
	}
}

func TestScorerStreamPushesAdminLockAndUnlock(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	token := f.pair(t, 1)
	s := openScorerStream(f, token)
	if s.Status != http.StatusOK {
		t.Fatalf("status %d", s.Status)
	}
	first := nextSession(t, s)
	if first.Lane != 1 || first.ScoreLane != 2 || first.Stages[0].Status != "open" {
		t.Fatalf("first %+v", first)
	}
	base := fmt.Sprintf("/api/competitions/%d/entries/%d/stages/1", f.comp, f.entries[2])
	if status, out := f.env.Do("PUT", base, counts(12, 0)); status != http.StatusOK {
		t.Fatalf("set: %d %v", status, out)
	}
	if status, out := f.env.Do("POST", base+"/lock", nil); status != http.StatusOK {
		t.Fatalf("lock: %d %v", status, out)
	}
	for nextSession(t, s).Stages[0].Status == "open" {
	}
	if status, out := f.env.Do("POST", base+"/unlock", nil); status != http.StatusOK && status != http.StatusNoContent {
		t.Fatalf("unlock: %d %v", status, out)
	}
	for nextSession(t, s).Stages[0].Status != "open" {
	}
}

func TestScorerStreamPushesScoreLaneChange(t *testing.T) {
	f := newScoringFixture(t, 1, 2, 3)
	token := f.pair(t, 1)
	s := openScorerStream(f, token)
	if got := nextSession(t, s).ScoreLane; got != 2 {
		t.Fatalf("score lane %d", got)
	}
	if status, out := f.env.Do("POST", fmt.Sprintf("/api/competitions/%d/entries/%d/withdraw", f.comp, f.entries[2]), nil); status != http.StatusNoContent && status != http.StatusOK {
		t.Fatalf("delete: %d %v", status, out)
	}
	for nextSession(t, s).ScoreLane != 3 {
	}
}

func TestScorerStreamSendsRevokedWhenPairingRevoked(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	token := f.pair(t, 1)
	s := openScorerStream(f, token)
	nextSession(t, s)
	_, out := f.env.Do("GET", fmt.Sprintf("/api/competitions/%d/pairings", f.comp), nil)
	id := int(out["pairings"].([]any)[0].(map[string]any)["id"].(float64))
	if status, _ := f.env.Do("DELETE", fmt.Sprintf("/api/competitions/%d/pairings/%d", f.comp, id), nil); status != http.StatusNoContent {
		t.Fatalf("revoke status %d", status)
	}
	awaitEvent(t, s, "revoked")
	if !s.Closed(eventWait) {
		t.Fatal("stream should end after revoked")
	}
}

func TestScorerStreamSendsRevokedWhenCompetitionFinishes(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	token := f.pair(t, 1)
	s := openScorerStream(f, token)
	nextSession(t, s)
	f.env.Do("POST", fmt.Sprintf("/api/competitions/%d/status", f.comp), map[string]any{"status": "finished"})
	awaitEvent(t, s, "revoked")
	if !s.Closed(eventWait) {
		t.Fatal("stream should end after revoked")
	}
}

func TestDeviceWritesFollowCurrentScoreLane(t *testing.T) {
	f := newScoringFixture(t, 1, 2, 3)
	token := f.pair(t, 1)
	if status, _ := f.env.Do("POST", fmt.Sprintf("/api/competitions/%d/entries/%d/withdraw", f.comp, f.entries[2]), nil); status != http.StatusNoContent && status != http.StatusOK {
		t.Fatalf("delete status %d", status)
	}
	if status, out := f.env.DoToken(token, "PUT", "/api/scorer/stages/1", counts(12, 0)); status != http.StatusOK {
		t.Fatalf("set: %d %v", status, out)
	}
	_, out := f.env.Do("GET", fmt.Sprintf("/api/competitions/%d/entries", f.comp), nil)
	for _, e := range out["entries"].([]any) {
		m := e.(map[string]any)
		if int(m["lane"].(float64)) == 3 && m["stages"].([]any)[0].(map[string]any)["shots_entered"].(float64) != 12 {
			t.Fatalf("lane 3 entry %v", m)
		}
	}
}
