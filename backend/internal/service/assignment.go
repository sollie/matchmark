package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/sollie/matchmark/internal/model"
	"github.com/sollie/matchmark/internal/repository"
)

var (
	ErrShooterAlreadyClaimed = errors.New("shooter is already claimed by another scorer")
	ErrUserAlreadyClaimed    = errors.New("you are already assigned to a shooter in this competition")
	ErrAssignmentNotFound    = errors.New("assignment not found")
)

type AssignmentService struct {
	assignments  *repository.AssignmentRepository
	competitions *repository.CompetitionRepository
	shooters     *repository.ShooterRepository
}

func NewAssignmentService(
	assignments *repository.AssignmentRepository,
	competitions *repository.CompetitionRepository,
	shooters *repository.ShooterRepository,
) *AssignmentService {
	return &AssignmentService{
		assignments:  assignments,
		competitions: competitions,
		shooters:     shooters,
	}
}

func (s *AssignmentService) SelectShooter(ctx context.Context, competitionID, shooterID, callerID int64) (*model.ScorerAssignment, error) {
	if _, err := s.competitions.GetByID(ctx, competitionID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCompetitionNotFound
		}
		return nil, fmt.Errorf("get competition: %w", err)
	}

	shooter, err := s.shooters.GetByID(ctx, shooterID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrShooterNotFound
		}
		return nil, fmt.Errorf("get shooter: %w", err)
	}
	if shooter.CompetitionID != competitionID {
		return nil, ErrShooterNotFound
	}

	existing, err := s.assignments.GetByUser(ctx, competitionID, callerID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("check user assignment: %w", err)
	}
	if existing != nil {
		return nil, ErrUserAlreadyClaimed
	}

	_, err = s.assignments.GetByShooter(ctx, competitionID, shooterID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("check shooter assignment: %w", err)
	}
	if err == nil {
		return nil, ErrShooterAlreadyClaimed
	}

	assignment, err := s.assignments.Create(ctx, competitionID, shooterID, callerID)
	if err != nil {
		return nil, fmt.Errorf("create assignment: %w", err)
	}
	return assignment, nil
}

func (s *AssignmentService) ReleaseAssignment(ctx context.Context, competitionID, shooterID int64) error {
	if _, err := s.competitions.GetByID(ctx, competitionID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrCompetitionNotFound
		}
		return fmt.Errorf("get competition: %w", err)
	}

	if err := s.assignments.DeleteByShooter(ctx, competitionID, shooterID); err != nil {
		return fmt.Errorf("delete assignment: %w", err)
	}
	return nil
}

func (s *AssignmentService) GetMyAssignment(ctx context.Context, competitionID, callerID int64) (*model.ScorerAssignment, error) {
	assignment, err := s.assignments.GetByUser(ctx, competitionID, callerID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrAssignmentNotFound
		}
		return nil, fmt.Errorf("get assignment: %w", err)
	}
	return assignment, nil
}

func (s *AssignmentService) ListAssignments(ctx context.Context, competitionID int64) ([]model.ScorerAssignment, error) {
	if _, err := s.competitions.GetByID(ctx, competitionID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCompetitionNotFound
		}
		return nil, fmt.Errorf("get competition: %w", err)
	}

	assignments, err := s.assignments.ListByCompetition(ctx, competitionID)
	if err != nil {
		return nil, fmt.Errorf("list assignments: %w", err)
	}
	return assignments, nil
}
