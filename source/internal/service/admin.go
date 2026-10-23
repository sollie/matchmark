package service

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
)

var ErrNotBetweenStages = errors.New("entries can only change between stages")

type CompetitionStatus string

const (
	StatusDraft    CompetitionStatus = "draft"
	StatusActive   CompetitionStatus = "active"
	StatusFinished CompetitionStatus = "finished"
)

var statusTransitions = map[CompetitionStatus]CompetitionStatus{StatusDraft: StatusActive, StatusActive: StatusFinished}

func (st CompetitionStatus) Valid() bool {
	switch st {
	case StatusDraft, StatusActive, StatusFinished:
		return true
	}
	return false
}

func validShooter(sh repository.Shooter) bool {
	return strings.TrimSpace(sh.LastName) != "" && strings.TrimSpace(sh.FirstName) != "" &&
		(sh.Sex == "M" || sh.Sex == "F") && sh.Country != "" && sh.Club != "" &&
		len(sh.LastName) <= 128 && len(sh.FirstName) <= 128 && len(sh.WAID) <= 32 && len(sh.Club) <= 128
}

func (s *Service) ListShooters(ctx context.Context) ([]repository.Shooter, error) {
	return s.repo.ListShooters(ctx)
}

func (s *Service) GetShooter(ctx context.Context, id int64) (repository.Shooter, error) {
	sh, err := s.repo.GetShooter(ctx, id)
	return sh, mapRepoErr(err)
}

func (s *Service) UpdateShooter(ctx context.Context, sh repository.Shooter) error {
	if !validShooter(sh) {
		return ErrInvalid
	}
	if err := s.repo.UpdateShooter(ctx, sh); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "shooter.update", Target: idTarget("shooter", sh.ID)}, Live: LiveAll, Reemit: &repository.LockedStageFilter{ShooterID: sh.ID}})
	return nil
}

func (s *Service) DeleteShooter(ctx context.Context, id int64) error {
	if err := s.repo.DeleteShooter(ctx, id); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "shooter.delete", Target: idTarget("shooter", id)}, Live: LiveAll})
	return nil
}

func (s *Service) ListCompetitions(ctx context.Context) ([]repository.Competition, error) {
	return s.repo.ListCompetitions(ctx)
}

func (s *Service) GetCompetition(ctx context.Context, id int64) (repository.Competition, error) {
	c, err := s.repo.GetCompetition(ctx, id)
	return c, mapRepoErr(err)
}

func (s *Service) UpdateCompetition(ctx context.Context, c repository.Competition) error {
	if strings.TrimSpace(c.Name) == "" || len(c.Name) > 128 || !validDate(c.Date) {
		return ErrInvalid
	}
	if _, err := rulebook.CourseByKey(c.Course); err != nil {
		return ErrInvalid
	}
	current, err := s.repo.GetCompetition(ctx, c.ID)
	if err != nil {
		return mapRepoErr(err)
	}
	if CompetitionStatus(current.Status) != StatusDraft {
		return ErrConflict
	}
	if c.Course != current.Course {
		n, err := s.repo.CountEntries(ctx, c.ID)
		if err != nil {
			return err
		}
		if n > 0 {
			return ErrConflict
		}
	}
	if err := s.repo.UpdateCompetition(ctx, c); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "competition.update", Target: idTarget("competition", c.ID), Details: c.Name}, CompetitionID: c.ID, Live: LiveCompetition, Reemit: &repository.LockedStageFilter{CompetitionID: c.ID}})
	return nil
}

func (s *Service) SetCompetitionStatus(ctx context.Context, id int64, status CompetitionStatus) error {
	if !status.Valid() {
		return ErrInvalid
	}
	current, err := s.repo.GetCompetition(ctx, id)
	if err != nil {
		return mapRepoErr(err)
	}
	if statusTransitions[CompetitionStatus(current.Status)] != status {
		return ErrConflict
	}
	return s.changeStatus(ctx, id, CompetitionStatus(current.Status), status, nil)
}

func (s *Service) changeStatus(ctx context.Context, id int64, from, status CompetitionStatus, actor *Actor) error {
	if err := s.repo.SetCompetitionStatus(ctx, id, string(from), string(status)); err != nil {
		return mapRepoErr(err)
	}
	record := Change{Audit: AuditEvent{Action: "competition.status", Target: idTarget("competition", id), Details: string(status)}, Actor: actor, CompetitionID: id, Live: LiveCompetition}
	if status != StatusFinished {
		s.changes.Record(ctx, record)
		return nil
	}
	s.changes.Record(ctx, Change{Audit: record.Audit, Actor: actor})
	if err := s.repo.RevokeCompetitionPairings(ctx, id); err != nil {
		return mapRepoErr(err)
	}
	if s.replicator != nil {
		s.replicator.Request()
	}
	s.changes.Record(ctx, Change{CompetitionID: id, Live: LiveCompetition})
	return nil
}

func (s *Service) DeleteCompetition(ctx context.Context, id int64) error {
	if err := s.repo.DeleteCompetition(ctx, id); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "competition.delete", Target: idTarget("competition", id)}, CompetitionID: id, Live: LiveCompetition})
	return nil
}

type StageScoring struct {
	repository.StageInfo
	Score        int
	ShotsEntered int
}

type EntryScoring struct {
	repository.Entry
	Stages []StageScoring
}

func (s *Service) ListEntries(ctx context.Context, competitionID int64) ([]EntryScoring, error) {
	if _, err := s.repo.GetCompetition(ctx, competitionID); err != nil {
		return nil, mapRepoErr(err)
	}
	entries, err := s.repo.ListEntries(ctx, competitionID)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	out := make([]EntryScoring, 0, len(entries))
	for _, e := range entries {
		p := EntryScoring{Entry: e.Entry, Stages: make([]StageScoring, 0, len(e.Stages))}
		for _, st := range e.Stages {
			c := st.Counts
			p.Stages = append(p.Stages, StageScoring{
				StageInfo:    st,
				Score:        stagePoints(c),
				ShotsEntered: c.X + c.Ten + c.Nine + c.Eight + c.Seven + c.Zero + c.Miss,
			})
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *Service) requireDraft(ctx context.Context, competitionID int64) (repository.Competition, error) {
	comp, err := s.repo.GetCompetition(ctx, competitionID)
	if err != nil {
		return comp, mapRepoErr(err)
	}
	if CompetitionStatus(comp.Status) != StatusDraft {
		return comp, ErrConflict
	}
	return comp, nil
}

func (s *Service) requireWithdrawalAllowed(ctx context.Context, competitionID int64) error {
	comp, err := s.repo.GetCompetition(ctx, competitionID)
	if err != nil {
		return mapRepoErr(err)
	}
	switch CompetitionStatus(comp.Status) {
	case StatusDraft:
		return nil
	case StatusFinished:
		return ErrConflict
	}
	entries, err := s.repo.ListEntries(ctx, competitionID)
	if err != nil {
		return err
	}
	if !stageAccepted(entries, comp.CurrentStage) {
		return ErrNotBetweenStages
	}
	return nil
}

func stageAccepted(entries []repository.EntryWithStages, stageNumber int) bool {
	for _, e := range entries {
		if e.Withdrawn {
			continue
		}
		accepted := false
		for _, st := range e.Stages {
			if st.Number == stageNumber && st.Status == repository.StatusAccepted {
				accepted = true
			}
		}
		if !accepted {
			return false
		}
	}
	return true
}

func (s *Service) StartNextStage(ctx context.Context, competitionID int64) (int, error) {
	comp, err := s.repo.GetCompetition(ctx, competitionID)
	if err != nil {
		return 0, mapRepoErr(err)
	}
	if CompetitionStatus(comp.Status) != StatusActive {
		return 0, ErrConflict
	}
	last, err := lastStage(comp.Course)
	if err != nil {
		return 0, err
	}
	if comp.CurrentStage >= last {
		return 0, ErrConflict
	}
	entries, err := s.repo.ListEntries(ctx, competitionID)
	if err != nil {
		return 0, err
	}
	if !stageAccepted(entries, comp.CurrentStage) {
		return 0, ErrStageGateNotMet
	}
	if err := s.repo.AdvanceStage(ctx, competitionID, comp.CurrentStage); err != nil {
		return 0, mapRepoErr(err)
	}
	next := comp.CurrentStage + 1
	s.changes.Record(ctx, Change{
		Audit:         AuditEvent{Action: "competition.next_stage", Target: idTarget("competition", competitionID), Details: "stage " + strconv.Itoa(next)},
		CompetitionID: competitionID,
		Live:          LiveCompetition,
	})
	return next, nil
}

func lastStage(course string) (int, error) {
	c, err := rulebook.CourseByKey(course)
	if err != nil {
		return 0, err
	}
	last := 0
	for _, st := range c.Stages {
		last = max(last, st.StageNumber)
	}
	return last, nil
}

func (s *Service) finishIfComplete(ctx context.Context, competitionID int64, actor *Actor) {
	comp, err := s.repo.GetCompetition(ctx, competitionID)
	if err != nil || CompetitionStatus(comp.Status) != StatusActive {
		return
	}
	last, err := lastStage(comp.Course)
	if err != nil || comp.CurrentStage != last {
		return
	}
	entries, err := s.repo.ListEntries(ctx, competitionID)
	if err != nil || !stageAccepted(entries, last) {
		return
	}
	_ = s.changeStatus(ctx, competitionID, StatusActive, StatusFinished, actor)
}

func (s *Service) UpdateEntry(ctx context.Context, competitionID, entryID int64, category string, lane int) (repository.Entry, error) {
	if lane < 1 || !contains(categories, category) {
		return repository.Entry{}, ErrInvalid
	}
	if _, err := s.repo.GetEntry(ctx, competitionID, entryID); err != nil {
		return repository.Entry{}, mapRepoErr(err)
	}
	if _, err := s.requireDraft(ctx, competitionID); err != nil {
		return repository.Entry{}, err
	}
	if err := s.repo.UpdateEntry(ctx, competitionID, entryID, category, lane); err != nil {
		return repository.Entry{}, mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "entry.update", Target: entryTarget(competitionID, entryID)}, CompetitionID: competitionID, Live: LiveCompetition, Reemit: &repository.LockedStageFilter{CompetitionID: competitionID, EntryID: entryID}})
	e, err := s.repo.GetEntry(ctx, competitionID, entryID)
	return e, mapRepoErr(err)
}

func (s *Service) DeleteEntry(ctx context.Context, competitionID, entryID int64) error {
	comp, err := s.repo.GetCompetition(ctx, competitionID)
	if err != nil {
		return mapRepoErr(err)
	}
	if CompetitionStatus(comp.Status) != StatusDraft {
		return ErrConflict
	}
	if err := s.repo.DeleteEntry(ctx, competitionID, entryID); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "entry.delete", Target: entryTarget(competitionID, entryID)}, CompetitionID: competitionID, Live: LiveCompetition})
	return nil
}
