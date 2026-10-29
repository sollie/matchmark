package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/repository"
)

func TestScreenCapPurgeAndMatchDeletion(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	repo := repository.New(conn)
	ctx := context.Background()
	now := time.Now()

	if err := repo.CreateScreen(ctx, "h1", "AAAAAA", now.Add(-48*time.Hour), 2); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateScreen(ctx, "h2", "BBBBBB", now, 2); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateScreen(ctx, "h3", "CCCCCC", now, 2); !errors.Is(err, repository.ErrLimit) {
		t.Fatalf("cap: %v", err)
	}
	if err := repo.CreateScreen(ctx, "h4", "BBBBBB", now, 5); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("duplicate code: %v", err)
	}

	if n, err := repo.PurgeScreensSeenBefore(ctx, now.Add(-24*time.Hour)); err != nil || n != 1 {
		t.Fatalf("purge = %d, %v", n, err)
	}
	if _, err := repo.ScreenByTokenHash(ctx, "h1"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("stale screen kept: %v", err)
	}

	matchID, err := repo.CreateMatch(ctx, repository.Match{Name: "A", Date: "2026-10-19", Course: "40"})
	if err != nil {
		t.Fatal(err)
	}
	id, err := repo.ApproveScreenByCode(ctx, "BBBBBB", "", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.AssignScreen(ctx, id, &matchID); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`DELETE FROM matches WHERE id = ?`, matchID); err != nil {
		t.Fatal(err)
	}
	if s, err := repo.GetScreen(ctx, id); err != nil || s.MatchID != nil {
		t.Fatalf("match_id after delete = %v, %v", s.MatchID, err)
	}
}
