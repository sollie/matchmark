package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
)

func validDate(d string) bool {
	t, err := time.Parse(time.DateOnly, d)
	return err == nil && t.Format(time.DateOnly) == d
}

func (s *Service) CreateShooter(ctx context.Context, sh repository.Shooter) (int64, error) {
	if !validShooter(sh) {
		return 0, ErrInvalid
	}
	id, err := s.repo.CreateShooter(ctx, sh)
	if err != nil {
		return id, mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "shooter.create", Target: idTarget("shooter", id)}})
	return id, nil
}

func (s *Service) checkMatchDate(ctx context.Context, containerID int64, date string, editing bool) error {
	if containerID == 0 {
		return nil
	}
	r, err := s.repo.CompetitionDateRange(ctx, containerID)
	if err != nil {
		return mapRepoErr(err)
	}
	if editing && r.Matches == 1 && r.Start == r.End {
		return nil
	}
	if date < r.Start || date > r.End {
		return ErrInvalid
	}
	return nil
}

func (s *Service) CreateMatch(ctx context.Context, c repository.Match) (int64, error) {
	if !validMatch(c) {
		return 0, ErrInvalid
	}
	if err := s.checkMatchDate(ctx, c.CompetitionID, c.Date, false); err != nil {
		return 0, err
	}
	id, err := s.repo.CreateMatch(ctx, c)
	if err != nil {
		return id, mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "match.create", Target: idTarget("match", id), Details: c.Name}})
	return id, nil
}

func validMatch(c repository.Match) bool {
	if strings.TrimSpace(c.Name) == "" || len(c.Name) > 128 || !validDate(c.Date) {
		return false
	}
	_, err := rulebook.CourseByKey(c.Course)
	return err == nil
}

func validCompetition(c repository.Competition) bool {
	return strings.TrimSpace(c.Name) != "" && len(c.Name) <= 128 &&
		validDate(c.StartDate) && validDate(c.EndDate) && c.EndDate >= c.StartDate
}

func (s *Service) CreateMatchInNewCompetition(ctx context.Context, comp repository.Competition, c repository.Match) (int64, error) {
	if !validCompetition(comp) || !validMatch(c) || c.Date < comp.StartDate || c.Date > comp.EndDate {
		return 0, ErrInvalid
	}
	compID, id, err := s.repo.CreateMatchInNewCompetition(ctx, comp, c)
	if err != nil {
		return 0, mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "competition.create", Target: idTarget("competition", compID), Details: comp.Name}})
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "match.create", Target: idTarget("match", id), Details: c.Name}})
	return id, nil
}

func (s *Service) CreateCompetition(ctx context.Context, c repository.Competition) (int64, error) {
	if !validCompetition(c) {
		return 0, ErrInvalid
	}
	id, err := s.repo.CreateCompetition(ctx, c)
	if err != nil {
		return 0, mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "competition.create", Target: idTarget("competition", id), Details: c.Name}, Live: LiveAll})
	return id, nil
}

func (s *Service) UpdateCompetition(ctx context.Context, c repository.Competition) error {
	if !validCompetition(c) {
		return ErrInvalid
	}
	if _, err := s.repo.GetCompetition(ctx, c.ID); err != nil {
		return mapRepoErr(err)
	}
	first, last, err := s.repo.MatchDateBounds(ctx, c.ID)
	if err != nil {
		return mapRepoErr(err)
	}
	if first != "" && (first < c.StartDate || last > c.EndDate) {
		return ErrInvalid
	}
	if err := s.repo.UpdateCompetition(ctx, c); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "competition.update", Target: idTarget("competition", c.ID), Details: c.Name}, Live: LiveAll})
	return nil
}

func (s *Service) DeleteCompetition(ctx context.Context, id int64) error {
	c, err := s.repo.GetCompetition(ctx, id)
	if err != nil {
		return mapRepoErr(err)
	}
	if c.MatchCount > 0 {
		return ErrConflict
	}
	if err := s.repo.DeleteCompetition(ctx, id); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "competition.delete", Target: idTarget("competition", id)}, Live: LiveAll, Screens: true})
	return nil
}

func (s *Service) AddEntry(ctx context.Context, e repository.Entry) (int64, error) {
	if e.Lane < 1 {
		return 0, ErrInvalid
	}
	comp, err := s.requireDraft(ctx, e.MatchID)
	if err != nil {
		return 0, err
	}
	shooter, err := s.repo.GetShooter(ctx, e.ShooterID)
	if err != nil {
		return 0, mapRepoErr(err)
	}
	if shooter.Archived {
		return 0, ErrShooterArchived
	}
	if rulebook.ValidateCategory(comp.Course, e.Category) != nil {
		return 0, ErrInvalid
	}
	course, err := rulebook.CourseByKey(comp.Course)
	if err != nil {
		return 0, err
	}
	seeds := make([]repository.StageSeed, 0, len(course.Stages))
	for _, st := range course.Stages {
		seeds = append(seeds, repository.StageSeed{Number: st.StageNumber, ExpectedShots: st.ExpectedShots})
	}
	id, err := s.repo.CreateEntry(ctx, e, seeds)
	if err != nil {
		return id, mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "entry.add", Target: entryTarget(e.MatchID, id)}, MatchID: e.MatchID, Live: LiveMatch})
	return id, nil
}

func (s *Service) WithdrawEntry(ctx context.Context, matchID, entryID int64) error {
	if err := s.requireWithdrawalAllowed(ctx, matchID); err != nil {
		return err
	}
	if err := s.repo.WithdrawEntry(ctx, matchID, entryID); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "entry.withdraw", Target: entryTarget(matchID, entryID)}, MatchID: matchID, Live: LiveMatch})
	return nil
}

func idTarget(kind string, id int64) string {
	return kind + ":" + strconv.FormatInt(id, 10)
}

func entryTarget(matchID, entryID int64) string {
	return "match:" + strconv.FormatInt(matchID, 10) + "/entry:" + strconv.FormatInt(entryID, 10)
}

func stageTarget(matchID, entryID int64, stage int) string {
	return entryTarget(matchID, entryID) + "/stage:" + strconv.Itoa(stage)
}
