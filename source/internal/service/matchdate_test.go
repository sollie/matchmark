package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

func TestMatchDateMustFallWithinCompetitionRange(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	res, err := conn.Exec(`INSERT INTO competitions (name, start_date, end_date) VALUES ('Nationals', '2026-10-19', '2026-10-20')`)
	if err != nil {
		t.Fatal(err)
	}
	containerID, _ := res.LastInsertId()
	svc := service.New(repository.New(conn))
	ctx := context.Background()

	_, err = svc.CreateCompetition(ctx, repository.Competition{ContainerID: containerID, Name: "M1", Date: "2026-10-21", Course: "150"})
	if !errors.Is(err, service.ErrInvalid) {
		t.Fatalf("out-of-range create: %v", err)
	}
	id, err := svc.CreateCompetition(ctx, repository.Competition{ContainerID: containerID, Name: "M1", Date: "2026-10-20", Course: "150"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateCompetition(ctx, repository.Competition{ContainerID: containerID, Name: "M2", Date: "2026-10-19", Course: "150"}); err != nil {
		t.Fatal(err)
	}
	err = svc.UpdateCompetition(ctx, repository.Competition{ID: id, Name: "M1", Date: "2026-10-22", Course: "150"})
	if !errors.Is(err, service.ErrInvalid) {
		t.Fatalf("out-of-range update: %v", err)
	}
	if err := svc.UpdateCompetition(ctx, repository.Competition{ID: id, Name: "M1", Date: "2026-10-19", Course: "150"}); err != nil {
		t.Fatal(err)
	}
}
