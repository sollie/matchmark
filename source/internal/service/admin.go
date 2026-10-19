package service

import (
	"context"
	"errors"
	"strings"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
)

var ErrNotBetweenStages = errors.New("entries can only change between stages")

var statusTransitions = map[string]string{"draft": "active", "active": "finished"}

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
	return mapRepoErr(s.repo.UpdateShooter(ctx, sh))
}

func (s *Service) DeleteShooter(ctx context.Context, id int64) error {
	return mapRepoErr(s.repo.DeleteShooter(ctx, id))
}

func (s *Service) ListCompetitions(ctx context.Context) ([]repository.Competition, error) {
	return s.repo.ListCompetitions(ctx)
}

func (s *Service) GetCompetition(ctx context.Context, id int64) (repository.Competition, error) {
	c, err := s.repo.GetCompetition(ctx, id)
	return c, mapRepoErr(err)
}

func (s *Service) UpdateCompetition(ctx context.Context, c repository.Competition) error {
	if strings.TrimSpace(c.Name) == "" || len(c.Name) > 128 || c.Date == "" {
		return ErrInvalid
	}
	if _, err := rulebook.CourseByKey(c.Course); err != nil {
		return ErrInvalid
	}
	current, err := s.repo.GetCompetition(ctx, c.ID)
	if err != nil {
		return mapRepoErr(err)
	}
	if current.Status != "draft" {
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
	return mapRepoErr(s.repo.UpdateCompetition(ctx, c))
}

func (s *Service) SetCompetitionStatus(ctx context.Context, id int64, status string) error {
	if status != "draft" && status != "active" && status != "finished" {
		return ErrInvalid
	}
	current, err := s.repo.GetCompetition(ctx, id)
	if err != nil {
		return mapRepoErr(err)
	}
	if statusTransitions[current.Status] != status {
		return ErrConflict
	}
	return mapRepoErr(s.repo.SetCompetitionStatus(ctx, id, status))
}

func (s *Service) DeleteCompetition(ctx context.Context, id int64) error {
	return mapRepoErr(s.repo.DeleteCompetition(ctx, id))
}

func (s *Service) ListEntries(ctx context.Context, competitionID int64) ([]repository.EntryWithStages, error) {
	if _, err := s.repo.GetCompetition(ctx, competitionID); err != nil {
		return nil, mapRepoErr(err)
	}
	entries, err := s.repo.ListEntries(ctx, competitionID)
	return entries, mapRepoErr(err)
}

func (s *Service) requireEntryChangeAllowed(ctx context.Context, competitionID int64) (repository.Competition, error) {
	comp, err := s.repo.GetCompetition(ctx, competitionID)
	if err != nil {
		return comp, mapRepoErr(err)
	}
	if comp.Status == "finished" {
		return comp, ErrConflict
	}
	busy, err := s.repo.StageInProgress(ctx, competitionID)
	if err != nil {
		return comp, err
	}
	if busy {
		return comp, ErrNotBetweenStages
	}
	return comp, nil
}

func (s *Service) UpdateEntry(ctx context.Context, competitionID, entryID int64, category string, lane int) (repository.Entry, error) {
	if lane < 1 || !contains(categories, category) {
		return repository.Entry{}, ErrInvalid
	}
	if _, err := s.repo.GetEntry(ctx, competitionID, entryID); err != nil {
		return repository.Entry{}, mapRepoErr(err)
	}
	if _, err := s.requireEntryChangeAllowed(ctx, competitionID); err != nil {
		return repository.Entry{}, err
	}
	if err := s.repo.UpdateEntry(ctx, competitionID, entryID, category, lane); err != nil {
		return repository.Entry{}, mapRepoErr(err)
	}
	e, err := s.repo.GetEntry(ctx, competitionID, entryID)
	return e, mapRepoErr(err)
}

func (s *Service) DeleteEntry(ctx context.Context, competitionID, entryID int64) error {
	comp, err := s.repo.GetCompetition(ctx, competitionID)
	if err != nil {
		return mapRepoErr(err)
	}
	if comp.Status != "draft" {
		return ErrConflict
	}
	return mapRepoErr(s.repo.DeleteEntry(ctx, competitionID, entryID))
}
