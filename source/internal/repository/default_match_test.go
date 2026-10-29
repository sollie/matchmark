package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/repository"
)

func TestActivatedAtAndDefaultMatch(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	repo := repository.New(conn)
	ctx := context.Background()
	create := func(name string) int64 {
		id, err := repo.CreateMatch(ctx, repository.Match{Name: name, Date: "2026-10-19", Course: "40"})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	first, second := create("A"), create("B")

	if _, err := repo.DefaultMatchID(ctx); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("default with no active matches: %v", err)
	}
	if m, _ := repo.GetMatch(ctx, first); m.ActivatedAt != "" {
		t.Fatalf("draft activated_at = %q", m.ActivatedAt)
	}

	if err := repo.SetMatchStatus(ctx, first, "draft", "active"); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetMatchStatus(ctx, second, "draft", "active"); err != nil {
		t.Fatal(err)
	}
	if m, _ := repo.GetMatch(ctx, first); m.ActivatedAt == "" {
		t.Fatal("activated_at not set on activation")
	}
	if id, err := repo.DefaultMatchID(ctx); err != nil || id != second {
		t.Fatalf("default = %d, %v; want %d", id, err, second)
	}

	if err := repo.SetMatchStatus(ctx, second, "active", "finished"); err != nil {
		t.Fatal(err)
	}
	if m, _ := repo.GetMatch(ctx, second); m.ActivatedAt == "" {
		t.Fatal("activated_at cleared on finish")
	}
	if id, err := repo.DefaultMatchID(ctx); err != nil || id != first {
		t.Fatalf("default after finish = %d, %v; want %d", id, err, first)
	}
}
