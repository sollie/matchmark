package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/sollie/matchmark/internal/apitest"
)

func adminStreamPath(f *fixture) string {
	return fmt.Sprintf("/api/matches/%d/stream", f.comp)
}

func awaitEvent(t *testing.T, s *apitest.Stream, name string) {
	t.Helper()
	for {
		ev, ok := s.Next(eventWait)
		if !ok {
			t.Fatalf("no %q event received", name)
		}
		if ev.Name == name {
			return
		}
	}
}

func TestAdminStreamRequiresAdminSession(t *testing.T) {
	f := newFixture(t, "150")
	status, _ := publicGet(t, f, adminStreamPath(f))
	if status != http.StatusUnauthorized {
		t.Fatalf("status %d", status)
	}
}

func TestAdminStreamPingsOnConnectAndAfterScorerLock(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	token := f.pair(t, 1)
	s := f.env.OpenStream(adminStreamPath(f.fixture), nil)
	if s.Status != http.StatusOK {
		t.Fatalf("status %d", s.Status)
	}
	awaitEvent(t, s, "changed")
	if status, out := f.env.DoToken(token, "PUT", "/api/scorer/stages/1", counts(10, 2)); status != http.StatusOK {
		t.Fatalf("set counts: %d %v", status, out)
	}
	awaitEvent(t, s, "changed")
	if status, out := f.env.DoToken(token, "POST", "/api/scorer/stages/1/lock", nil); status != http.StatusOK {
		t.Fatalf("lock: %d %v", status, out)
	}
	awaitEvent(t, s, "changed")
}
