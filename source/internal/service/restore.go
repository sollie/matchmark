package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/restore"
	"github.com/sollie/matchmark/internal/rulebook"
)

var ErrRestoreTargetNotEmpty = errors.New("database already contains data; use --merge to apply only data whose ids match")

func (s *Service) RestoreScoreLog(ctx context.Context, plan restore.Plan, merge bool) (restore.Applied, error) {
	var total restore.Applied
	matches, err := s.repo.CountMatches(ctx)
	if err != nil {
		return total, err
	}
	shooters, err := s.repo.CountShooters(ctx)
	if err != nil {
		return total, err
	}
	nonEmpty := matches > 0 || shooters > 0
	if nonEmpty && !merge {
		return total, ErrRestoreTargetNotEmpty
	}
	var toWrite []repository.RestoreMatch
	for _, c := range plan.Matches {
		rc, err := buildRestoreMatch(c)
		if err != nil {
			return total, err
		}
		if nonEmpty {
			var conflicts []string
			rc, conflicts, err = s.dropConflicts(ctx, rc)
			if err != nil {
				return total, err
			}
			total.Conflicts = append(total.Conflicts, conflicts...)
		}
		if rc.Match.ID != 0 {
			toWrite = append(toWrite, rc)
		}
	}
	counts, err := s.repo.RestoreMatches(ctx, toWrite)
	if err != nil {
		return restore.Applied{}, fmt.Errorf("restore score log: %w", mapRepoErr(err))
	}
	s.changes.Record(ctx, Change{Live: LiveAll, Screens: true})
	total.Matches = counts.Matches
	total.Entries = counts.Entries
	total.Stages = counts.Stages
	return total, nil
}

func ringCounts(c [7]int) repository.RingCounts {
	return repository.RingCounts{X: c[0], Ten: c[1], Nine: c[2], Eight: c[3], Seven: c[4], Zero: c[5], Miss: c[6]}
}

func buildRestoreMatch(c restore.RestoredMatch) (repository.RestoreMatch, error) {
	course, err := rulebook.CourseByKey(c.CourseOfFire.Key)
	if err != nil {
		return repository.RestoreMatch{}, fmt.Errorf("match %d: %w", c.ID, err)
	}
	expected := map[int]int{}
	for _, st := range course.Stages {
		expected[st.StageNumber] = st.ExpectedShots
	}
	rc := repository.RestoreMatch{CompetitionID: c.Competition.ID, CompetitionName: c.Competition.Name, Match: repository.Match{
		ID: c.ID, Name: c.Name, Date: c.Date, Course: c.CourseOfFire.Key,
	}}
	allAccepted := true
	for _, e := range c.Entries {
		re := repository.RestoreEntry{
			ID: e.ID, Category: e.Category, Lane: e.Lane,
			Shooter: repository.Shooter{
				WAID: e.Shooter.WAID, FirstName: e.Shooter.FirstName, LastName: e.Shooter.LastName,
				Club: e.Shooter.Club,
			},
		}
		scored := map[int]restore.ScoredStage{}
		for _, st := range e.Stages {
			if _, ok := expected[st.Number]; !ok {
				return repository.RestoreMatch{}, fmt.Errorf("match %d entry %d: unknown stage %d", c.ID, e.ID, st.Number)
			}
			scored[st.Number] = st
		}
		for _, cs := range course.Stages {
			rs := repository.RestoreStage{Number: cs.StageNumber, Expected: cs.ExpectedShots}
			if st, ok := scored[cs.StageNumber]; ok {
				rs.Scored = true
				rs.Status = repository.StageStatus(st.Status)
				rs.Counts = ringCounts(st.RingCounts)
				rs.ScorerName = st.ScorerName
				rs.LockedAt = st.LockedAt
				rs.AcceptedBy = st.AcceptedBy
				rs.AcceptedAt = st.AcceptedAt
				rs.DisputedAt = st.DisputedAt
				if st.Lane > 0 {
					lane := st.Lane
					rs.ScorerLane = &lane
				}
			}
			if rs.Status != repository.StatusAccepted {
				allAccepted = false
			}
			re.Stages = append(re.Stages, rs)
		}
		rc.Entries = append(rc.Entries, re)
	}
	rc.Match.Status = string(StatusActive)
	rc.Match.CurrentStage = restoredCurrentStage(rc.Entries)
	if allAccepted {
		rc.Match.Status = string(StatusFinished)
	}
	return rc, nil
}

func restoredCurrentStage(entries []repository.RestoreEntry) int {
	lowest, last := 0, 1
	for _, e := range entries {
		for _, st := range e.Stages {
			last = max(last, st.Number)
			if st.Status != repository.StatusAccepted && (lowest == 0 || st.Number < lowest) {
				lowest = st.Number
			}
		}
	}
	if lowest == 0 {
		return last
	}
	return lowest
}

func (s *Service) dropConflicts(ctx context.Context, rc repository.RestoreMatch) (repository.RestoreMatch, []string, error) {
	id := rc.Match.ID
	var conflicts []string
	have, err := s.repo.GetMatch(ctx, id)
	switch {
	case err == nil:
		if have.Name != rc.Match.Name || have.Date != rc.Match.Date || have.Course != rc.Match.Course {
			conflicts = append(conflicts, fmt.Sprintf("match %d: database has %q (%s, %s), log has %q (%s, %s)",
				id, have.Name, have.Date, have.Course, rc.Match.Name, rc.Match.Date, rc.Match.Course))
			return repository.RestoreMatch{}, conflicts, nil
		}
	case !errors.Is(err, repository.ErrNotFound):
		return rc, nil, err
	}
	kept := rc
	kept.Entries = nil
	for _, e := range rc.Entries {
		conflict, found, err := s.entryConflict(ctx, id, e)
		if err != nil {
			return rc, nil, err
		}
		if found {
			conflicts = append(conflicts, conflict)
			continue
		}
		e.Stages = append([]repository.RestoreStage(nil), e.Stages...)
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
			if cur.Status != repository.StatusOpen && !sameLock(cur, st) {
				conflicts = append(conflicts, fmt.Sprintf("match %d entry %d stage %d: database has %+v locked by %s at %s, log has %+v locked by %s at %s",
					id, e.ID, st.Number, cur.Counts, lockOwner(cur.ScorerLane, cur.ScorerName), cur.LockedAt,
					st.Counts, lockOwner(st.ScorerLane, st.ScorerName), st.LockedAt))
				e.Stages[i].Scored = false
			}
		}
		kept.Entries = append(kept.Entries, e)
	}
	return kept, conflicts, nil
}

func sameLock(cur repository.Stage, st repository.RestoreStage) bool {
	sameLane := (cur.ScorerLane == nil) == (st.ScorerLane == nil) &&
		(cur.ScorerLane == nil || *cur.ScorerLane == *st.ScorerLane)
	return cur.Counts == st.Counts && sameLane && cur.Status == st.Status && cur.ScorerName == st.ScorerName &&
		cur.LockedAt == st.LockedAt && cur.AcceptedBy == st.AcceptedBy && cur.AcceptedAt == st.AcceptedAt
}

func lockOwner(lane *int, name string) string {
	if lane == nil {
		return fmt.Sprintf("user %s", name)
	}
	return fmt.Sprintf("lane %d (%s)", *lane, name)
}

func (s *Service) entryConflict(ctx context.Context, matchID int64, e repository.RestoreEntry) (string, bool, error) {
	have, err := s.repo.GetEntry(ctx, matchID, e.ID)
	if errors.Is(err, repository.ErrNotFound) {
		elsewhere, err := s.repo.EntryExists(ctx, e.ID)
		if err != nil {
			return "", false, err
		}
		if elsewhere {
			return fmt.Sprintf("match %d entry %d: id belongs to another match in the database", matchID, e.ID), true, nil
		}
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	shooter, err := s.repo.GetShooter(ctx, have.ShooterID)
	if err != nil {
		return "", false, err
	}
	if have.Lane != e.Lane || have.Category != e.Category || shooter.FirstName != e.Shooter.FirstName || shooter.LastName != e.Shooter.LastName {
		return fmt.Sprintf("match %d entry %d: database has %s %s, lane %d, %s; log has %s %s, lane %d, %s",
			matchID, e.ID, shooter.FirstName, shooter.LastName, have.Lane, have.Category,
			e.Shooter.FirstName, e.Shooter.LastName, e.Lane, e.Category), true, nil
	}
	return "", false, nil
}
