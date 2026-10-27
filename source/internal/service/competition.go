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
	if editing && r.Matches == 1 {
		return nil
	}
	if date < r.Start || date > r.End {
		return ErrInvalid
	}
	return nil
}

func (s *Service) CreateCompetition(ctx context.Context, c repository.Competition) (int64, error) {
	if strings.TrimSpace(c.Name) == "" || len(c.Name) > 128 || !validDate(c.Date) {
		return 0, ErrInvalid
	}
	if _, err := rulebook.CourseByKey(c.Course); err != nil {
		return 0, ErrInvalid
	}
	if err := s.checkMatchDate(ctx, c.ContainerID, c.Date, false); err != nil {
		return 0, err
	}
	id, err := s.repo.CreateCompetition(ctx, c)
	if err != nil {
		return id, mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "competition.create", Target: idTarget("competition", id), Details: c.Name}})
	return id, nil
}

func (s *Service) AddEntry(ctx context.Context, e repository.Entry) (int64, error) {
	if e.Lane < 1 {
		return 0, ErrInvalid
	}
	comp, err := s.requireDraft(ctx, e.CompetitionID)
	if err != nil {
		return 0, err
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
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "entry.add", Target: entryTarget(e.CompetitionID, id)}, CompetitionID: e.CompetitionID, Live: LiveCompetition})
	return id, nil
}

func (s *Service) WithdrawEntry(ctx context.Context, competitionID, entryID int64) error {
	if err := s.requireWithdrawalAllowed(ctx, competitionID); err != nil {
		return err
	}
	if err := s.repo.WithdrawEntry(ctx, competitionID, entryID); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "entry.withdraw", Target: entryTarget(competitionID, entryID)}, CompetitionID: competitionID, Live: LiveCompetition})
	return nil
}

func idTarget(kind string, id int64) string {
	return kind + ":" + strconv.FormatInt(id, 10)
}

func entryTarget(competitionID, entryID int64) string {
	return "competition:" + strconv.FormatInt(competitionID, 10) + "/entry:" + strconv.FormatInt(entryID, 10)
}

func stageTarget(competitionID, entryID int64, stage int) string {
	return entryTarget(competitionID, entryID) + "/stage:" + strconv.Itoa(stage)
}
