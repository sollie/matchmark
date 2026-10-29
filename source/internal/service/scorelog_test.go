package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

type logLine struct {
	Msg           string `json:"msg"`
	Level         string `json:"level"`
	SchemaVersion int    `json:"schema_version"`
	Event         string `json:"event"`
	Status        string `json:"status"`
	AcceptedBy    string `json:"accepted_by"`
	EmittedAt     string `json:"emitted_at"`
	Seq           int64  `json:"seq"`
	Competition   struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"competition"`
	Match struct {
		ID           int64  `json:"id"`
		Name         string `json:"name"`
		Date         string `json:"date"`
		CourseOfFire struct {
			Key        string `json:"key"`
			Discipline string `json:"discipline"`
			TotalShots int    `json:"total_shots"`
		} `json:"course_of_fire"`
	} `json:"match"`
	Entry struct {
		ID       int64  `json:"id"`
		Lane     int    `json:"lane"`
		Category string `json:"category"`
		Shooter  struct {
			WAID      string `json:"waid"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Club      string `json:"club"`
		} `json:"shooter"`
	} `json:"entry"`
	Segment struct {
		Number        int `json:"number"`
		ExpectedShots int `json:"expected_shots"`
	} `json:"segment"`
	RingCounts []int  `json:"ring_counts"`
	LockedBy   any    `json:"locked_by"`
	ScorerName string `json:"scorer_name"`
	LockedAt   string `json:"locked_at"`
}

type scoreFixture struct {
	svc    *service.Service
	repo   *repository.Repository
	buf    *bytes.Buffer
	compID int64
	entry1 int64
	entry2 int64
}

func newScoreFixture(t *testing.T) *scoreFixture {
	t.Helper()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	repo := repository.New(conn)
	buf := &bytes.Buffer{}
	return buildFixture(t, repo, buf, newLogger(buf))
}

func TestSegmentLockedLines(t *testing.T) {
	f := newScoreFixture(t)
	ctx := context.Background()
	f.fillSegment(t, f.entry2, 1)
	if _, err := f.svc.DeviceLockSegment(ctx, service.Device{MatchID: f.compID, Lane: 1}, 1); err != nil {
		t.Fatal(err)
	}
	lines := f.lines(t)
	if len(lines) != 1 {
		t.Fatalf("lines = %d", len(lines))
	}
	l := lines[0]
	if l.Msg != "segment_status" || l.Event != "segment_status" || l.Status != "awaiting_acceptance" || l.Level != "INFO" || l.SchemaVersion != 4 {
		t.Fatalf("header = %+v", l)
	}
	if !emittedAtPattern.MatchString(l.EmittedAt) {
		t.Fatalf("emitted_at = %q", l.EmittedAt)
	}
	if l.Seq < 1 {
		t.Fatalf("seq = %d", l.Seq)
	}
	if l.Match.ID != f.compID || l.Match.Name != "Spring Match" || l.Match.Date != "2026-10-19" ||
		l.Match.CourseOfFire.Key != "150" || l.Match.CourseOfFire.Discipline != "1500" || l.Match.CourseOfFire.TotalShots != 150 {
		t.Fatalf("match = %+v", l.Match)
	}
	if l.Entry.ID != f.entry2 || l.Entry.Lane != 2 || l.Entry.Category != "Pistol" ||
		l.Entry.Shooter.WAID != "W2" || l.Entry.Shooter.FirstName != "Bo" || l.Entry.Shooter.LastName != "BERG" || l.Entry.Shooter.Club != "Club B" {
		t.Fatalf("entry = %+v", l.Entry)
	}
	if l.Segment.Number != 1 || l.Segment.ExpectedShots != 24 {
		t.Fatalf("segment = %+v", l.Segment)
	}
	if len(l.RingCounts) != 7 || l.RingCounts[0] != 4 || l.RingCounts[1] != 8 || l.RingCounts[2] != 12 {
		t.Fatalf("ring_counts = %v", l.RingCounts)
	}
	if l.LockedBy != float64(1) || l.ScorerName != "Al ALM" {
		t.Fatalf("locked_by = %v scorer = %q", l.LockedBy, l.ScorerName)
	}
	if ts, err := time.Parse(time.RFC3339, l.LockedAt); err != nil || ts.Location() != time.UTC {
		t.Fatalf("locked_at = %q", l.LockedAt)
	}
}

func TestAdminLockLine(t *testing.T) {
	f := newScoreFixture(t)
	f.fillSegment(t, f.entry1, 2)
	if _, err := f.svc.AdminLockSegment(service.WithActor(context.Background(), service.Actor{UserID: 1, Username: "carol"}), f.compID, f.entry1, 2); err != nil {
		t.Fatal(err)
	}
	lines := f.lines(t)
	if len(lines) != 1 {
		t.Fatalf("lines = %d", len(lines))
	}
	if lines[0].LockedBy != "carol" || lines[0].ScorerName != "carol" || lines[0].Segment.Number != 2 || lines[0].Entry.ID != f.entry1 {
		t.Fatalf("line = %+v", lines[0])
	}
}

func TestNoLineOnFailedLock(t *testing.T) {
	f := newScoreFixture(t)
	ctx := context.Background()
	dev := service.Device{MatchID: f.compID, Lane: 1}
	if _, err := f.svc.DeviceLockSegment(ctx, dev, 1); !errors.Is(err, service.ErrInvalid) {
		t.Fatalf("err = %v", err)
	}
	if _, err := f.svc.AdminLockSegment(ctx, f.compID, f.entry2, 1); !errors.Is(err, service.ErrInvalid) {
		t.Fatalf("err = %v", err)
	}
	f.fillSegment(t, f.entry2, 1)
	if _, err := f.svc.DeviceLockSegment(ctx, dev, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.DeviceLockSegment(ctx, dev, 1); !errors.Is(err, service.ErrSegmentLocked) {
		t.Fatalf("err = %v", err)
	}
	if _, err := f.svc.AdminLockSegment(ctx, f.compID, f.entry2, 1); !errors.Is(err, service.ErrSegmentLocked) {
		t.Fatalf("err = %v", err)
	}
	if n := len(f.lines(t)); n != 1 {
		t.Fatalf("lines = %d", n)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestLockSucceedsWhenLogWriterFails(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	f := buildFixture(t, repository.New(conn), &bytes.Buffer{}, newLogger(failingWriter{}))
	f.fillSegment(t, f.entry2, 1)
	if _, err := f.svc.DeviceLockSegment(context.Background(), service.Device{MatchID: f.compID, Lane: 1}, 1); err != nil {
		t.Fatalf("err = %v", err)
	}
	entries, err := f.repo.ListEntries(context.Background(), f.compID)
	if err != nil {
		t.Fatal(err)
	}
	if entries[1].Segments[0].Status != repository.StatusAwaitingAcceptance {
		t.Fatal("segment not locked")
	}
}

func TestReplayMatchesDatabase(t *testing.T) {
	f := newScoreFixture(t)
	ctx := context.Background()
	for _, n := range []int{1, 2} {
		f.fillSegment(t, f.entry2, n)
	}
	f.fillSegment(t, f.entry1, 1)
	if _, err := f.svc.DeviceLockSegment(ctx, service.Device{MatchID: f.compID, Lane: 1}, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.AdminLockSegment(ctx, f.compID, f.entry1, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.AdminAcceptSegment(ctx, f.compID, f.entry2, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.StartNextSegment(ctx, f.compID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.DeviceLockSegment(ctx, service.Device{MatchID: f.compID, Lane: 1}, 2); err != nil {
		t.Fatal(err)
	}
	folded := replayScoreLog(t, f.buf.String())
	entries, err := f.repo.ListEntries(ctx, f.compID)
	if err != nil {
		t.Fatal(err)
	}
	locked := 0
	for _, e := range entries {
		for _, st := range e.Segments {
			if st.Status == repository.StatusOpen {
				continue
			}
			locked++
			got, ok := folded[segmentKey{f.compID, e.ID, st.Number}]
			if !ok {
				t.Fatalf("missing %d/%d", e.ID, st.Number)
			}
			c := st.Counts
			want := []int{c.X, c.Ten, c.Nine, c.Eight, c.Seven, c.Zero, c.Miss}
			if !equalInts(got.RingCounts, want) || got.Segment.ExpectedShots != st.ExpectedShots ||
				got.Entry.Lane != e.Lane || got.Entry.Category != e.Category {
				t.Fatalf("mismatch %d/%d: %+v vs %+v", e.ID, st.Number, got, st)
			}
		}
	}
	if locked != 3 || len(folded) != 3 {
		t.Fatalf("locked = %d folded = %d", locked, len(folded))
	}
}

var emittedAtPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$`)

func newerOrEqual(a, b logLine) bool {
	if a.EmittedAt != b.EmittedAt {
		return a.EmittedAt > b.EmittedAt
	}
	return a.Seq >= b.Seq
}

type segmentKey struct {
	comp, entry int64
	segment     int
}

func replayScoreLog(t *testing.T, raw string) map[segmentKey]logLine {
	t.Helper()
	out := map[segmentKey]logLine{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if line == "" {
			continue
		}
		var l logLine
		if err := json.Unmarshal([]byte(line), &l); err != nil {
			t.Fatal(err)
		}
		if l.Event == "" {
			continue
		}
		k := segmentKey{l.Match.ID, l.Entry.ID, l.Segment.Number}
		if prev, ok := out[k]; ok && !newerOrEqual(l, prev) {
			continue
		}
		out[k] = l
	}
	return out
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func newLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, nil))
}

func buildFixture(t *testing.T, repo *repository.Repository, buf *bytes.Buffer, logger *slog.Logger) *scoreFixture {
	t.Helper()
	f := buildDraftFixture(t, repo, buf, logger)
	if err := f.svc.SetMatchStatus(context.Background(), f.compID, service.StatusActive); err != nil {
		t.Fatal(err)
	}
	return f
}

func newDraftScoreFixture(t *testing.T) *scoreFixture {
	t.Helper()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	buf := &bytes.Buffer{}
	return buildDraftFixture(t, repository.New(conn), buf, newLogger(buf))
}

func buildDraftFixture(t *testing.T, repo *repository.Repository, buf *bytes.Buffer, logger *slog.Logger) *scoreFixture {
	t.Helper()
	ctx := context.Background()
	svc := service.New(repo, service.WithScoreLog(logger))
	s1, err := svc.CreateShooter(ctx, repository.Shooter{FirstName: "Al", LastName: "ALM", Country: "NOR", Club: "Club A", WAID: "W1"})
	if err != nil {
		t.Fatal(err)
	}
	s2, err := svc.CreateShooter(ctx, repository.Shooter{FirstName: "Bo", LastName: "BERG", Country: "NOR", Club: "Club B", WAID: "W2"})
	if err != nil {
		t.Fatal(err)
	}
	compID, err := svc.CreateMatch(ctx, repository.Match{Name: "Spring Match", Date: "2026-10-19", Course: "150"})
	if err != nil {
		t.Fatal(err)
	}
	e1, err := svc.AddEntry(ctx, repository.Entry{MatchID: compID, ShooterID: s1, Category: "Revolver", Lane: 1})
	if err != nil {
		t.Fatal(err)
	}
	e2, err := svc.AddEntry(ctx, repository.Entry{MatchID: compID, ShooterID: s2, Category: "Pistol", Lane: 2})
	if err != nil {
		t.Fatal(err)
	}
	return &scoreFixture{svc: svc, repo: repo, buf: buf, compID: compID, entry1: e1, entry2: e2}
}

func (f *scoreFixture) fillSegment(t *testing.T, entryID int64, segment int) {
	t.Helper()
	seg, err := f.repo.GetSegment(context.Background(), f.compID, entryID, segment)
	if err != nil {
		t.Fatal(err)
	}
	shots := seg.ExpectedShots
	err = f.svc.SetRingCounts(context.Background(), f.compID, entryID, segment, repository.RingCounts{X: shots / 6, Ten: shots / 3, Nine: shots - shots/6 - shots/3})
	if err != nil {
		t.Fatal(err)
	}
}

func (f *scoreFixture) lines(t *testing.T) []logLine {
	t.Helper()
	var out []logLine
	for _, line := range strings.Split(strings.TrimSpace(f.buf.String()), "\n") {
		if line == "" {
			continue
		}
		var l logLine
		if err := json.Unmarshal([]byte(line), &l); err != nil {
			t.Fatal(err)
		}
		out = append(out, l)
	}
	return out
}

func TestUnlockLine(t *testing.T) {
	f := newScoreFixture(t)
	ctx := context.Background()
	f.fillSegment(t, f.entry1, 1)
	if _, err := f.svc.AdminLockSegment(ctx, f.compID, f.entry1, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.UnlockSegment(ctx, f.compID, f.entry1, 1); err != nil {
		t.Fatal(err)
	}
	lines := f.lines(t)
	if len(lines) != 2 {
		t.Fatalf("lines = %d", len(lines))
	}
	l := lines[1]
	if l.Event != "segment_status" || l.Status != "open" || l.LockedBy != nil || l.Level != "INFO" || l.Entry.Shooter.WAID != "W1" ||
		l.Match.Name != "Spring Match" || l.Segment.Number != 1 || !equalInts(l.RingCounts, []int{4, 8, 12, 0, 0, 0, 0}) {
		t.Fatalf("line = %+v", l)
	}
	if got := replayScoreLog(t, f.buf.String())[segmentKey{f.compID, f.entry1, 1}]; got.Status != "open" {
		t.Fatalf("replay = %+v", got)
	}
}

func TestNoUnlockLineOnMissingSegment(t *testing.T) {
	f := newScoreFixture(t)
	if _, err := f.svc.UnlockSegment(context.Background(), f.compID, f.entry1, 99); err == nil {
		t.Fatal("expected error")
	}
	if n := len(f.lines(t)); n != 0 {
		t.Fatalf("lines = %d", n)
	}
}

func TestAdminEditOfLockedSegmentReEmitsLock(t *testing.T) {
	f := newScoreFixture(t)
	ctx := context.Background()
	f.fillSegment(t, f.entry2, 1)
	if _, err := f.svc.DeviceLockSegment(ctx, service.Device{MatchID: f.compID, Lane: 1}, 1); err != nil {
		t.Fatal(err)
	}
	edit := repository.RingCounts{X: 2, Ten: 2, Nine: 20}
	if err := f.svc.AdminSetRingCounts(service.WithActor(ctx, service.Actor{UserID: 1, Username: "carol"}), f.compID, f.entry2, 1, edit); err != nil {
		t.Fatal(err)
	}
	lines := f.lines(t)
	if len(lines) != 2 {
		t.Fatalf("lines = %d", len(lines))
	}
	first, second := lines[0], lines[1]
	if second.Event != "segment_status" || second.Status != "accepted" || second.LockedBy != "carol" || second.ScorerName != "carol" ||
		!equalInts(second.RingCounts, []int{2, 2, 20, 0, 0, 0, 0}) {
		t.Fatalf("line = %+v", second)
	}
	if second.Seq <= first.Seq || second.LockedAt != first.LockedAt {
		t.Fatalf("emitted %q/%q locked %q/%q", first.EmittedAt, second.EmittedAt, first.LockedAt, second.LockedAt)
	}
	got := replayScoreLog(t, f.buf.String())[segmentKey{f.compID, f.entry2, 1}]
	if !equalInts(got.RingCounts, []int{2, 2, 20, 0, 0, 0, 0}) {
		t.Fatalf("replay = %+v", got)
	}
}

func TestAdminEditOfUnlockedSegmentEmitsNothing(t *testing.T) {
	f := newScoreFixture(t)
	if err := f.svc.AdminSetRingCounts(context.Background(), f.compID, f.entry1, 1, repository.RingCounts{X: 24}); err != nil {
		t.Fatal(err)
	}
	if n := len(f.lines(t)); n != 0 {
		t.Fatalf("lines = %d", n)
	}
}

func (f *scoreFixture) lockedAt(t *testing.T) map[segmentKey]logLine {
	t.Helper()
	return replayScoreLog(t, f.buf.String())
}

func TestEntryEditReemitsLockedSegments(t *testing.T) {
	f := newDraftScoreFixture(t)
	ctx := context.Background()
	for _, e := range []int64{f.entry1, f.entry2} {
		for _, n := range []int{1, 2} {
			f.fillSegment(t, e, n)
			if _, err := f.svc.AdminLockSegment(ctx, f.compID, e, n); err != nil {
				t.Fatal(err)
			}
		}
	}
	before := f.lockedAt(t)
	n := len(f.lines(t))
	if _, err := f.svc.UpdateEntry(ctx, f.compID, f.entry2, "Revolver", 5); err != nil {
		t.Fatal(err)
	}
	if got := len(f.lines(t)) - n; got != 2 {
		t.Fatalf("new lines = %d", got)
	}
	after := f.lockedAt(t)
	if len(after) != 4 {
		t.Fatalf("folded = %d", len(after))
	}
	for k, l := range after {
		if k.entry != f.entry2 {
			if l.Seq != before[k].Seq {
				t.Fatalf("unaffected entry re-emitted: %+v", k)
			}
			continue
		}
		if l.Entry.Lane != 5 || l.Entry.Category != "Revolver" {
			t.Fatalf("context = %+v", l.Entry)
		}
		if l.LockedBy != before[k].LockedBy || l.LockedAt != before[k].LockedAt || l.ScorerName != before[k].ScorerName {
			t.Fatalf("provenance changed: %+v vs %+v", l, before[k])
		}
		if l.Seq <= before[k].Seq {
			t.Fatalf("seq not newer: %d vs %d", l.Seq, before[k].Seq)
		}
	}
}

func TestShooterEditReemitsAcrossMatches(t *testing.T) {
	f := newScoreFixture(t)
	ctx := context.Background()
	f.fillSegment(t, f.entry2, 1)
	if _, err := f.svc.AdminLockSegment(ctx, f.compID, f.entry2, 1); err != nil {
		t.Fatal(err)
	}
	n := len(f.lines(t))
	entry, err := f.repo.GetEntry(ctx, f.compID, f.entry2)
	if err != nil {
		t.Fatal(err)
	}
	sh, err := f.repo.GetShooter(ctx, entry.ShooterID)
	if err != nil {
		t.Fatal(err)
	}
	sh.Club = "Club Z"
	sh.FirstName = "Bob"
	if err := f.svc.UpdateShooter(ctx, sh); err != nil {
		t.Fatal(err)
	}
	if got := len(f.lines(t)) - n; got != 1 {
		t.Fatalf("new lines = %d", got)
	}
	l := f.lockedAt(t)[segmentKey{f.compID, f.entry2, 1}]
	if l.Entry.Shooter.Club != "Club Z" || l.Entry.Shooter.FirstName != "Bob" {
		t.Fatalf("shooter = %+v", l.Entry.Shooter)
	}
}

func TestMatchEditReemitsLockedSegments(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	ctx := context.Background()
	buf := &bytes.Buffer{}
	repo := repository.New(conn)
	svc := service.New(repo, service.WithScoreLog(newLogger(buf)))
	sh, err := svc.CreateShooter(ctx, repository.Shooter{FirstName: "Al", LastName: "ALM", Country: "NOR", Club: "Club A", WAID: "W1"})
	if err != nil {
		t.Fatal(err)
	}
	compID, err := svc.CreateMatch(ctx, repository.Match{Name: "Spring Match", Date: "2026-10-19", Course: "150"})
	if err != nil {
		t.Fatal(err)
	}
	e1, err := svc.AddEntry(ctx, repository.Entry{MatchID: compID, ShooterID: sh, Category: "Revolver", Lane: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SetRingCounts(ctx, compID, e1, 1, repository.RingCounts{X: 4, Ten: 8, Nine: 12}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AdminLockSegment(ctx, compID, e1, 1); err != nil {
		t.Fatal(err)
	}
	n := strings.Count(buf.String(), "\n")
	if err := svc.UpdateMatch(ctx, repository.Match{ID: compID, Name: "Autumn Match", Date: "2026-11-01", Course: "150"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(buf.String(), "\n") - n; got != 1 {
		t.Fatalf("new lines = %d", got)
	}
	l := replayScoreLog(t, buf.String())[segmentKey{compID, e1, 1}]
	if l.Match.Name != "Autumn Match" || l.Match.Date != "2026-11-01" {
		t.Fatalf("match = %+v", l.Match)
	}
}

func TestSeqStrictlyIncreasesAndReplayIgnoresEqualTimestamps(t *testing.T) {
	f := newScoreFixture(t)
	ctx := context.Background()
	f.fillSegment(t, f.entry1, 1)
	if _, err := f.svc.AdminLockSegment(ctx, f.compID, f.entry1, 1); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if err := f.svc.AdminSetRingCounts(ctx, f.compID, f.entry1, 1, repository.RingCounts{X: i, Ten: 24 - i}); err != nil {
			t.Fatal(err)
		}
	}
	lines := f.lines(t)
	for i := 1; i < len(lines); i++ {
		if lines[i].Seq <= lines[i-1].Seq {
			t.Fatalf("seq %d after %d", lines[i].Seq, lines[i-1].Seq)
		}
	}
	got := replayScoreLog(t, f.buf.String())[segmentKey{f.compID, f.entry1, 1}]
	if !equalInts(got.RingCounts, []int{4, 20, 0, 0, 0, 0, 0}) {
		t.Fatalf("replay = %+v", got)
	}
}

func TestUnlockLineKeepsPreUnlockProvenance(t *testing.T) {
	f := newScoreFixture(t)
	ctx := context.Background()
	f.fillSegment(t, f.entry2, 1)
	if _, err := f.svc.DeviceLockSegment(ctx, service.Device{MatchID: f.compID, Lane: 1}, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.UnlockSegment(ctx, f.compID, f.entry2, 1); err != nil {
		t.Fatal(err)
	}
	lines := f.lines(t)
	locked, unlocked := lines[0], lines[1]
	if unlocked.Status != "open" || unlocked.LockedBy != nil || unlocked.ScorerName != "" || unlocked.LockedAt != "" || locked.Status != "awaiting_acceptance" {
		t.Fatalf("unlock line = %+v", unlocked)
	}
}

func TestCourseChangeRejectedWhenLockedSegmentsExist(t *testing.T) {
	f := newScoreFixture(t)
	ctx := context.Background()
	f.fillSegment(t, f.entry1, 1)
	if _, err := f.svc.AdminLockSegment(ctx, f.compID, f.entry1, 1); err != nil {
		t.Fatal(err)
	}
	n := len(f.lines(t))
	err := f.svc.UpdateMatch(ctx, repository.Match{ID: f.compID, Name: "Spring Match", Date: "2026-10-19", Course: "60"})
	if !errors.Is(err, service.ErrConflict) {
		t.Fatalf("err = %v", err)
	}
	if got := len(f.lines(t)); got != n {
		t.Fatalf("lines = %d", got)
	}
	if l := f.lockedAt(t)[segmentKey{f.compID, f.entry1, 1}]; l.Segment.ExpectedShots != 24 || l.Match.CourseOfFire.Key != "150" {
		t.Fatalf("line = %+v", l)
	}
}
