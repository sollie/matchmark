package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/restore"
	"github.com/sollie/matchmark/internal/rulebook"
)

const (
	restoredSex    = "M"
	restoredStatus = "finished"
)

type RestoreResult struct {
	Competitions int
	Entries      int
	Stages       int
}

func (s *Service) RestoreScoreLog(ctx context.Context, plan restore.Plan) (RestoreResult, error) {
	var total RestoreResult
	for _, c := range plan.Competitions {
		rc, err := restoreCompetition(c)
		if err != nil {
			return total, err
		}
		counts, err := s.repo.RestoreCompetition(ctx, rc)
		if err != nil {
			return total, fmt.Errorf("restore competition %d: %w", c.ID, mapRepoErr(err))
		}
		total.Competitions += counts.Competitions
		total.Entries += counts.Entries
		total.Stages += counts.Stages
	}
	return total, nil
}

func restoreCompetition(c restore.RestoredCompetition) (repository.RestoreCompetition, error) {
	course, err := rulebook.CourseByKey(c.CourseOfFire.Key)
	if err != nil {
		return repository.RestoreCompetition{}, fmt.Errorf("competition %d: %w", c.ID, err)
	}
	expected := map[int]int{}
	for _, st := range course.Stages {
		expected[st.StageNumber] = st.ExpectedShots
	}
	rc := repository.RestoreCompetition{Competition: repository.Competition{
		ID: c.ID, Name: c.Name, Date: c.Date, Course: c.CourseOfFire.Key, Status: restoredStatus,
	}}
	for _, e := range c.Entries {
		re := repository.RestoreEntry{
			ID: e.ID, Category: e.Category, Lane: e.Lane,
			Shooter: repository.Shooter{
				WAID: e.Shooter.WAID, FirstName: e.Shooter.FirstName, LastName: e.Shooter.LastName,
				Club: e.Shooter.Club, Sex: restoredSex,
			},
		}
		scored := map[int]restore.ScoredStage{}
		for _, st := range e.Stages {
			if _, ok := expected[st.Number]; !ok {
				return repository.RestoreCompetition{}, fmt.Errorf("competition %d entry %d: unknown stage %d", c.ID, e.ID, st.Number)
			}
			scored[st.Number] = st
		}
		for _, cs := range course.Stages {
			rs := repository.RestoreStage{Number: cs.StageNumber, Expected: cs.ExpectedShots}
			if st, ok := scored[cs.StageNumber]; ok {
				rs.Scored = true
				rs.Counts = repository.RingCounts{
					X: st.RingCounts[0], Ten: st.RingCounts[1], Nine: st.RingCounts[2], Eight: st.RingCounts[3],
					Seven: st.RingCounts[4], Zero: st.RingCounts[5], Miss: st.RingCounts[6],
				}
				rs.ScorerName = st.ScorerName
				rs.LockedAt = st.LockedAt
				if lane, err := strconv.Atoi(st.LockedBy); err == nil {
					rs.ScorerLane = &lane
				}
			}
			re.Stages = append(re.Stages, rs)
		}
		rc.Entries = append(rc.Entries, re)
	}
	return rc, nil
}
