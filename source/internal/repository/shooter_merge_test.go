package repository_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/repository"
)

func TestMergeShootersRechecksBlockersInsideTheTransaction(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "mm.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	repo := repository.New(conn)
	ctx := context.Background()

	shooter := func(last, waid string) int64 {
		id, err := repo.CreateShooter(ctx, repository.Shooter{LastName: last, FirstName: "Ann", Country: "NOR", Club: "C", WAID: waid})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	source, target, other := shooter("SOURCE", ""), shooter("TARGET", ""), shooter("OTHER", "W1")
	matchID, err := repo.CreateMatch(ctx, repository.Match{Name: "M", Date: "2026-05-01", Course: "150"})
	if err != nil {
		t.Fatal(err)
	}
	for i, id := range []int64{source, target} {
		if _, err := repo.CreateEntry(ctx, repository.Entry{MatchID: matchID, ShooterID: id, Category: "Revolver", Lane: i + 1}, nil); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := repo.MergeShooters(ctx, source, target); !errors.Is(err, repository.ErrMergeBlocked) {
		t.Fatalf("shared match: %v", err)
	}
	if _, err := repo.GetShooter(ctx, source); err != nil {
		t.Fatalf("source removed by blocked merge: %v", err)
	}

	if err := repo.SetShooterArchived(ctx, other, true); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.MergeShooters(ctx, source, other); !errors.Is(err, repository.ErrMergeArchived) {
		t.Fatalf("archived target: %v", err)
	}
}
