package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

func newShooterSvc(t *testing.T, lanesTotal int) (*service.ShooterService, int64) {
	t.Helper()
	database := newTestDB(t)
	comp := seedCompetition(t, database, lanesTotal)
	svc := service.NewShooterService(
		repository.NewShooterRepository(database),
		repository.NewStageRepository(database),
		repository.NewCompetitionRepository(database),
	)
	return svc, comp.ID
}

func TestShooterService_ListByCompetition_CompNotFound(t *testing.T) {
	database := newTestDB(t)
	svc := service.NewShooterService(
		repository.NewShooterRepository(database),
		repository.NewStageRepository(database),
		repository.NewCompetitionRepository(database),
	)

	_, err := svc.ListByCompetition(context.Background(), 999)
	if !errors.Is(err, service.ErrCompetitionNotFound) {
		t.Errorf("expected ErrCompetitionNotFound, got %v", err)
	}
}

func TestShooterService_ListByCompetition_Success(t *testing.T) {
	database := newTestDB(t)
	comp := seedCompetition(t, database, 3)
	seedShooterWithStages(t, database, comp.ID, 1)

	svc := service.NewShooterService(
		repository.NewShooterRepository(database),
		repository.NewStageRepository(database),
		repository.NewCompetitionRepository(database),
	)

	shooters, err := svc.ListByCompetition(context.Background(), comp.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(shooters) != 1 {
		t.Errorf("expected 1 shooter, got %d", len(shooters))
	}
}

func TestShooterService_Create_CompNotFound(t *testing.T) {
	database := newTestDB(t)
	svc := service.NewShooterService(
		repository.NewShooterRepository(database),
		repository.NewStageRepository(database),
		repository.NewCompetitionRepository(database),
	)

	_, err := svc.Create(context.Background(), service.CreateShooterParams{
		CompetitionID: 999,
		Name:          "Alice",
		WAID:          "WA001",
		LaneNumber:    1,
		Discipline:    "1500",
		WeaponClass:   "Revolver 1500",
	})
	if !errors.Is(err, service.ErrCompetitionNotFound) {
		t.Errorf("expected ErrCompetitionNotFound, got %v", err)
	}
}

func TestShooterService_Create_LaneOutOfRange(t *testing.T) {
	svc, compID := newShooterSvc(t, 3)

	_, err := svc.Create(context.Background(), service.CreateShooterParams{
		CompetitionID: compID,
		Name:          "Alice",
		WAID:          "WA001",
		LaneNumber:    10,
		Discipline:    "1500",
		WeaponClass:   "Revolver 1500",
	})
	if !errors.Is(err, service.ErrLaneOutOfRange) {
		t.Errorf("expected ErrLaneOutOfRange, got %v", err)
	}
}

func TestShooterService_Create_InvalidWeaponClass(t *testing.T) {
	svc, compID := newShooterSvc(t, 3)

	_, err := svc.Create(context.Background(), service.CreateShooterParams{
		CompetitionID: compID,
		Name:          "Alice",
		WAID:          "WA001",
		LaneNumber:    2,
		Discipline:    "1500",
		WeaponClass:   "BogusClass",
	})
	if !errors.Is(err, service.ErrInvalidWeaponClass) {
		t.Errorf("expected ErrInvalidWeaponClass, got %v", err)
	}
}

func TestShooterService_Create_LaneTaken(t *testing.T) {
	database := newTestDB(t)
	comp := seedCompetition(t, database, 3)
	seedShooterWithStages(t, database, comp.ID, 1)

	svc := service.NewShooterService(
		repository.NewShooterRepository(database),
		repository.NewStageRepository(database),
		repository.NewCompetitionRepository(database),
	)

	_, err := svc.Create(context.Background(), service.CreateShooterParams{
		CompetitionID: comp.ID,
		Name:          "Bob",
		WAID:          "WA002",
		LaneNumber:    1,
		Discipline:    "1500",
		WeaponClass:   "Revolver 1500",
	})
	if !errors.Is(err, service.ErrLaneTaken) {
		t.Errorf("expected ErrLaneTaken, got %v", err)
	}
}

func TestShooterService_Create_Success(t *testing.T) {
	svc, compID := newShooterSvc(t, 3)

	shooter, err := svc.Create(context.Background(), service.CreateShooterParams{
		CompetitionID: compID,
		Name:          "Alice",
		WAID:          "WA001",
		LaneNumber:    2,
		Discipline:    "1500",
		WeaponClass:   "Revolver 1500",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shooter.Name != "Alice" {
		t.Errorf("expected name Alice, got %s", shooter.Name)
	}
}

func TestShooterService_Delete_CompNotFound(t *testing.T) {
	database := newTestDB(t)
	svc := service.NewShooterService(
		repository.NewShooterRepository(database),
		repository.NewStageRepository(database),
		repository.NewCompetitionRepository(database),
	)

	err := svc.Delete(context.Background(), 999, 1)
	if !errors.Is(err, service.ErrCompetitionNotFound) {
		t.Errorf("expected ErrCompetitionNotFound, got %v", err)
	}
}

func TestShooterService_Delete_ShooterNotFound(t *testing.T) {
	svc, compID := newShooterSvc(t, 3)

	err := svc.Delete(context.Background(), compID, 999)
	if !errors.Is(err, service.ErrShooterNotFound) {
		t.Errorf("expected ErrShooterNotFound, got %v", err)
	}
}

func TestShooterService_Delete_Success(t *testing.T) {
	database := newTestDB(t)
	comp := seedCompetition(t, database, 3)
	shooter := seedShooterWithStages(t, database, comp.ID, 1)

	svc := service.NewShooterService(
		repository.NewShooterRepository(database),
		repository.NewStageRepository(database),
		repository.NewCompetitionRepository(database),
	)

	if err := svc.Delete(context.Background(), comp.ID, shooter.ID); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
