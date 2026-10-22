package service

import (
	"context"

	"github.com/sollie/matchmark/internal/repository"
)

func ValidateRingCounts(c repository.RingCounts, expectedShots int) error {
	counts := []int{c.X, c.Ten, c.Nine, c.Eight, c.Seven, c.Zero, c.Miss}
	sum := 0
	for _, n := range counts {
		if n < 0 || n > expectedShots {
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
	if err := s.writeRingCounts(ctx, competitionID, entryID, stageNumber, c); err != nil {
		return err
	}
	s.changes.Record(ctx, Change{CompetitionID: competitionID, Live: LiveCompetition})
	return nil
}

func (s *Service) writeRingCounts(ctx context.Context, competitionID, entryID int64, stageNumber int, c repository.RingCounts) error {
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
	if err := s.repo.OverwriteRingCounts(ctx, stage.ID, c, lockedByUser(ctx, stage.Locked)); err != nil {
		return mapRepoErr(err)
	}
	change := Change{
		Audit:         AuditEvent{Action: "stage.set_counts", Target: stageTarget(competitionID, entryID, stageNumber)},
		CompetitionID: competitionID,
		Live:          LiveCompetition,
	}
	if stage.Locked {
		change.Stage = &StageEvent{Event: eventStageLocked, CompetitionID: competitionID, EntryID: entryID, StageNumber: stageNumber}
	}
	s.changes.Record(ctx, change)
	return nil
}

func lockedByUser(ctx context.Context, locked bool) string {
	if !locked {
		return ""
	}
	return ActorFrom(ctx).Username
}
