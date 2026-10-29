package service

import (
	"context"
	"errors"

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

func (s *Service) validatedSegment(ctx context.Context, matchID, entryID int64, segmentNumber int, c repository.RingCounts, requireUnlocked bool) (repository.Segment, error) {
	segment, err := s.repo.GetSegment(ctx, matchID, entryID, segmentNumber)
	if err != nil {
		return repository.Segment{}, mapRepoErr(err)
	}
	if requireUnlocked && segment.Status != repository.StatusOpen {
		return repository.Segment{}, ErrSegmentLocked
	}
	if err := ValidateRingCounts(c, segment.ExpectedShots); err != nil {
		return repository.Segment{}, err
	}
	return segment, nil
}

func (s *Service) SetRingCounts(ctx context.Context, matchID, entryID int64, segmentNumber int, c repository.RingCounts) error {
	if err := s.writeRingCounts(ctx, matchID, entryID, segmentNumber, c, false); err != nil {
		return err
	}
	s.changes.Record(ctx, Change{MatchID: matchID, Live: LiveMatch})
	return nil
}

func (s *Service) writeRingCounts(ctx context.Context, matchID, entryID int64, segmentNumber int, c repository.RingCounts, activeOnly bool) error {
	segment, err := s.validatedSegment(ctx, matchID, entryID, segmentNumber, c, true)
	if err != nil {
		return err
	}
	if err := s.repo.SetRingCounts(ctx, segment.ID, c, activeOnly); err != nil {
		if err == repository.ErrConflict {
			if m, gerr := s.repo.GetMatch(ctx, matchID); gerr == nil && MatchStatus(m.Status) != StatusActive {
				return ErrConflict
			}
			return ErrSegmentLocked
		}
		return mapRepoErr(err)
	}
	return nil
}

func (s *Service) AdminSetRingCounts(ctx context.Context, matchID, entryID int64, segmentNumber int, c repository.RingCounts) error {
	segment, err := s.validatedSegment(ctx, matchID, entryID, segmentNumber, c, false)
	if err != nil {
		return err
	}
	username := ActorFrom(ctx).Username
	ev, err := s.userSegmentEvent(ctx, repository.EventAccepted, username, matchID, entryID, segment)
	if err != nil {
		return err
	}
	if err := s.repo.OverwriteRingCounts(ctx, segment, c, username, ev); err != nil {
		return mapSegmentWriteErr(err)
	}
	change := Change{
		Audit:   AuditEvent{Action: "segment.set_counts", Target: segmentTarget(matchID, entryID, segmentNumber)},
		MatchID: matchID,
		Live:    LiveMatch,
	}
	if segment.Status != repository.StatusOpen {
		change.Segment = &SegmentEvent{MatchID: matchID, EntryID: entryID, SegmentNumber: segmentNumber}
	}
	s.changes.Record(ctx, change)
	if segment.Status != repository.StatusOpen {
		s.finishIfComplete(ctx, matchID, nil)
	}
	return nil
}

func mapSegmentWriteErr(err error) error {
	if errors.Is(err, repository.ErrConflict) {
		return ErrSegmentLocked
	}
	return mapRepoErr(err)
}

func (s *Service) userSegmentEvent(ctx context.Context, kind repository.SegmentEventKind, username string, matchID, entryID int64, segment repository.Segment) (repository.SegmentEvent, error) {
	entry, err := s.repo.GetEntry(ctx, matchID, entryID)
	if err != nil {
		return repository.SegmentEvent{}, mapRepoErr(err)
	}
	shooter, err := s.repo.GetShooter(ctx, entry.ShooterID)
	if err != nil {
		return repository.SegmentEvent{}, mapRepoErr(err)
	}
	return repository.SegmentEvent{
		MatchID: matchID, EntryID: entryID, SegmentNumber: segment.Number, Event: kind,
		ActorType: "user", ActorName: username, ScorerName: deviceScorerName(segment), ShooterName: shooterName(shooter),
	}, nil
}

func deviceScorerName(segment repository.Segment) string {
	if segment.ScorerLane == nil {
		return ""
	}
	return segment.ScorerName
}
