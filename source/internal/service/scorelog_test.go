package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
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
	EmittedAt     string `json:"emitted_at"`
	Competition   struct {
		ID           int64  `json:"id"`
		Name         string `json:"name"`
		Date         string `json:"date"`
		CourseOfFire struct {
			Key        string `json:"key"`
			Discipline string `json:"discipline"`
			TotalShots int    `json:"total_shots"`
		} `json:"course_of_fire"`
	} `json:"competition"`
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
	Stage struct {
		Number        int `json:"number"`
		ExpectedShots int `json:"expected_shots"`
	} `json:"stage"`
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

func TestStageLockedLines(t *testing.T) {
	f := newScoreFixture(t)
	ctx := context.Background()
	f.fillStage(t, f.entry2, 1)
	if err := f.svc.DeviceLockStage(ctx, service.Device{CompetitionID: f.compID, Lane: 1}, 1); err != nil {
		t.Fatal(err)
	}
	lines := f.lines(t)
	if len(lines) != 1 {
		t.Fatalf("lines = %d", len(lines))
	}
	l := lines[0]
	if l.Msg != "stage_locked" || l.Event != "stage_locked" || l.Level != "INFO" || l.SchemaVersion != 1 {
		t.Fatalf("header = %+v", l)
	}
	if ts, err := time.Parse(time.RFC3339, l.EmittedAt); err != nil || ts.Location() != time.UTC {
		t.Fatalf("emitted_at = %q", l.EmittedAt)
	}
	if l.Competition.ID != f.compID || l.Competition.Name != "Spring Match" || l.Competition.Date != "2026-10-19" ||
		l.Competition.CourseOfFire.Key != "150" || l.Competition.CourseOfFire.Discipline != "1500" || l.Competition.CourseOfFire.TotalShots != 150 {
		t.Fatalf("competition = %+v", l.Competition)
	}
	if l.Entry.ID != f.entry2 || l.Entry.Lane != 2 || l.Entry.Category != "Pistol" ||
		l.Entry.Shooter.WAID != "W2" || l.Entry.Shooter.FirstName != "Bo" || l.Entry.Shooter.LastName != "Berg" || l.Entry.Shooter.Club != "Club B" {
		t.Fatalf("entry = %+v", l.Entry)
	}
	if l.Stage.Number != 1 || l.Stage.ExpectedShots != 12 {
		t.Fatalf("stage = %+v", l.Stage)
	}
	if len(l.RingCounts) != 7 || l.RingCounts[0] != 2 || l.RingCounts[1] != 4 || l.RingCounts[2] != 6 {
		t.Fatalf("ring_counts = %v", l.RingCounts)
	}
	if l.LockedBy != float64(1) || l.ScorerName != "Al Alm" {
		t.Fatalf("locked_by = %v scorer = %q", l.LockedBy, l.ScorerName)
	}
	if ts, err := time.Parse(time.RFC3339, l.LockedAt); err != nil || ts.Location() != time.UTC {
		t.Fatalf("locked_at = %q", l.LockedAt)
	}
}

func TestAdminLockLine(t *testing.T) {
	f := newScoreFixture(t)
	f.fillStage(t, f.entry1, 2)
	if err := f.svc.AdminLockStage(context.Background(), f.compID, f.entry1, 2); err != nil {
		t.Fatal(err)
	}
	lines := f.lines(t)
	if len(lines) != 1 {
		t.Fatalf("lines = %d", len(lines))
	}
	if lines[0].LockedBy != "Admin" || lines[0].ScorerName != "Admin" || lines[0].Stage.Number != 2 || lines[0].Entry.ID != f.entry1 {
		t.Fatalf("line = %+v", lines[0])
	}
}

func TestNoLineOnFailedLock(t *testing.T) {
	f := newScoreFixture(t)
	ctx := context.Background()
	dev := service.Device{CompetitionID: f.compID, Lane: 1}
	if err := f.svc.DeviceLockStage(ctx, dev, 1); !errors.Is(err, service.ErrInvalid) {
		t.Fatalf("err = %v", err)
	}
	if err := f.svc.AdminLockStage(ctx, f.compID, f.entry2, 1); !errors.Is(err, service.ErrInvalid) {
		t.Fatalf("err = %v", err)
	}
	f.fillStage(t, f.entry2, 1)
	if err := f.svc.DeviceLockStage(ctx, dev, 1); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.DeviceLockStage(ctx, dev, 1); !errors.Is(err, service.ErrStageLocked) {
		t.Fatalf("err = %v", err)
	}
	if err := f.svc.AdminLockStage(ctx, f.compID, f.entry2, 1); !errors.Is(err, service.ErrStageLocked) {
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
	f.fillStage(t, f.entry2, 1)
	if err := f.svc.DeviceLockStage(context.Background(), service.Device{CompetitionID: f.compID, Lane: 1}, 1); err != nil {
		t.Fatalf("err = %v", err)
	}
	entries, err := f.repo.ListEntries(context.Background(), f.compID)
	if err != nil {
		t.Fatal(err)
	}
	if !entries[1].Stages[0].Locked {
		t.Fatal("stage not locked")
	}
}

func TestReplayMatchesDatabase(t *testing.T) {
	f := newScoreFixture(t)
	ctx := context.Background()
	for _, n := range []int{1, 2} {
		f.fillStage(t, f.entry2, n)
	}
	f.fillStage(t, f.entry1, 1)
	if err := f.svc.DeviceLockStage(ctx, service.Device{CompetitionID: f.compID, Lane: 1}, 1); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.DeviceLockStage(ctx, service.Device{CompetitionID: f.compID, Lane: 1}, 2); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.AdminLockStage(ctx, f.compID, f.entry1, 1); err != nil {
		t.Fatal(err)
	}
	folded := replayScoreLog(t, f.buf.String())
	entries, err := f.repo.ListEntries(ctx, f.compID)
	if err != nil {
		t.Fatal(err)
	}
	locked := 0
	for _, e := range entries {
		for _, st := range e.Stages {
			if !st.Locked {
				continue
			}
			locked++
			got, ok := folded[stageKey{f.compID, e.ID, st.Number}]
			if !ok {
				t.Fatalf("missing %d/%d", e.ID, st.Number)
			}
			c := st.Counts
			want := []int{c.X, c.Ten, c.Nine, c.Eight, c.Seven, c.Zero, c.Miss}
			if !equalInts(got.RingCounts, want) || got.Stage.ExpectedShots != st.ExpectedShots ||
				got.Entry.Lane != e.Lane || got.Entry.Category != e.Category {
				t.Fatalf("mismatch %d/%d: %+v vs %+v", e.ID, st.Number, got, st)
			}
		}
	}
	if locked != 3 || len(folded) != 3 {
		t.Fatalf("locked = %d folded = %d", locked, len(folded))
	}
}

type stageKey struct {
	comp, entry int64
	stage       int
}

func replayScoreLog(t *testing.T, raw string) map[stageKey]logLine {
	t.Helper()
	out := map[stageKey]logLine{}
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
		k := stageKey{l.Competition.ID, l.Entry.ID, l.Stage.Number}
		if prev, ok := out[k]; ok && prev.EmittedAt > l.EmittedAt {
			continue
		}
		if l.Event == "stage_unlocked" {
			delete(out, k)
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
	ctx := context.Background()
	svc := service.New(repo, service.WithScoreLog(logger))
	s1, err := svc.CreateShooter(ctx, repository.Shooter{FirstName: "Al", LastName: "Alm", Sex: "M", Country: "NOR", Club: "Club A", WAID: "W1"})
	if err != nil {
		t.Fatal(err)
	}
	s2, err := svc.CreateShooter(ctx, repository.Shooter{FirstName: "Bo", LastName: "Berg", Sex: "M", Country: "NOR", Club: "Club B", WAID: "W2"})
	if err != nil {
		t.Fatal(err)
	}
	compID, err := svc.CreateCompetition(ctx, repository.Competition{Name: "Spring Match", Date: "2026-10-19", Course: "150"})
	if err != nil {
		t.Fatal(err)
	}
	e1, err := svc.AddEntry(ctx, repository.Entry{CompetitionID: compID, ShooterID: s1, Category: "Revolver", Lane: 1})
	if err != nil {
		t.Fatal(err)
	}
	e2, err := svc.AddEntry(ctx, repository.Entry{CompetitionID: compID, ShooterID: s2, Category: "Pistol", Lane: 2})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SetCompetitionStatus(ctx, compID, service.StatusActive); err != nil {
		t.Fatal(err)
	}
	return &scoreFixture{svc: svc, repo: repo, buf: buf, compID: compID, entry1: e1, entry2: e2}
}

func (f *scoreFixture) fillStage(t *testing.T, entryID int64, stage int) {
	t.Helper()
	err := f.svc.SetRingCounts(context.Background(), f.compID, entryID, stage, repository.RingCounts{X: 2, Ten: 4, Nine: 6})
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
