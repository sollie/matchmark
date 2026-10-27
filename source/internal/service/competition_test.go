package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

func TestCompetitionDateRules(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	svc := service.New(repository.New(conn))
	repo := repository.New(conn)
	ctx := context.Background()

	if _, err := svc.CreateCompetition(ctx, repository.Competition{Name: "Bad", StartDate: "2026-10-20", EndDate: "2026-10-19"}); !errors.Is(err, service.ErrInvalid) {
		t.Fatalf("reversed range: %v", err)
	}
	id, err := svc.CreateCompetition(ctx, repository.Competition{Name: "Nationals", StartDate: "2026-10-19", EndDate: "2026-10-21"})
	if err != nil {
		t.Fatal(err)
	}
	m := func(date string) repository.Match {
		return repository.Match{CompetitionID: id, Name: "M", Date: date, Course: "150"}
	}
	if _, err := svc.CreateMatch(ctx, m("2026-10-22")); !errors.Is(err, service.ErrInvalid) {
		t.Fatalf("match after range: %v", err)
	}
	if _, err := svc.CreateMatch(ctx, m("2026-10-19")); err != nil {
		t.Fatal(err)
	}
	last, err := svc.CreateMatch(ctx, m("2026-10-21"))
	if err != nil {
		t.Fatal(err)
	}

	narrow := repository.Competition{ID: id, Name: "Nationals", StartDate: "2026-10-19", EndDate: "2026-10-20"}
	if err := svc.UpdateCompetition(ctx, narrow); !errors.Is(err, service.ErrInvalid) {
		t.Fatalf("range excluding a match: %v", err)
	}
	wide := repository.Competition{ID: id, Name: "Nationals", StartDate: "2026-10-18", EndDate: "2026-10-25"}
	if err := svc.UpdateCompetition(ctx, wide); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteCompetition(ctx, id); !errors.Is(err, service.ErrConflict) {
		t.Fatalf("delete with matches: %v", err)
	}
	if err := svc.DeleteMatch(ctx, last); err != nil {
		t.Fatal(err)
	}
	c, err := repo.GetCompetition(ctx, id)
	if err != nil || c.MatchCount != 1 {
		t.Fatalf("competition %+v, %v", c, err)
	}
}

func TestEmptyCompetitionCanBeDeleted(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	svc := service.New(repository.New(conn))
	ctx := context.Background()
	id, err := svc.CreateCompetition(ctx, repository.Competition{Name: "Empty", StartDate: "2026-10-19", EndDate: "2026-10-19"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteCompetition(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteCompetition(ctx, id); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("second delete: %v", err)
	}
}

func TestCreateMatchInNewCompetition(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	svc := service.New(repository.New(conn))
	repo := repository.New(conn)
	ctx := context.Background()
	comp := repository.Competition{Name: "Open", StartDate: "2026-10-19", EndDate: "2026-10-20"}

	if _, err := svc.CreateMatchInNewCompetition(ctx, comp, repository.Match{Name: "M", Date: "2026-10-21", Course: "150"}); !errors.Is(err, service.ErrInvalid) {
		t.Fatalf("date outside new range: %v", err)
	}
	cs, _ := repo.ListCompetitions(ctx)
	if len(cs) != 0 {
		t.Fatalf("rejected create left %d competitions", len(cs))
	}
	id, err := svc.CreateMatchInNewCompetition(ctx, comp, repository.Match{Name: "M", Date: "2026-10-20", Course: "150"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetMatch(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	c, err := repo.GetCompetition(ctx, got.CompetitionID)
	if err != nil || c.Name != "Open" || c.MatchCount != 1 {
		t.Fatalf("competition %+v, %v", c, err)
	}
}
