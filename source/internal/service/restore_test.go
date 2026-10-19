package service_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/restore"
	"github.com/sollie/matchmark/internal/service"
)

func TestRestoreIntoEmptyDatabase(t *testing.T) {
	f := newScoreFixture(t)
	ctx := context.Background()
	for _, n := range []int{1, 2} {
		f.fillStage(t, f.entry2, n)
	}
	f.fillStage(t, f.entry1, 1)
	for _, n := range []int{1, 2} {
		if err := f.svc.DeviceLockStage(ctx, service.Device{CompetitionID: f.compID, Lane: 1}, n); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.svc.AdminLockStage(ctx, f.compID, f.entry1, 1); err != nil {
		t.Fatal(err)
	}
	events, _, err := restore.Parse(strings.NewReader(f.buf.String()))
	if err != nil {
		t.Fatal(err)
	}
	plan := restore.Fold(events)

	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	repo := repository.New(conn)
	svc := service.New(repo)

	first, err := svc.RestoreScoreLog(ctx, plan)
	if err != nil {
		t.Fatal(err)
	}
	if first.Competitions != 1 || first.Entries != 2 || first.Stages != 3 {
		t.Fatalf("first = %+v", first)
	}
	second, err := svc.RestoreScoreLog(ctx, plan)
	if err != nil {
		t.Fatal(err)
	}
	if second != (service.RestoreResult{}) {
		t.Fatalf("second = %+v", second)
	}

	want := normalizedReport(t, f.repo, f.compID)
	got := normalizedReport(t, repo, f.compID)
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("report mismatch\nwant %+v\ngot  %+v", want, got)
	}
	for _, ref := range []struct {
		entry  int64
		number int
	}{{f.entry1, 1}, {f.entry2, 1}, {f.entry2, 2}} {
		w, err := f.repo.GetStage(ctx, f.compID, ref.entry, ref.number)
		if err != nil {
			t.Fatal(err)
		}
		g, err := repo.GetStage(ctx, f.compID, ref.entry, ref.number)
		if err != nil {
			t.Fatal(err)
		}
		w.ID, g.ID = 0, 0
		if !reflect.DeepEqual(w, g) {
			t.Fatalf("stage mismatch\nwant %+v\ngot  %+v", w, g)
		}
	}
	other, err := repo.GetStage(ctx, f.compID, f.entry1, 2)
	if err != nil || other.Locked {
		t.Fatalf("unscored stage = %+v, %v", other, err)
	}
}

func normalizedReport(t *testing.T, repo *repository.Repository, compID int64) []repository.ReportEntry {
	t.Helper()
	entries, err := repo.ReportEntries(context.Background(), compID)
	if err != nil {
		t.Fatal(err)
	}
	for i := range entries {
		entries[i].Shooter.ID = 0
		entries[i].ShooterID = 0
		entries[i].Shooter.Sex = ""
		entries[i].Shooter.Country = ""
	}
	return entries
}
