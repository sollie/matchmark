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

type MatchStatus string

const (
	StatusDraft    MatchStatus = "draft"
	StatusActive   MatchStatus = "active"
	StatusFinished MatchStatus = "finished"
)

var statusTransitions = map[MatchStatus]MatchStatus{StatusDraft: StatusActive, StatusActive: StatusFinished}

func (st MatchStatus) Valid() bool {
	switch st {
	case StatusDraft, StatusActive, StatusFinished:
		return true
	}
	return false
}

func validShooter(sh repository.Shooter) bool {
	return strings.TrimSpace(sh.LastName) != "" && strings.TrimSpace(sh.FirstName) != "" &&
		validShooterNames(sh.FirstName, sh.LastName) &&
		sh.Country != "" && sh.Club != "" &&
		len(sh.LastName) <= 128 && len(sh.FirstName) <= 128 && len(sh.WAID) <= 32 && len(sh.Club) <= 128
}

const (
	defaultShooterSearchLimit = 20
	maxShooterSearchLimit     = 100
)

func (s *Service) ListShooters(ctx context.Context, query string, includeArchived bool, limit int) ([]repository.Shooter, error) {
	query = strings.TrimSpace(query)
	if limit < 0 || limit > maxShooterSearchLimit {
		return nil, ErrInvalid
	}
	if limit == 0 && query != "" {
		limit = defaultShooterSearchLimit
	}
	return s.repo.ListShooters(ctx, query, includeArchived, limit)
}

type ShooterUsage struct {
	EntryCount int
	Matches    []repository.Match
}

func (s *Service) ShooterEntryCounts(ctx context.Context) (map[int64]int, error) {
	return s.repo.ShooterEntryCounts(ctx)
}

func (s *Service) ShooterUsage(ctx context.Context, id int64) (repository.Shooter, ShooterUsage, error) {
	sh, err := s.repo.GetShooter(ctx, id)
	if err != nil {
		return sh, ShooterUsage{}, mapRepoErr(err)
	}
	matches, err := s.repo.ShooterMatches(ctx, id)
	if err != nil {
		return sh, ShooterUsage{}, mapRepoErr(err)
	}
	return sh, ShooterUsage{EntryCount: len(matches), Matches: matches}, nil
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

func (s *Service) ArchiveShooter(ctx context.Context, id int64) (repository.Shooter, error) {
	return s.setShooterArchived(ctx, id, true, "shooter.archive")
}

func (s *Service) UnarchiveShooter(ctx context.Context, id int64) (repository.Shooter, error) {
	return s.setShooterArchived(ctx, id, false, "shooter.unarchive")
}

func (s *Service) setShooterArchived(ctx context.Context, id int64, archived bool, action string) (repository.Shooter, error) {
	sh, err := s.repo.GetShooter(ctx, id)
	if err != nil {
		return sh, mapRepoErr(err)
	}
	if sh.Archived == archived {
		return sh, nil
	}
	if err := s.repo.SetShooterArchived(ctx, id, archived); err != nil {
		return repository.Shooter{}, mapRepoErr(err)
	}
	sh.Archived = archived
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: action, Target: idTarget("shooter", id)}, Live: LiveAll})
	return sh, nil
}

func (s *Service) DeleteShooter(ctx context.Context, id int64) error {
	if err := s.repo.DeleteShooter(ctx, id); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return ErrShooterHasEntries
		}
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "shooter.delete", Target: idTarget("shooter", id)}, Live: LiveAll})
	return nil
}

func (s *Service) UpdateMatch(ctx context.Context, c repository.Match) error {
	if strings.TrimSpace(c.Name) == "" || len(c.Name) > 128 || !validDate(c.Date) {
		return ErrInvalid
	}
	if _, err := rulebook.CourseByKey(c.Course); err != nil {
		return ErrInvalid
	}
	current, err := s.repo.GetMatch(ctx, c.ID)
	if err != nil {
		return mapRepoErr(err)
	}
	if MatchStatus(current.Status) != StatusDraft {
		return ErrConflict
	}
	if err := s.checkMatchDate(ctx, current.CompetitionID, c.Date, true); err != nil {
		return err
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
	if err := s.repo.UpdateMatch(ctx, c); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "match.update", Target: idTarget("match", c.ID), Details: c.Name}, MatchID: c.ID, Live: LiveMatch, Reemit: &repository.LockedStageFilter{MatchID: c.ID}})
	return nil
}

func (s *Service) SetMatchStatus(ctx context.Context, id int64, status MatchStatus) error {
	if !status.Valid() {
		return ErrInvalid
	}
	current, err := s.repo.GetMatch(ctx, id)
	if err != nil {
		return mapRepoErr(err)
	}
	if statusTransitions[MatchStatus(current.Status)] != status {
		return ErrConflict
	}
	return s.changeStatus(ctx, id, MatchStatus(current.Status), status, nil)
}

func (s *Service) changeStatus(ctx context.Context, id int64, from, status MatchStatus, actor *Actor) error {
	if err := s.repo.SetMatchStatus(ctx, id, string(from), string(status)); err != nil {
		return mapRepoErr(err)
	}
	record := Change{Audit: AuditEvent{Action: "match.status", Target: idTarget("match", id), Details: string(status)}, Actor: actor, MatchID: id, Live: LiveMatch, Screens: true}
	if status != StatusFinished {
		s.changes.Record(ctx, record)
		return nil
	}
	s.changes.Record(ctx, Change{Audit: record.Audit, Actor: actor})
	if err := s.repo.RevokeMatchPairings(ctx, id); err != nil {
		return mapRepoErr(err)
	}
	if s.replicator != nil {
		s.replicator.Request()
	}
	s.changes.Record(ctx, Change{MatchID: id, Live: LiveMatch, Screens: true})
	return nil
}

func (s *Service) DeleteMatch(ctx context.Context, id int64) error {
	if err := s.repo.DeleteMatch(ctx, id); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "match.delete", Target: idTarget("match", id)}, MatchID: id, Live: LiveMatch, Screens: true})
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

func (s *Service) ListEntries(ctx context.Context, matchID int64) ([]EntryScoring, error) {
	if _, err := s.repo.GetMatch(ctx, matchID); err != nil {
		return nil, mapRepoErr(err)
	}
	entries, err := s.repo.ListEntries(ctx, matchID)
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
				Score:        c.Score(),
				ShotsEntered: c.X + c.Ten + c.Nine + c.Eight + c.Seven + c.Zero + c.Miss,
			})
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *Service) requireDraft(ctx context.Context, matchID int64) (repository.Match, error) {
	comp, err := s.repo.GetMatch(ctx, matchID)
	if err != nil {
		return comp, mapRepoErr(err)
	}
	if MatchStatus(comp.Status) != StatusDraft {
		return comp, ErrConflict
	}
	return comp, nil
}

func (s *Service) requireWithdrawalAllowed(ctx context.Context, matchID int64) error {
	comp, err := s.repo.GetMatch(ctx, matchID)
	if err != nil {
		return mapRepoErr(err)
	}
	switch MatchStatus(comp.Status) {
	case StatusDraft:
		return nil
	case StatusFinished:
		return ErrConflict
	}
	entries, err := s.repo.ListEntries(ctx, matchID)
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

func (s *Service) StartNextStage(ctx context.Context, matchID int64) (int, error) {
	comp, err := s.repo.GetMatch(ctx, matchID)
	if err != nil {
		return 0, mapRepoErr(err)
	}
	if MatchStatus(comp.Status) != StatusActive {
		return 0, ErrConflict
	}
	last, err := lastStage(comp.Course)
	if err != nil {
		return 0, err
	}
	if comp.CurrentStage >= last {
		return 0, ErrConflict
	}
	entries, err := s.repo.ListEntries(ctx, matchID)
	if err != nil {
		return 0, err
	}
	if !stageAccepted(entries, comp.CurrentStage) {
		return 0, ErrStageGateNotMet
	}
	if err := s.repo.AdvanceStage(ctx, matchID, comp.CurrentStage); err != nil {
		return 0, mapRepoErr(err)
	}
	next := comp.CurrentStage + 1
	s.changes.Record(ctx, Change{
		Audit:   AuditEvent{Action: "match.next_stage", Target: idTarget("match", matchID), Details: "stage " + strconv.Itoa(next)},
		MatchID: matchID,
		Live:    LiveMatch,
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

func (s *Service) finishIfComplete(ctx context.Context, matchID int64, actor *Actor) {
	comp, err := s.repo.GetMatch(ctx, matchID)
	if err != nil || MatchStatus(comp.Status) != StatusActive {
		return
	}
	last, err := lastStage(comp.Course)
	if err != nil || comp.CurrentStage != last {
		return
	}
	entries, err := s.repo.ListEntries(ctx, matchID)
	if err != nil || !stageAccepted(entries, last) {
		return
	}
	_ = s.changeStatus(ctx, matchID, StatusActive, StatusFinished, actor)
}

func (s *Service) UpdateEntry(ctx context.Context, matchID, entryID int64, category string, lane int) (repository.Entry, error) {
	if lane < 1 {
		return repository.Entry{}, ErrInvalid
	}
	current, err := s.repo.GetEntry(ctx, matchID, entryID)
	if err != nil {
		return repository.Entry{}, mapRepoErr(err)
	}
	comp, err := s.repo.GetMatch(ctx, matchID)
	if err != nil {
		return repository.Entry{}, mapRepoErr(err)
	}
	if category != current.Category && rulebook.ValidateCategory(comp.Course, category) != nil {
		return repository.Entry{}, ErrInvalid
	}
	switch MatchStatus(comp.Status) {
	case StatusDraft:
	case StatusActive:
		if lane != current.Lane {
			return repository.Entry{}, ErrConflict
		}
	default:
		return repository.Entry{}, ErrConflict
	}
	if err := s.repo.UpdateEntry(ctx, matchID, entryID, category, lane); err != nil {
		return repository.Entry{}, mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "entry.update", Target: entryTarget(matchID, entryID)}, MatchID: matchID, Live: LiveMatch, Reemit: &repository.LockedStageFilter{MatchID: matchID, EntryID: entryID}})
	e, err := s.repo.GetEntry(ctx, matchID, entryID)
	return e, mapRepoErr(err)
}

func (s *Service) DeleteEntry(ctx context.Context, matchID, entryID int64) error {
	comp, err := s.repo.GetMatch(ctx, matchID)
	if err != nil {
		return mapRepoErr(err)
	}
	if MatchStatus(comp.Status) != StatusDraft {
		return ErrConflict
	}
	if err := s.repo.DeleteEntry(ctx, matchID, entryID); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "entry.delete", Target: entryTarget(matchID, entryID)}, MatchID: matchID, Live: LiveMatch})
	return nil
}

func (s *Service) DefaultMatchID(ctx context.Context) (int64, bool, error) {
	id, err := s.repo.DefaultMatchID(ctx)
	if errors.Is(err, repository.ErrNotFound) {
		return 0, false, nil
	}
	return id, err == nil, err
}
