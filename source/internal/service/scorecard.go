package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/sollie/matchmark/internal/report"
	"github.com/sollie/matchmark/internal/rulebook"
)

func (s *Service) ScorecardPNG(ctx context.Context, competitionID int64, lane int) (Download, error) {
	data, err := s.loadReport(ctx, competitionID)
	if err != nil {
		return Download{}, err
	}
	if data.Competition.Status != string(StatusFinished) {
		return Download{}, ErrNotFound
	}
	card := report.ScorecardData{Competition: data.Competition}
	found := false
	for _, e := range data.Entries {
		if e.Lane == lane && !e.Withdrawn {
			card.Entry = e
			found = true
			break
		}
	}
	if !found {
		return Download{}, ErrNotFound
	}
	if card.Course, err = rulebook.CourseByKey(data.Competition.Course); err != nil {
		return Download{}, err
	}
	club, err := s.GetClub(ctx)
	if err != nil {
		return Download{}, err
	}
	card.ClubName = club.Name
	if club.HasLogo {
		logo, err := s.GetClubLogo(ctx)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return Download{}, err
		}
		card.Logo = logo.Data
	}
	png, err := report.ScorecardPNG(card)
	if err != nil {
		return Download{}, err
	}
	base := fmt.Sprintf("%s-lane-%d", downloadBaseName(data.Competition.Name, data.Competition.Date), lane)
	return Download{BaseName: base, Data: png}, nil
}
