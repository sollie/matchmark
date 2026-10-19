package service

import (
	"context"
	"slices"
)

func DeriveScoreLane(manned []int, ownLane int) (int, error) {
	if !slices.Contains(manned, ownLane) {
		return 0, ErrLaneNotManned
	}
	if len(manned) < 2 {
		return 0, ErrAdminScoringNeeded
	}
	sorted := slices.Clone(manned)
	slices.Sort(sorted)
	for _, l := range sorted {
		if l > ownLane {
			return l, nil
		}
	}
	return sorted[0], nil
}

func (s *Service) ScoreLane(ctx context.Context, competitionID int64, ownLane int) (int, error) {
	if _, err := s.repo.GetCompetition(ctx, competitionID); err != nil {
		return 0, mapRepoErr(err)
	}
	manned, err := s.repo.MannedLanes(ctx, competitionID)
	if err != nil {
		return 0, err
	}
	return DeriveScoreLane(manned, ownLane)
}
