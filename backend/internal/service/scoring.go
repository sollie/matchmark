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
	ErrStageLocked       = errors.New("stage is locked")
	ErrStageLaneNotFound = errors.New("no shooter on that lane")
	ErrWrongShotCount    = errors.New("hit count does not match expected shots")
	ErrInvalidRing       = errors.New("invalid ring value")
	ErrNotAssignedToLane = errors.New("you are not assigned to score this lane")
)

type ScoringService struct {
	competitions *repository.CompetitionRepository
	shooters     *repository.ShooterRepository
	stages       *repository.StageRepository
	hits         *repository.HitRepository
	locks        *repository.LockRepository
	assignments  *repository.AssignmentRepository
}

func NewScoringService(
	competitions *repository.CompetitionRepository,
	shooters *repository.ShooterRepository,
	stages *repository.StageRepository,
	hits *repository.HitRepository,
	locks *repository.LockRepository,
	assignments *repository.AssignmentRepository,
) *ScoringService {
	return &ScoringService{
		competitions: competitions,
		shooters:     shooters,
		stages:       stages,
		hits:         hits,
		locks:        locks,
		assignments:  assignments,
	}
}

func GetScoreLane(ownLane, lanesTotal int) int {
	if ownLane == 1 {
		return lanesTotal
	}
	return ownLane - 1
}

type StageDetail struct {
	StageNumber      int
	ExpectedShots    int
	Distance         string
	TimeSeconds      int
	Position         string
	Discipline       string
	TotalStages      int
	Shooter          model.Shooter
	Hits             []model.Hit
	LockedByScorer   bool
	FullyLocked      bool
	StageScore       int
	StageXCount      int
	CumulativeScore  int
	CumulativeXCount int
}

func (s *ScoringService) GetStageDetail(ctx context.Context, competitionID int64, lane, stageNumber int) (*StageDetail, error) {
	if _, err := s.competitions.GetByID(ctx, competitionID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCompetitionNotFound
		}
		return nil, fmt.Errorf("get competition: %w", err)
	}

	targetShooter, err := s.shooters.GetByLane(ctx, competitionID, lane)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrStageLaneNotFound
		}
		return nil, fmt.Errorf("get shooter by lane: %w", err)
	}

	stage, err := s.stages.GetByShooterAndNumber(ctx, targetShooter.ID, stageNumber)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrShooterNotFound
		}
		return nil, fmt.Errorf("get stage: %w", err)
	}

	course, err := rulebook.CourseOfFire(targetShooter.Discipline, targetShooter.WeaponClass)
	if err != nil {
		return nil, fmt.Errorf("course of fire: %w", err)
	}

	var rbStage rulebook.Stage
	for _, rs := range course.Stages {
		if rs.StageNumber == stageNumber {
			rbStage = rs
			break
		}
	}

	stageHits, err := s.hits.ListByStage(ctx, stage.ID)
	if err != nil {
		return nil, fmt.Errorf("list hits: %w", err)
	}

	lockedByScorer, err := s.locks.IsLockedByScorer(ctx, stage.ID)
	if err != nil {
		return nil, fmt.Errorf("check lock: %w", err)
	}

	stageScore, stageX, err := s.hits.SumByStage(ctx, stage.ID)
	if err != nil {
		return nil, fmt.Errorf("sum stage: %w", err)
	}

	cumScore, cumX, err := s.hits.SumByShooter(ctx, targetShooter.ID)
	if err != nil {
		return nil, fmt.Errorf("sum shooter: %w", err)
	}

	return &StageDetail{
		StageNumber:      stageNumber,
		ExpectedShots:    stage.ExpectedShots,
		Distance:         rbStage.Distance,
		TimeSeconds:      rbStage.TimeSeconds,
		Position:         rbStage.Position,
		Discipline:       targetShooter.Discipline,
		TotalStages:      len(course.Stages),
		Shooter:          *targetShooter,
		Hits:             stageHits,
		LockedByScorer:   lockedByScorer,
		FullyLocked:      stage.FullyLocked,
		StageScore:       stageScore,
		StageXCount:      stageX,
		CumulativeScore:  cumScore,
		CumulativeXCount: cumX,
	}, nil
}

type SubmitHitsResult struct {
	Hits       []model.Hit
	StageScore int
	XCount     int
}

func (s *ScoringService) SubmitHits(ctx context.Context, competitionID int64, lane, stageNumber int, rings []string, callerID int64, isAdmin bool) (*SubmitHitsResult, error) {
	if _, err := s.competitions.GetByID(ctx, competitionID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCompetitionNotFound
		}
		return nil, fmt.Errorf("get competition: %w", err)
	}

	targetShooter, err := s.shooters.GetByLane(ctx, competitionID, lane)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrStageLaneNotFound
		}
		return nil, fmt.Errorf("get shooter: %w", err)
	}

	stage, err := s.stages.GetByShooterAndNumber(ctx, targetShooter.ID, stageNumber)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrShooterNotFound
		}
		return nil, fmt.Errorf("get stage: %w", err)
	}

	if !isAdmin {
		assignment, err := s.assignments.GetByUser(ctx, competitionID, callerID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, ErrNotAssignedToLane
			}
			return nil, fmt.Errorf("get assignment: %w", err)
		}
		if assignment.LaneNumber != lane {
			return nil, ErrNotAssignedToLane
		}

		if stage.FullyLocked {
			return nil, ErrStageLocked
		}
		locked, err := s.locks.IsLockedByScorer(ctx, stage.ID)
		if err != nil {
			return nil, fmt.Errorf("check lock: %w", err)
		}
		if locked {
			return nil, ErrStageLocked
		}
	}

	if len(rings) != stage.ExpectedShots {
		return nil, ErrWrongShotCount
	}

	hitModels := make([]model.Hit, len(rings))
	for i, ring := range rings {
		pts, err := rulebook.RingPoints(ring)
		if err != nil {
			return nil, ErrInvalidRing
		}
		hitModels[i] = model.Hit{StageID: stage.ID, Ring: ring, Points: pts}
	}

	if err := s.hits.ReplaceForStage(ctx, stage.ID, hitModels); err != nil {
		return nil, fmt.Errorf("replace hits: %w", err)
	}

	stageScore, xCount, err := s.hits.SumByStage(ctx, stage.ID)
	if err != nil {
		return nil, fmt.Errorf("sum stage: %w", err)
	}

	stored, err := s.hits.ListByStage(ctx, stage.ID)
	if err != nil {
		return nil, fmt.Errorf("list hits: %w", err)
	}

	return &SubmitHitsResult{Hits: stored, StageScore: stageScore, XCount: xCount}, nil
}

func (s *ScoringService) LockStage(ctx context.Context, competitionID int64, lane, stageNumber int, callerID int64, isAdmin bool) error {
	if _, err := s.competitions.GetByID(ctx, competitionID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrCompetitionNotFound
		}
		return fmt.Errorf("get competition: %w", err)
	}

	targetShooter, err := s.shooters.GetByLane(ctx, competitionID, lane)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrStageLaneNotFound
		}
		return fmt.Errorf("get shooter: %w", err)
	}

	stage, err := s.stages.GetByShooterAndNumber(ctx, targetShooter.ID, stageNumber)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrShooterNotFound
		}
		return fmt.Errorf("get stage: %w", err)
	}

	if !isAdmin {
		assignment, err := s.assignments.GetByUser(ctx, competitionID, callerID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrNotAssignedToLane
			}
			return fmt.Errorf("get assignment: %w", err)
		}
		if assignment.LaneNumber != lane {
			return ErrNotAssignedToLane
		}
	}

	if err := s.locks.Upsert(ctx, stage.ID, callerID, true); err != nil {
		return fmt.Errorf("upsert lock: %w", err)
	}

	if err := s.locks.SetFullyLocked(ctx, stage.ID, true); err != nil {
		return fmt.Errorf("set fully locked: %w", err)
	}

	return nil
}

func (s *ScoringService) UnlockStage(ctx context.Context, competitionID int64, lane, stageNumber int) error {
	if _, err := s.competitions.GetByID(ctx, competitionID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrCompetitionNotFound
		}
		return fmt.Errorf("get competition: %w", err)
	}

	targetShooter, err := s.shooters.GetByLane(ctx, competitionID, lane)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrStageLaneNotFound
		}
		return fmt.Errorf("get shooter: %w", err)
	}

	stage, err := s.stages.GetByShooterAndNumber(ctx, targetShooter.ID, stageNumber)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrShooterNotFound
		}
		return fmt.Errorf("get stage: %w", err)
	}

	if err := s.locks.ClearForStage(ctx, stage.ID); err != nil {
		return fmt.Errorf("clear locks: %w", err)
	}

	if err := s.locks.SetFullyLocked(ctx, stage.ID, false); err != nil {
		return fmt.Errorf("clear fully locked: %w", err)
	}

	return nil
}
