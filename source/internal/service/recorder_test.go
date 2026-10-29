package service_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

func TestChangeRecorderOrdersAuditScoreLogThenLive(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	repo := repository.New(conn)
	var out bytes.Buffer
	logger := newLogger(&out)
	f := buildFixture(t, repo, &out, logger)
	f.fillSegment(t, f.entry1, 1)
	if _, err := f.svc.AdminLockSegment(context.Background(), f.compID, f.entry1, 1); err != nil {
		t.Fatal(err)
	}

	out.Reset()
	bus := service.NewBroadcaster(testWindow)
	sub := bus.Subscribe(f.compID)
	defer sub.Close()
	rec := service.NewChangeRecorder(service.NewAudit(repo, logger), service.NewScoreLog(repo, logger), bus)
	rec.Record(context.Background(), service.Change{
		Audit:   service.AuditEvent{Action: "entry.update", Target: "entry:1"},
		MatchID: f.compID,
		Live:    service.LiveMatch,
		Reemit:  &repository.LockedSegmentFilter{MatchID: f.compID},
	})

	audit := strings.Index(out.String(), `"msg":"audit"`)
	scored := strings.Index(out.String(), `"msg":"segment_status"`)
	if audit < 0 || scored < 0 || audit > scored {
		t.Fatalf("want audit before score log, got audit=%d score=%d", audit, scored)
	}
	expectPing(t, sub)
}

func TestChangeRecorderSwallowsScoreLogFailure(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	repo := repository.New(conn)
	var out bytes.Buffer
	logger := newLogger(&out)
	bus := service.NewBroadcaster(testWindow)
	sub := bus.Subscribe(99)
	defer sub.Close()
	rec := service.NewChangeRecorder(service.NewAudit(repo, logger), service.NewScoreLog(repo, logger), bus)
	rec.Record(context.Background(), service.Change{
		Audit:   service.AuditEvent{Action: "segment.lock", Target: "segment:99/99/1"},
		MatchID: 99,
		Live:    service.LiveMatch,
		Segment: &service.SegmentEvent{MatchID: 99, EntryID: 99, SegmentNumber: 1},
	})
	if !strings.Contains(out.String(), "score log context failed") {
		t.Fatalf("expected logged failure, got %s", out.String())
	}
	expectPing(t, sub)
}
