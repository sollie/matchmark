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
