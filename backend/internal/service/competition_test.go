package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

func TestCompetitionService_Create(t *testing.T) {
	database := newTestDB(t)
	repo := repository.NewCompetitionRepository(database)
	svc := service.NewCompetitionService(repo)
	ctx := context.Background()

	tests := []struct {
		name    string
		params  service.CreateCompetitionParams
		wantErr bool
	}{
		{
			name:    "valid",
			params:  service.CreateCompetitionParams{Name: "Test Comp", Date: "2026-06-01", LanesTotal: 5},
			wantErr: false,
		},
		{
			name:    "missing name",
			params:  service.CreateCompetitionParams{Name: "", Date: "2026-06-01", LanesTotal: 5},
			wantErr: true,
		},
		{
			name:    "missing date",
			params:  service.CreateCompetitionParams{Name: "X", Date: "", LanesTotal: 5},
			wantErr: true,
		},
		{
			name:    "invalid date format",
			params:  service.CreateCompetitionParams{Name: "X", Date: "01-06-2026", LanesTotal: 5},
			wantErr: true,
		},
		{
			name:    "zero lanes",
			params:  service.CreateCompetitionParams{Name: "X", Date: "2026-06-01", LanesTotal: 0},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(ctx, tc.params)
			if (err != nil) != tc.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestCompetitionService_GetByID_NotFound(t *testing.T) {
	database := newTestDB(t)
	svc := service.NewCompetitionService(repository.NewCompetitionRepository(database))

	_, err := svc.GetByID(context.Background(), 999)
	if !errors.Is(err, service.ErrCompetitionNotFound) {
		t.Errorf("expected ErrCompetitionNotFound, got %v", err)
	}
}

func TestCompetitionService_List_InvalidStatus(t *testing.T) {
	database := newTestDB(t)
	svc := service.NewCompetitionService(repository.NewCompetitionRepository(database))

	_, err := svc.List(context.Background(), "invalid")
	if !errors.Is(err, service.ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestCompetitionService_List_ValidStatus(t *testing.T) {
	database := newTestDB(t)
	repo := repository.NewCompetitionRepository(database)
	svc := service.NewCompetitionService(repo)
	ctx := context.Background()

	_, _ = repo.Create(ctx, "A", "2026-01-01", 3, false)

	for _, status := range []string{"", "draft", "active", "finished"} {
		results, err := svc.List(ctx, status)
		if err != nil {
			t.Errorf("List(%q) unexpected error: %v", status, err)
		}
		if results == nil {
			t.Errorf("List(%q) returned nil slice", status)
		}
	}
}

func TestCompetitionService_ListVisible_ScorersOnlySeeActive(t *testing.T) {
	database := newTestDB(t)
	repo := repository.NewCompetitionRepository(database)
	svc := service.NewCompetitionService(repo)
	ctx := context.Background()

	if _, err := repo.Create(ctx, "Draft", "2026-01-01", 3, false); err != nil {
		t.Fatalf("create draft competition: %v", err)
	}
	activeComp, err := repo.Create(ctx, "Active", "2026-01-02", 3, false)
	if err != nil {
		t.Fatalf("create active competition: %v", err)
	}
	active := "active"
	if _, err := svc.Update(ctx, activeComp.ID, service.UpdateCompetitionParams{Status: &active}); err != nil {
		t.Fatalf("activate competition: %v", err)
	}

	results, err := svc.ListVisible(ctx, "draft", false)
	if err != nil {
		t.Fatalf("list visible: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 visible competition, got %d", len(results))
	}
	if results[0].Status != "active" {
		t.Fatalf("expected only active competition, got %s", results[0].Status)
	}
}

func TestCompetitionService_GetVisibleByID_HidesNonActiveFromScorers(t *testing.T) {
	database := newTestDB(t)
	repo := repository.NewCompetitionRepository(database)
	svc := service.NewCompetitionService(repo)
	ctx := context.Background()

	comp, err := repo.Create(ctx, "Draft", "2026-01-01", 3, false)
	if err != nil {
		t.Fatalf("create competition: %v", err)
	}
	if _, err := svc.GetVisibleByID(ctx, comp.ID, false); !errors.Is(err, service.ErrCompetitionNotFound) {
		t.Fatalf("expected ErrCompetitionNotFound, got %v", err)
	}
	if _, err := svc.GetVisibleByID(ctx, comp.ID, true); err != nil {
		t.Fatalf("expected admin access, got %v", err)
	}
}

func TestCompetitionService_Update_NotFound(t *testing.T) {
	database := newTestDB(t)
	svc := service.NewCompetitionService(repository.NewCompetitionRepository(database))

	_, err := svc.Update(context.Background(), 999, service.UpdateCompetitionParams{})
	if !errors.Is(err, service.ErrCompetitionNotFound) {
		t.Errorf("expected ErrCompetitionNotFound, got %v", err)
	}
}

func TestCompetitionService_Update_StatusTransition(t *testing.T) {
	database := newTestDB(t)
	repo := repository.NewCompetitionRepository(database)
	svc := service.NewCompetitionService(repo)
	ctx := context.Background()

	comp, _ := repo.Create(ctx, "A", "2026-01-01", 3, false)

	active := "active"
	finished := "finished"
	draft := "draft"

	_, err := svc.Update(ctx, comp.ID, service.UpdateCompetitionParams{Status: &active})
	if err != nil {
		t.Fatalf("transition draft→active: %v", err)
	}

	_, err = svc.Update(ctx, comp.ID, service.UpdateCompetitionParams{Status: &finished})
	if err != nil {
		t.Fatalf("transition active→finished: %v", err)
	}

	_, err = svc.Update(ctx, comp.ID, service.UpdateCompetitionParams{Status: &draft})
	if !errors.Is(err, service.ErrInvalidStatusTransition) {
		t.Errorf("expected ErrInvalidStatusTransition for finished→draft, got %v", err)
	}
}

func TestCompetitionService_Update_InvalidStatus(t *testing.T) {
	database := newTestDB(t)
	repo := repository.NewCompetitionRepository(database)
	svc := service.NewCompetitionService(repo)
	ctx := context.Background()

	comp, _ := repo.Create(ctx, "A", "2026-01-01", 3, false)
	bad := "bogus"
	_, err := svc.Update(ctx, comp.ID, service.UpdateCompetitionParams{Status: &bad})
	if !errors.Is(err, service.ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}
