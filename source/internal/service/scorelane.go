package service

import (
	"context"
	"slices"

	"github.com/sollie/matchmark/internal/repository"
)

type ScoreTarget struct {
	ScorerName   string
	ScoreLane    int
	CurrentStage int
	Entry        repository.EntryWithStages
	Own          repository.EntryWithStages
	ShooterName  string
}

func deriveScoreLane(manned []int, ownLane int) (int, error) {
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

func (s *Service) ScoreTarget(ctx context.Context, d Device) (ScoreTarget, error) {
	return s.resolveScoreTarget(ctx, d, true)
}

func (s *Service) ScoreLane(ctx context.Context, competitionID int64, ownLane int) (int, error) {
	t, err := s.resolveScoreTarget(ctx, Device{CompetitionID: competitionID, Lane: ownLane}, false)
	if err != nil {
		return 0, err
	}
	return t.ScoreLane, nil
}

func (s *Service) resolveScoreTarget(ctx context.Context, d Device, requireActive bool) (ScoreTarget, error) {
	comp, err := s.repo.GetCompetition(ctx, d.CompetitionID)
	if err != nil {
		return ScoreTarget{}, mapRepoErr(err)
	}
	if requireActive && CompetitionStatus(comp.Status) != StatusActive {
		return ScoreTarget{}, ErrConflict
	}
	entries, err := s.repo.ListEntries(ctx, d.CompetitionID)
	if err != nil {
		return ScoreTarget{}, err
	}
	manned := map[int]repository.EntryWithStages{}
	lanes := make([]int, 0, len(entries))
	for _, e := range entries {
		if e.Withdrawn {
			continue
		}
		manned[e.Lane] = e
		lanes = append(lanes, e.Lane)
	}
	scoreLane, err := deriveScoreLane(lanes, d.Lane)
	if err != nil {
		return ScoreTarget{}, err
	}
	scorer, err := s.repo.GetShooter(ctx, manned[d.Lane].ShooterID)
	if err != nil {
		return ScoreTarget{}, mapRepoErr(err)
	}
	target := manned[scoreLane]
	shooter, err := s.repo.GetShooter(ctx, target.ShooterID)
	if err != nil {
		return ScoreTarget{}, mapRepoErr(err)
	}
	return ScoreTarget{
		ScorerName:   shooterName(scorer),
		ScoreLane:    scoreLane,
		CurrentStage: comp.CurrentStage,
		Entry:        target,
		Own:          manned[d.Lane],
		ShooterName:  shooterName(shooter),
	}, nil
}
