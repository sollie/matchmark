package service

import (
	"context"
	"strings"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
)

var categories = []string{"Revolver", "Pistol", "Revolver OS", "Pistol OS", "Rifle"}

func (s *Service) CreateShooter(ctx context.Context, sh repository.Shooter) (int64, error) {
	if !validShooter(sh) {
		return 0, ErrInvalid
	}
	id, err := s.repo.CreateShooter(ctx, sh)
	return id, mapRepoErr(err)
}

func (s *Service) CreateCompetition(ctx context.Context, c repository.Competition) (int64, error) {
	if strings.TrimSpace(c.Name) == "" || len(c.Name) > 128 || c.Date == "" {
		return 0, ErrInvalid
	}
	if _, err := rulebook.CourseByKey(c.Course); err != nil {
		return 0, ErrInvalid
	}
	id, err := s.repo.CreateCompetition(ctx, c)
	return id, mapRepoErr(err)
}

func (s *Service) AddEntry(ctx context.Context, e repository.Entry) (int64, error) {
	if e.Lane < 1 || !contains(categories, e.Category) {
		return 0, ErrInvalid
	}
	comp, err := s.requireEntryChangeAllowed(ctx, e.CompetitionID)
	if err != nil {
		return 0, err
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
	return id, mapRepoErr(err)
}

func (s *Service) WithdrawEntry(ctx context.Context, competitionID, entryID int64) error {
	if _, err := s.requireEntryChangeAllowed(ctx, competitionID); err != nil {
		return err
	}
	return mapRepoErr(s.repo.WithdrawEntry(ctx, competitionID, entryID))
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
