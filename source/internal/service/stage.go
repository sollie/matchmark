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

func (s *Service) SetRingCounts(ctx context.Context, competitionID, entryID int64, stageNumber int, c repository.RingCounts) error {
	stage, err := s.repo.GetStage(ctx, competitionID, entryID, stageNumber)
	if err != nil {
		return mapRepoErr(err)
	}
	if stage.Locked {
		return ErrStageLocked
	}
	if err := ValidateRingCounts(c, stage.ExpectedShots); err != nil {
		return err
	}
	return mapRepoErr(s.repo.SetRingCounts(ctx, stage.ID, c))
}
