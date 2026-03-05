package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/sollie/matchmark/internal/auth"
	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/model"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.Migrate(database); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return database
}

func seedCompetition(t *testing.T, database *sql.DB, lanesTotal int) *model.Competition {
	t.Helper()
	repo := repository.NewCompetitionRepository(database)
	comp, err := repo.Create(context.Background(), "Test", "2026-01-01", lanesTotal, false)
	if err != nil {
		t.Fatalf("create competition: %v", err)
	}
	return comp
}

func seedShooterWithStages(t *testing.T, database *sql.DB, competitionID int64, lane int) *model.Shooter {
	t.Helper()
	shooterRepo := repository.NewShooterRepository(database)
	stageRepo := repository.NewStageRepository(database)

	shooter, err := shooterRepo.Create(context.Background(), competitionID, "Alice", "WA001", nil, lane, "1500", "Revolver 1500")
	if err != nil {
		t.Fatalf("create shooter lane %d: %v", lane, err)
	}

	stages := []model.Stage{
		{StageNumber: 1, ExpectedShots: 12},
		{StageNumber: 2, ExpectedShots: 12},
	}
	if err := stageRepo.CreateBulk(context.Background(), shooter.ID, stages); err != nil {
		t.Fatalf("create stages: %v", err)
	}
	return shooter
}

func seedUser(t *testing.T, database *sql.DB) int64 {
	t.Helper()
	userRepo := repository.NewUserRepository(database)
	passwordHash, err := auth.HashPassword("supersecretpassword1234")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user, err := userRepo.Create(context.Background(), "scorer1", passwordHash, "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user.ID
}

func newScoringService(database *sql.DB) *service.ScoringService {
	return service.NewScoringService(
		repository.NewCompetitionRepository(database),
		repository.NewShooterRepository(database),
		repository.NewStageRepository(database),
		repository.NewHitRepository(database),
		repository.NewLockRepository(database),
		repository.NewAssignmentRepository(database),
	)
}

func TestGetScoreLane(t *testing.T) {
	tests := []struct {
		ownLane    int
		lanesTotal int
		want       int
	}{
		{1, 5, 5},
		{2, 5, 1},
		{3, 5, 2},
		{5, 5, 4},
		{1, 1, 1},
	}
	for _, tc := range tests {
		got := service.GetScoreLane(tc.ownLane, tc.lanesTotal)
		if got != tc.want {
			t.Errorf("GetScoreLane(%d, %d) = %d, want %d", tc.ownLane, tc.lanesTotal, got, tc.want)
		}
	}
}

func TestResolveStage_CompNotFound(t *testing.T) {
	database := newTestDB(t)
	svc := newScoringService(database)

	_, err := svc.GetStageDetail(context.Background(), 999, 1, 1, 1, true)
	if !errors.Is(err, service.ErrCompetitionNotFound) {
		t.Errorf("expected ErrCompetitionNotFound, got %v", err)
	}
}

func TestResolveStage_NoShooterOnLane(t *testing.T) {
	database := newTestDB(t)
	comp := seedCompetition(t, database, 3)
	svc := newScoringService(database)

	_, err := svc.GetStageDetail(context.Background(), comp.ID, 2, 1, 1, true)
	if !errors.Is(err, service.ErrStageLaneNotFound) {
		t.Errorf("expected ErrStageLaneNotFound, got %v", err)
	}
}

func TestResolveStage_StageNotFound(t *testing.T) {
	database := newTestDB(t)
	comp := seedCompetition(t, database, 3)
	seedShooterWithStages(t, database, comp.ID, 1)
	svc := newScoringService(database)

	_, err := svc.GetStageDetail(context.Background(), comp.ID, 1, 99, 1, true)
	if !errors.Is(err, service.ErrShooterNotFound) {
		t.Errorf("expected ErrShooterNotFound, got %v", err)
	}
}

func TestResolveStage_Success(t *testing.T) {
	database := newTestDB(t)
	comp := seedCompetition(t, database, 3)
	seedShooterWithStages(t, database, comp.ID, 1)
	svc := newScoringService(database)

	detail, err := svc.GetStageDetail(context.Background(), comp.ID, 1, 1, 1, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.StageNumber != 1 {
		t.Errorf("expected stage 1, got %d", detail.StageNumber)
	}
}

func TestCheckScorerAuthorization_SingleLane(t *testing.T) {
	database := newTestDB(t)
	comp := seedCompetition(t, database, 1)
	shooter := seedShooterWithStages(t, database, comp.ID, 1)
	userID := seedUser(t, database)

	assignRepo := repository.NewAssignmentRepository(database)
	_, err := assignRepo.Create(context.Background(), comp.ID, shooter.ID, userID)
	if err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	svc := newScoringService(database)
	_, err = svc.SubmitHits(context.Background(), comp.ID, 1, 1, make([]string, 12), userID, false)
	if !errors.Is(err, service.ErrSingleLaneAdminOnly) {
		t.Errorf("expected ErrSingleLaneAdminOnly, got %v", err)
	}
}

func TestCheckScorerAuthorization_NotAssigned(t *testing.T) {
	database := newTestDB(t)
	comp := seedCompetition(t, database, 3)
	seedShooterWithStages(t, database, comp.ID, 2)
	userID := seedUser(t, database)
	svc := newScoringService(database)

	_, err := svc.SubmitHits(context.Background(), comp.ID, 2, 1, make([]string, 12), userID, false)
	if !errors.Is(err, service.ErrNotAssignedToLane) {
		t.Errorf("expected ErrNotAssignedToLane, got %v", err)
	}
}

func TestCheckScorerAuthorization_WrongLane(t *testing.T) {
	database := newTestDB(t)
	comp := seedCompetition(t, database, 3)
	seedShooterWithStages(t, database, comp.ID, 1)
	seedShooterWithStages(t, database, comp.ID, 2)
	shooter3 := seedShooterWithStages(t, database, comp.ID, 3)
	userID := seedUser(t, database)

	assignRepo := repository.NewAssignmentRepository(database)
	_, err := assignRepo.Create(context.Background(), comp.ID, shooter3.ID, userID)
	if err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	svc := newScoringService(database)
	_, err = svc.SubmitHits(context.Background(), comp.ID, 1, 1, make([]string, 12), userID, false)
	if !errors.Is(err, service.ErrNotAssignedToLane) {
		t.Errorf("expected ErrNotAssignedToLane, got %v", err)
	}
}

func TestCheckScorerAuthorization_Success(t *testing.T) {
	database := newTestDB(t)
	comp := seedCompetition(t, database, 3)
	seedShooterWithStages(t, database, comp.ID, 1)
	shooter2 := seedShooterWithStages(t, database, comp.ID, 2)
	userID := seedUser(t, database)

	assignRepo := repository.NewAssignmentRepository(database)
	_, err := assignRepo.Create(context.Background(), comp.ID, shooter2.ID, userID)
	if err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	svc := newScoringService(database)
	rings := make([]string, 12)
	for i := range rings {
		rings[i] = "10"
	}
	_, err = svc.SubmitHits(context.Background(), comp.ID, 1, 1, rings, userID, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
