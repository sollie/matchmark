package service

import (
	"context"

	"github.com/sollie/matchmark/internal/repository"
)

func ValidateRingCounts(c repository.RingCounts, expectedShots int) error {
	counts := []int{c.X, c.Ten, c.Nine, c.Eight, c.Seven, c.Zero, c.Miss}
	sum := 0
	for _, n := range counts {
		if n < 0 {
			return ErrInvalid
		}
		sum += n
	}
	if sum != expectedShots {
		return ErrInvalid
	}
	return nil
}

func (s *Service) validatedStage(ctx context.Context, competitionID, entryID int64, stageNumber int, c repository.RingCounts, requireUnlocked bool) (repository.Stage, error) {
	stage, err := s.repo.GetStage(ctx, competitionID, entryID, stageNumber)
	if err != nil {
		return repository.Stage{}, mapRepoErr(err)
	}
	if requireUnlocked && stage.Locked {
		return repository.Stage{}, ErrStageLocked
	}
	if err := ValidateRingCounts(c, stage.ExpectedShots); err != nil {
		return repository.Stage{}, err
	}
	return stage, nil
}

func (s *Service) SetRingCounts(ctx context.Context, competitionID, entryID int64, stageNumber int, c repository.RingCounts) error {
	stage, err := s.validatedStage(ctx, competitionID, entryID, stageNumber, c, true)
	if err != nil {
		return err
	}
	if err := s.repo.SetRingCounts(ctx, stage.ID, c); err != nil {
		if err == repository.ErrConflict {
			return ErrStageLocked
		}
		return mapRepoErr(err)
	}
	return nil
}

func (s *Service) AdminSetRingCounts(ctx context.Context, competitionID, entryID int64, stageNumber int, c repository.RingCounts) error {
	stage, err := s.validatedStage(ctx, competitionID, entryID, stageNumber, c, false)
	if err != nil {
		return err
	}
	if err := s.repo.OverwriteRingCounts(ctx, stage.ID, c, stage.Locked); err != nil {
		return mapRepoErr(err)
	}
	if stage.Locked {
		s.emitStageEvent(ctx, eventStageLocked, competitionID, entryID, stageNumber)
	}
	return nil
}
