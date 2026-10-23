package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/restore"
	"github.com/sollie/matchmark/internal/rulebook"
)

const (
	restoredSex = "M"
)

var ErrRestoreTargetNotEmpty = errors.New("database already contains data; use --merge to apply only data whose ids match")

func (s *Service) RestoreScoreLog(ctx context.Context, plan restore.Plan, merge bool) (restore.Applied, error) {
	var total restore.Applied
	competitions, err := s.repo.CountCompetitions(ctx)
	if err != nil {
		return total, err
	}
	shooters, err := s.repo.CountShooters(ctx)
	if err != nil {
		return total, err
	}
	nonEmpty := competitions > 0 || shooters > 0
	if nonEmpty && !merge {
		return total, ErrRestoreTargetNotEmpty
	}
	var toWrite []repository.RestoreCompetition
	for _, c := range plan.Competitions {
		rc, err := buildRestoreCompetition(c)
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
		if rc.Competition.ID != 0 {
			toWrite = append(toWrite, rc)
		}
	}
	counts, err := s.repo.RestoreCompetitions(ctx, toWrite)
	if err != nil {
		return restore.Applied{}, fmt.Errorf("restore score log: %w", mapRepoErr(err))
	}
	s.changes.Record(ctx, Change{Live: LiveAll})
	total.Competitions = counts.Competitions
	total.Entries = counts.Entries
	total.Stages = counts.Stages
	return total, nil
}

func ringCounts(c [7]int) repository.RingCounts {
	return repository.RingCounts{X: c[0], Ten: c[1], Nine: c[2], Eight: c[3], Seven: c[4], Zero: c[5], Miss: c[6]}
}

func buildRestoreCompetition(c restore.RestoredCompetition) (repository.RestoreCompetition, error) {
	course, err := rulebook.CourseByKey(c.CourseOfFire.Key)
	if err != nil {
		return repository.RestoreCompetition{}, fmt.Errorf("competition %d: %w", c.ID, err)
	}
	expected := map[int]int{}
	for _, st := range course.Stages {
		expected[st.StageNumber] = st.ExpectedShots
	}
	rc := repository.RestoreCompetition{Competition: repository.Competition{
		ID: c.ID, Name: c.Name, Date: c.Date, Course: c.CourseOfFire.Key,
	}}
	allAccepted := true
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
	rc.Competition.Status = string(StatusActive)
	if allAccepted {
		rc.Competition.Status = string(StatusFinished)
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
				conflicts = append(conflicts, fmt.Sprintf("competition %d entry %d stage %d: database has %+v locked by %s at %s, log has %+v locked by %s at %s",
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

func (s *Service) entryConflict(ctx context.Context, competitionID int64, e repository.RestoreEntry) (string, bool, error) {
	have, err := s.repo.GetEntry(ctx, competitionID, e.ID)
	if errors.Is(err, repository.ErrNotFound) {
		elsewhere, err := s.repo.EntryExists(ctx, e.ID)
		if err != nil {
			return "", false, err
		}
		if elsewhere {
			return fmt.Sprintf("competition %d entry %d: id belongs to another competition in the database", competitionID, e.ID), true, nil
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
		return fmt.Sprintf("competition %d entry %d: database has %s %s, lane %d, %s; log has %s %s, lane %d, %s",
			competitionID, e.ID, shooter.FirstName, shooter.LastName, have.Lane, have.Category,
			e.Shooter.FirstName, e.Shooter.LastName, e.Lane, e.Category), true, nil
	}
	return "", false, nil
}
