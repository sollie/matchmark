package service_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
	"github.com/xuri/excelize/v2"
)

func TestCompetitionDownloadNamesAndSkips(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	svc := service.New(repository.New(conn))
	ctx := context.Background()

	comp, err := svc.CreateCompetition(ctx, repository.Competition{Name: "Nationals", StartDate: "2026-10-19", EndDate: "2026-10-19"})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		id, err := svc.CreateMatch(ctx, repository.Match{CompetitionID: comp, Name: "Same", Date: "2026-10-19", Course: "150"})
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			continue
		}
		for _, s := range []service.MatchStatus{service.StatusActive, service.StatusFinished} {
			if err := svc.SetMatchStatus(ctx, id, s); err != nil {
				t.Fatal(err)
			}
		}
	}

	if _, err := svc.CompetitionDownload(ctx, comp, "tar"); !errors.Is(err, service.ErrInvalid) {
		t.Fatalf("bad format: %v", err)
	}
	file, err := svc.CompetitionDownload(ctx, comp, service.FormatCSV)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(file.Data), int64(len(file.Data)))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	if got := strings.Join(names, "|"); got != "SKIPPED.txt|Same-2026-10-19-150.csv" {
		t.Fatalf("files %s", got)
	}
}

func TestCompetitionWorkbook1500SelectionAndRepeatShooters(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	svc := service.New(repository.New(conn))
	ctx := context.Background()

	comp, err := svc.CreateCompetition(ctx, repository.Competition{Name: "Nationals", StartDate: "2026-10-19", EndDate: "2026-10-20"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CompetitionWorkbook1500(ctx, 99999); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("unknown competition: %v", err)
	}
	if _, err := svc.CompetitionWorkbook1500(ctx, comp); !errors.Is(err, service.ErrNothingToReport) {
		t.Fatalf("empty competition: %v", err)
	}

	shooter, err := svc.CreateShooter(ctx, repository.Shooter{LastName: "BERG", FirstName: "Ola", Country: "NOR", Club: "Oslo PPC", WAID: "W100"})
	if err != nil {
		t.Fatal(err)
	}
	add := func(name, date, course string, finish bool) {
		id, err := svc.CreateMatch(ctx, repository.Match{CompetitionID: comp, Name: name, Date: date, Course: course})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := svc.AddEntry(ctx, repository.Entry{MatchID: id, ShooterID: shooter, Category: "Revolver", Lane: 1}); err != nil {
			t.Fatal(err)
		}
		if !finish {
			return
		}
		for _, s := range []service.MatchStatus{service.StatusActive, service.StatusFinished} {
			if err := svc.SetMatchStatus(ctx, id, s); err != nil {
				t.Fatal(err)
			}
		}
	}
	add("Unfinished", "2026-10-19", "150", false)
	add("Short", "2026-10-19", "40", true)
	if _, err := svc.CompetitionWorkbook1500(ctx, comp); !errors.Is(err, service.ErrNothingToReport) {
		t.Fatalf("no finished 1500 match: %v", err)
	}

	add("Day one", "2026-10-19", "150", true)
	add("Day two", "2026-10-20", "150", true)
	file, err := svc.CompetitionWorkbook1500(ctx, comp)
	if err != nil {
		t.Fatal(err)
	}
	if file.BaseName != "Nationals-2026-10-19-1500" {
		t.Fatalf("base name %q", file.BaseName)
	}
	f, err := excelize.OpenReader(bytes.NewReader(file.Data))
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"A3", "A4"} {
		if v, _ := f.GetCellValue("Revolver", ref); v != "W100" {
			t.Fatalf("%s = %q, want one row per match", ref, v)
		}
	}
	if v, _ := f.GetCellValue("Revolver", "A5"); v != "" {
		t.Fatalf("A5 = %q, want only two rows", v)
	}
}
