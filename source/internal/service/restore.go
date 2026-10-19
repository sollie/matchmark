package service

import (
	"context"
	"errors"
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

var ErrRestoreTargetNotEmpty = errors.New("database already contains data; use --merge to apply only data whose ids match")

type RestoreResult struct {
	Competitions int
	Entries      int
	Stages       int
	Conflicts    []string
}

func (s *Service) RestoreScoreLog(ctx context.Context, plan restore.Plan, merge bool) (RestoreResult, error) {
	var total RestoreResult
	existing, err := s.repo.CountCompetitions(ctx)
	if err != nil {
		return total, err
	}
	if existing > 0 && !merge {
		return total, ErrRestoreTargetNotEmpty
	}
	for _, c := range plan.Competitions {
		rc, err := restoreCompetition(c)
		if err != nil {
			return total, err
		}
		if existing > 0 {
			var conflicts []string
			rc, conflicts, err = s.dropConflicts(ctx, rc)
			if err != nil {
				return total, err
			}
			total.Conflicts = append(total.Conflicts, conflicts...)
		}
		if rc.Competition.ID == 0 {
			continue
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

func (s *Service) dropConflicts(ctx context.Context, rc repository.RestoreCompetition) (repository.RestoreCompetition, []string, error) {
	id := rc.Competition.ID
	var conflicts []string
	have, err := s.repo.GetCompetition(ctx, id)
	switch {
	case err == nil:
		if have.Name != rc.Competition.Name || have.Date != rc.Competition.Date || have.Course != rc.Competition.Course {
			conflicts = append(conflicts, fmt.Sprintf("competition %d: database has %q (%s, %s), log has %q (%s, %s)",
				id, have.Name, have.Date, have.Course, rc.Competition.Name, rc.Competition.Date, rc.Competition.Course))
			return repository.RestoreCompetition{}, conflicts, nil
		}
	case !errors.Is(err, repository.ErrNotFound):
		return rc, nil, err
	}
	var entries []repository.RestoreEntry
	for _, e := range rc.Entries {
		entryConflict, err := s.entryConflict(ctx, id, e)
		if err != nil {
			return rc, nil, err
		}
		if entryConflict != "" {
			conflicts = append(conflicts, entryConflict)
			continue
		}
		for i, st := range e.Stages {
			if !st.Scored {
				continue
			}
			cur, err := s.repo.GetStage(ctx, id, e.ID, st.Number)
			if errors.Is(err, repository.ErrNotFound) {
				continue
			}
			if err != nil {
				return rc, nil, err
			}
			if cur.Locked && cur.Counts != st.Counts {
				conflicts = append(conflicts, fmt.Sprintf("competition %d entry %d stage %d: database has %+v locked, log has %+v",
					id, e.ID, st.Number, cur.Counts, st.Counts))
				e.Stages[i].Scored = false
			}
		}
		entries = append(entries, e)
	}
	rc.Entries = entries
	return rc, conflicts, nil
}

func (s *Service) entryConflict(ctx context.Context, competitionID int64, e repository.RestoreEntry) (string, error) {
	have, err := s.repo.GetEntry(ctx, competitionID, e.ID)
	if errors.Is(err, repository.ErrNotFound) {
		elsewhere, err := s.repo.EntryExists(ctx, e.ID)
		if err != nil {
			return "", err
		}
		if elsewhere {
			return fmt.Sprintf("competition %d entry %d: id belongs to another competition in the database", competitionID, e.ID), nil
		}
		return "", nil
	}
	if err != nil {
		return "", err
	}
	shooter, err := s.repo.GetShooter(ctx, have.ShooterID)
	if err != nil {
		return "", err
	}
	if have.Lane != e.Lane || have.Category != e.Category || shooter.FirstName != e.Shooter.FirstName || shooter.LastName != e.Shooter.LastName {
		return fmt.Sprintf("competition %d entry %d: database has %s %s, lane %d, %s; log has %s %s, lane %d, %s",
			competitionID, e.ID, shooter.FirstName, shooter.LastName, have.Lane, have.Category,
			e.Shooter.FirstName, e.Shooter.LastName, e.Lane, e.Category), nil
	}
	return "", nil
}
