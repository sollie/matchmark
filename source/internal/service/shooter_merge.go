package service

import (
	"context"

	"github.com/sollie/matchmark/internal/repository"
)

type MergeConflict struct {
	MatchID   int64
	MatchName string
	MatchDate string
}

type ShooterMergePreview struct {
	Source        repository.Shooter
	Target        repository.Shooter
	Entries       []repository.ShooterEntryRef
	Conflicts     []MergeConflict
	WAIDConflict  bool
	ResultingWAID string
}

func (p ShooterMergePreview) CanMerge() bool {
	return !p.WAIDConflict && len(p.Conflicts) == 0
}

type MergeBlockedError struct {
	Preview ShooterMergePreview
}

func (e *MergeBlockedError) Error() string { return "shooter merge is blocked" }

func (e *MergeBlockedError) Is(target error) bool { return target == ErrConflict }

func (s *Service) PreviewShooterMerge(ctx context.Context, sourceID, targetID int64) (ShooterMergePreview, error) {
	if sourceID == targetID {
		return ShooterMergePreview{}, ErrInvalid
	}
	source, err := s.repo.GetShooter(ctx, sourceID)
	if err != nil {
		return ShooterMergePreview{}, mapRepoErr(err)
	}
	target, err := s.repo.GetShooter(ctx, targetID)
	if err != nil {
		return ShooterMergePreview{}, mapRepoErr(err)
	}
	sourceEntries, err := s.repo.ShooterEntries(ctx, sourceID)
	if err != nil {
		return ShooterMergePreview{}, err
	}
	targetEntries, err := s.repo.ShooterEntries(ctx, targetID)
	if err != nil {
		return ShooterMergePreview{}, err
	}
	targetMatches := make(map[int64]bool, len(targetEntries))
	for _, e := range targetEntries {
		targetMatches[e.MatchID] = true
	}
	preview := ShooterMergePreview{Source: source, Target: target, Entries: sourceEntries, Conflicts: []MergeConflict{}}
	for _, e := range sourceEntries {
		if targetMatches[e.MatchID] {
			preview.Conflicts = append(preview.Conflicts, MergeConflict{MatchID: e.MatchID, MatchName: e.MatchName, MatchDate: e.MatchDate})
		}
	}
	preview.WAIDConflict = source.WAID != "" && target.WAID != "" && source.WAID != target.WAID
	preview.ResultingWAID = target.WAID
	if preview.ResultingWAID == "" {
		preview.ResultingWAID = source.WAID
	}
	return preview, nil
}

func (s *Service) MergeShooters(ctx context.Context, sourceID, targetID int64) (ShooterMergePreview, error) {
	preview, err := s.PreviewShooterMerge(ctx, sourceID, targetID)
	if err != nil {
		return preview, err
	}
	if !preview.CanMerge() {
		return preview, &MergeBlockedError{Preview: preview}
	}
	if err := s.repo.MergeShooters(ctx, sourceID, targetID, preview.ResultingWAID, shooterName(preview.Target)); err != nil {
		return preview, mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{
		Audit:  AuditEvent{Action: "shooter.merge", Target: idTarget("shooter", sourceID), Details: "into " + idTarget("shooter", targetID)},
		Live:   LiveAll,
		Reemit: &repository.LockedStageFilter{ShooterID: targetID},
	})
	return preview, nil
}
