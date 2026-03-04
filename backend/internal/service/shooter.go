package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/sollie/matchmark/internal/model"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
)

var (
	ErrShooterNotFound    = errors.New("shooter not found")
	ErrLaneOutOfRange     = errors.New("lane number out of range for this competition")
	ErrLaneTaken          = errors.New("lane number already assigned in this competition")
	ErrInvalidWeaponClass = errors.New("invalid weapon class for discipline")
)

type ShooterService struct {
	shooters     *repository.ShooterRepository
	stages       *repository.StageRepository
	competitions *repository.CompetitionRepository
}

func NewShooterService(
	shooters *repository.ShooterRepository,
	stages *repository.StageRepository,
	competitions *repository.CompetitionRepository,
) *ShooterService {
	return &ShooterService{
		shooters:     shooters,
		stages:       stages,
		competitions: competitions,
	}
}

func (s *ShooterService) ListByCompetition(ctx context.Context, competitionID int64) ([]model.Shooter, error) {
	if _, err := s.competitions.GetByID(ctx, competitionID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCompetitionNotFound
		}
		return nil, fmt.Errorf("get competition: %w", err)
	}
	return s.shooters.ListByCompetition(ctx, competitionID)
}

type CreateShooterParams struct {
	CompetitionID int64
	Name          string
	WAID          string
	Club          *string
	LaneNumber    int
	Discipline    string
	WeaponClass   string
}

func (s *ShooterService) Create(ctx context.Context, p CreateShooterParams) (*model.Shooter, error) {
	comp, err := s.competitions.GetByID(ctx, p.CompetitionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCompetitionNotFound
		}
		return nil, fmt.Errorf("get competition: %w", err)
	}

	if p.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(p.Name) > 100 {
		return nil, fmt.Errorf("name must not exceed 100 characters")
	}
	if p.WAID == "" {
		return nil, fmt.Errorf("wa_id is required")
	}
	if len(p.WAID) > 20 {
		return nil, fmt.Errorf("wa_id must not exceed 20 characters")
	}
	if p.Club != nil && len(*p.Club) > 100 {
		return nil, fmt.Errorf("club must not exceed 100 characters")
	}

	if p.LaneNumber < 1 || p.LaneNumber > comp.LanesTotal {
		return nil, ErrLaneOutOfRange
	}

	if err := rulebook.ValidateWeaponClass(p.Discipline, p.WeaponClass); err != nil {
		return nil, ErrInvalidWeaponClass
	}

	shooter, err := s.shooters.Create(ctx, p.CompetitionID, p.Name, p.WAID, p.Club, p.LaneNumber, p.Discipline, p.WeaponClass)
	if err != nil {
		if repository.IsUniqueConstraintErr(err) {
			return nil, ErrLaneTaken
		}
		return nil, fmt.Errorf("create shooter: %w", err)
	}

	course, err := rulebook.CourseOfFire(p.Discipline, p.WeaponClass)
	if err != nil {
		return nil, fmt.Errorf("course of fire: %w", err)
	}

	stageModels := make([]model.Stage, len(course.Stages))
	for i, rs := range course.Stages {
		stageModels[i] = model.Stage{
			ShooterID:     shooter.ID,
			StageNumber:   rs.StageNumber,
			ExpectedShots: rs.ExpectedShots,
		}
	}
	if err := s.stages.CreateBulk(ctx, shooter.ID, stageModels); err != nil {
		return nil, fmt.Errorf("create stages: %w", err)
	}

	return shooter, nil
}

type UpdateShooterParams struct {
	Name        *string
	WAID        *string
	Club        *string
	LaneNumber  *int
	Discipline  *string
	WeaponClass *string
}

func (s *ShooterService) Update(ctx context.Context, competitionID, shooterID int64, p UpdateShooterParams) (*model.Shooter, error) {
	comp, err := s.competitions.GetByID(ctx, competitionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCompetitionNotFound
		}
		return nil, fmt.Errorf("get competition: %w", err)
	}

	current, err := s.shooters.GetByID(ctx, shooterID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrShooterNotFound
		}
		return nil, fmt.Errorf("get shooter: %w", err)
	}

	name := current.Name
	waID := current.WAID
	club := current.Club
	lane := current.LaneNumber
	discipline := current.Discipline
	weaponClass := current.WeaponClass

	if p.Name != nil {
		if *p.Name == "" {
			return nil, fmt.Errorf("name must not be empty")
		}
		if len(*p.Name) > 100 {
			return nil, fmt.Errorf("name must not exceed 100 characters")
		}
		name = *p.Name
	}
	if p.WAID != nil {
		if *p.WAID == "" {
			return nil, fmt.Errorf("wa_id must not be empty")
		}
		if len(*p.WAID) > 20 {
			return nil, fmt.Errorf("wa_id must not exceed 20 characters")
		}
		waID = *p.WAID
	}
	if p.Club != nil {
		if len(*p.Club) > 100 {
			return nil, fmt.Errorf("club must not exceed 100 characters")
		}
		club = p.Club
	}
	if p.LaneNumber != nil {
		if *p.LaneNumber < 1 || *p.LaneNumber > comp.LanesTotal {
			return nil, ErrLaneOutOfRange
		}
		lane = *p.LaneNumber
	}
	if p.Discipline != nil {
		discipline = *p.Discipline
	}
	if p.WeaponClass != nil {
		weaponClass = *p.WeaponClass
	}

	if err := rulebook.ValidateWeaponClass(discipline, weaponClass); err != nil {
		return nil, ErrInvalidWeaponClass
	}

	updated, err := s.shooters.Update(ctx, shooterID, name, waID, club, lane, discipline, weaponClass)
	if err != nil {
		if repository.IsUniqueConstraintErr(err) {
			return nil, ErrLaneTaken
		}
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrShooterNotFound
		}
		return nil, fmt.Errorf("update shooter: %w", err)
	}
	return updated, nil
}

func (s *ShooterService) Delete(ctx context.Context, competitionID, shooterID int64) error {
	if _, err := s.competitions.GetByID(ctx, competitionID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrCompetitionNotFound
		}
		return fmt.Errorf("get competition: %w", err)
	}
	if err := s.shooters.Delete(ctx, shooterID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrShooterNotFound
		}
		return fmt.Errorf("delete shooter: %w", err)
	}
	return nil
}
