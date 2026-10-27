package service_test

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/report"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/restore"
	"github.com/sollie/matchmark/internal/rulebook"
	"github.com/sollie/matchmark/internal/service"
)

func TestRestoreIntoEmptyDatabase(t *testing.T) {
	f := newScoreFixture(t)
	ctx := context.Background()
	for _, n := range []int{1, 2} {
		f.fillStage(t, f.entry2, n)
	}
	f.fillStage(t, f.entry1, 1)
	if _, err := f.svc.DeviceLockStage(ctx, service.Device{MatchID: f.compID, Lane: 1}, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.AdminLockStage(service.WithActor(ctx, service.Actor{UserID: 1, Username: "carol"}), f.compID, f.entry1, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.AdminAcceptStage(ctx, f.compID, f.entry2, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.StartNextStage(ctx, f.compID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.DeviceLockStage(ctx, service.Device{MatchID: f.compID, Lane: 1}, 2); err != nil {
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

	first, err := svc.RestoreScoreLog(ctx, plan, false)
	if err != nil {
		t.Fatal(err)
	}
	if first.Matches != 1 || first.Entries != 2 || first.Stages != 3 {
		t.Fatalf("first = %+v", first)
	}
	if restored, err := repo.GetMatch(ctx, f.compID); err != nil || restored.Status != "active" || restored.CurrentStage != 2 {
		t.Fatalf("restored match = %+v, err %v", restored, err)
	}
	wantContainer, err := f.repo.GetMatch(ctx, f.compID)
	if err != nil {
		t.Fatal(err)
	}
	wantName, err := f.repo.CompetitionName(ctx, wantContainer.CompetitionID)
	if err != nil {
		t.Fatal(err)
	}
	restoredMatch, _ := repo.GetMatch(ctx, f.compID)
	gotName, err := repo.CompetitionName(ctx, restoredMatch.CompetitionID)
	if err != nil || restoredMatch.CompetitionID != wantContainer.CompetitionID || gotName != wantName {
		t.Fatalf("restored competition = %d %q, want %d %q, err %v", restoredMatch.CompetitionID, gotName, wantContainer.CompetitionID, wantName, err)
	}
	second, err := svc.RestoreScoreLog(ctx, plan, true)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(second, restore.Applied{}) {
		t.Fatalf("second = %+v", second)
	}

	want := normalizedReport(t, f.repo, f.compID)
	got := normalizedReport(t, repo, f.compID)
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("report mismatch\nwant %+v\ngot  %+v", want, got)
	}
	wantCSV, wantRows := reportOutputs(t, f.repo, f.compID)
	gotCSV, gotRows := reportOutputs(t, repo, f.compID)
	if !bytes.Equal(wantCSV, gotCSV) {
		t.Fatalf("csv mismatch\nwant %s\ngot  %s", wantCSV, gotCSV)
	}
	if !reflect.DeepEqual(wantRows, gotRows) {
		t.Fatalf("xlsx mismatch\nwant %v\ngot  %v", wantRows, gotRows)
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
	if err != nil || other.Status != repository.StatusOpen {
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
		entries[i].Shooter.Country = ""
	}
	return entries
}

func mergePlan(name string, counts [7]int, stages ...int) restore.Plan {
	var scored []restore.ScoredStage
	for _, n := range stages {
		scored = append(scored, restore.ScoredStage{Number: n, ExpectedShots: 10, RingCounts: counts, Status: restore.StatusAccepted, AcceptedBy: "bob", AcceptedAt: "2026-10-19T10:00:00Z", LockedBy: "bob", ScorerName: "bob", LockedAt: "2026-10-19T10:00:00Z"})
	}
	return restore.Plan{Matches: []restore.RestoredMatch{{
		ID: 7, Name: name, Date: "2026-10-19", CourseOfFire: restore.CourseOfFire{Key: "150"},
		Entries: []restore.RestoredEntry{{
			ID: 1, Lane: 2, Category: "Pistol",
			Shooter: restore.Shooter{FirstName: "Ann", LastName: "Lee"},
			Stages:  scored,
		}},
	}}}
}

func TestRestoreRefusesNonEmptyDatabaseWithoutMerge(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	repo := repository.New(conn)
	svc := service.New(repo)
	if _, err := svc.RestoreScoreLog(ctx, mergePlan("Club Match", [7]int{1, 2, 3, 4, 0, 0, 0}, 1), false); err != nil {
		t.Fatal(err)
	}
	_, err = svc.RestoreScoreLog(ctx, mergePlan("Club Match", [7]int{1, 2, 3, 4, 0, 0, 0}, 1, 2), false)
	if !errors.Is(err, service.ErrRestoreTargetNotEmpty) {
		t.Fatalf("err = %v", err)
	}
	st, err := repo.GetStage(ctx, 7, 1, 2)
	if err != nil || st.Status != repository.StatusOpen {
		t.Fatalf("stage 2 = %+v, %v", st, err)
	}
}

func TestRestoreMergeAppliesMatchingIDsAndListsConflicts(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	repo := repository.New(conn)
	svc := service.New(repo)
	if _, err := svc.RestoreScoreLog(ctx, mergePlan("Club Match", [7]int{1, 2, 3, 4, 0, 0, 0}, 1), false); err != nil {
		t.Fatal(err)
	}

	res, err := svc.RestoreScoreLog(ctx, mergePlan("Club Match", [7]int{9, 1, 0, 0, 0, 0, 0}, 1, 2), true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Stages != 1 || len(res.Conflicts) != 1 || !strings.Contains(res.Conflicts[0], "match 7 entry 1 stage 1") {
		t.Fatalf("res = %+v", res)
	}
	kept, err := repo.GetStage(ctx, 7, 1, 1)
	if err != nil || kept.Counts.X != 1 || kept.Counts.Ten != 2 {
		t.Fatalf("stage 1 = %+v, %v", kept, err)
	}
	added, err := repo.GetStage(ctx, 7, 1, 2)
	if err != nil || added.Status != repository.StatusAccepted || added.Counts.X != 9 {
		t.Fatalf("stage 2 = %+v, %v", added, err)
	}

	res, err = svc.RestoreScoreLog(ctx, mergePlan("Other Match", [7]int{1, 2, 3, 4, 0, 0, 0}, 1, 2), true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Matches+res.Entries+res.Stages != 0 || len(res.Conflicts) != 1 || !strings.Contains(res.Conflicts[0], "match 7") {
		t.Fatalf("res = %+v", res)
	}
}

func reportOutputs(t *testing.T, repo *repository.Repository, compID int64) ([]byte, map[string][][]string) {
	t.Helper()
	ctx := context.Background()
	comp, err := repo.GetMatch(ctx, compID)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := repo.ReportEntries(ctx, compID)
	if err != nil {
		t.Fatal(err)
	}
	data := report.Data{Match: comp, Entries: entries}
	csvBytes, err := report.CSV(data)
	if err != nil {
		t.Fatal(err)
	}
	xlsx, err := report.Workbook(data)
	if err != nil {
		t.Fatal(err)
	}
	wb, err := excelize.OpenReader(bytes.NewReader(xlsx))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = wb.Close() }()
	rows := map[string][][]string{}
	for _, name := range wb.GetSheetList() {
		r, err := wb.GetRows(name)
		if err != nil {
			t.Fatal(err)
		}
		rows[name] = r
	}
	return csvBytes, rows
}

func TestRestoreRefusesDatabaseWithOnlyShooters(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	repo := repository.New(conn)
	if _, err := repo.CreateShooter(ctx, repository.Shooter{FirstName: "Bo", LastName: "Ek"}); err != nil {
		t.Fatal(err)
	}
	_, err = service.New(repo).RestoreScoreLog(ctx, mergePlan("Club Match", [7]int{1, 2, 3, 4, 0, 0, 0}, 1), false)
	if !errors.Is(err, service.ErrRestoreTargetNotEmpty) {
		t.Fatalf("err = %v", err)
	}
}

func TestRestoreWritesNothingWhenALaterMatchIsInvalid(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	repo := repository.New(conn)
	plan := mergePlan("Club Match", [7]int{1, 2, 3, 4, 0, 0, 0}, 1)
	bad := plan.Matches[0]
	bad.ID = 8
	bad.CourseOfFire.Key = "nope"
	plan.Matches = append(plan.Matches, bad)
	if _, err := service.New(repo).RestoreScoreLog(ctx, plan, false); err == nil {
		t.Fatal("expected error")
	}
	n, err := repo.CountMatches(ctx)
	if err != nil || n != 0 {
		t.Fatalf("matches = %d, %v", n, err)
	}
}

func TestRestoreMergeReportsLockMetadataConflict(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	svc := service.New(repository.New(conn))
	counts := [7]int{1, 2, 3, 4, 0, 0, 0}
	if _, err := svc.RestoreScoreLog(ctx, mergePlan("Club Match", counts, 1), false); err != nil {
		t.Fatal(err)
	}
	other := mergePlan("Club Match", counts, 1)
	other.Matches[0].Entries[0].Stages[0].ScorerName = "Someone"
	res, err := svc.RestoreScoreLog(ctx, other, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Conflicts) != 1 || !strings.Contains(res.Conflicts[0], "stage 1") {
		t.Fatalf("res = %+v", res)
	}
}

func TestRestorePartialAcceptanceLeavesMatchActive(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	repo := repository.New(conn)
	svc := service.New(repo)
	counts := [7]int{10, 0, 0, 0, 0, 0, 0}
	if _, err := svc.RestoreScoreLog(ctx, mergePlan("Partial", counts, 1), false); err != nil {
		t.Fatal(err)
	}
	comp, err := repo.GetMatch(ctx, 7)
	if err != nil || comp.Status != "active" {
		t.Fatalf("partial = %+v, %v", comp, err)
	}
}

func restoreStages(t *testing.T, stages []restore.ScoredStage) (*repository.Repository, repository.Match) {
	t.Helper()
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	repo := repository.New(conn)
	plan := mergePlan("Mid", [7]int{1, 2, 3, 4, 0, 0, 0})
	plan.Matches[0].Entries[0].Stages = stages
	if _, err := service.New(repo).RestoreScoreLog(ctx, plan, false); err != nil {
		t.Fatal(err)
	}
	comp, err := repo.GetMatch(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	return repo, comp
}

func stageWith(number int, status string) restore.ScoredStage {
	st := restore.ScoredStage{Number: number, ExpectedShots: 10, RingCounts: [7]int{1, 2, 3, 4, 0, 0, 0}, Status: status, LockedBy: "bob", ScorerName: "bob", LockedAt: "2026-10-19T10:00:00Z"}
	if status == restore.StatusAccepted {
		st.AcceptedBy, st.AcceptedAt = "bob", "2026-10-19T10:01:00Z"
	}
	return st
}

func TestRestoreSetsPointerToLowestStageNotAccepted(t *testing.T) {
	repo, comp := restoreStages(t, []restore.ScoredStage{
		stageWith(1, restore.StatusAccepted),
		stageWith(2, restore.StatusAwaitingAcceptance),
		stageWith(3, restore.StatusAccepted),
	})
	if comp.Status != "active" || comp.CurrentStage != 2 {
		t.Fatalf("comp = %+v", comp)
	}
	st, err := repo.GetStage(context.Background(), 7, 1, 2)
	if err != nil || st.Status != repository.StatusAwaitingAcceptance || st.Counts.Ten != 2 {
		t.Fatalf("stage = %+v, %v", st, err)
	}
}

func TestRestoreReopenedDisputedStageIsOpenWithHits(t *testing.T) {
	disputed := stageWith(1, restore.StatusOpen)
	disputed.DisputedAt = "2026-10-19T10:02:00Z"
	repo, comp := restoreStages(t, []restore.ScoredStage{disputed})
	if comp.Status != "active" || comp.CurrentStage != 1 {
		t.Fatalf("comp = %+v", comp)
	}
	st, err := repo.GetStage(context.Background(), 7, 1, 1)
	if err != nil || st.Status != repository.StatusOpen || st.Counts.Nine != 3 || st.DisputedAt == "" {
		t.Fatalf("stage = %+v, %v", st, err)
	}
}

func TestRestoreFullyAcceptedMatchIsFinished(t *testing.T) {
	var all []restore.ScoredStage
	course, err := rulebook.CourseByKey("150")
	if err != nil {
		t.Fatal(err)
	}
	for _, cs := range course.Stages {
		all = append(all, stageWith(cs.StageNumber, restore.StatusAccepted))
	}
	_, comp := restoreStages(t, all)
	if comp.Status != "finished" {
		t.Fatalf("comp = %+v", comp)
	}
}
