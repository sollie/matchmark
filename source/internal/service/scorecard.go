package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/sollie/matchmark/internal/report"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
)

type ScorecardScorer struct {
	Stage  int    `json:"stage"`
	Scorer string `json:"scorer"`
}

type ScorecardSummary struct {
	Shooter string            `json:"shooter"`
	Lane    int               `json:"lane"`
	Scorers []ScorecardScorer `json:"scorers"`
}

func (s *Service) loadScorecard(ctx context.Context, competitionID int64, lane int) (report.ScorecardData, error) {
	data, err := s.loadReport(ctx, competitionID)
	if err != nil {
		return report.ScorecardData{}, err
	}
	if data.Competition.Status != string(StatusFinished) {
		return report.ScorecardData{}, ErrNotFound
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
		return report.ScorecardData{}, ErrNotFound
	}
	if card.Course, err = rulebook.CourseByKey(data.Competition.Course); err != nil {
		return report.ScorecardData{}, err
	}
	return card, nil
}

func (s *Service) ScorecardSummary(ctx context.Context, competitionID int64, lane int) (ScorecardSummary, error) {
	card, err := s.loadScorecard(ctx, competitionID, lane)
	if err != nil {
		return ScorecardSummary{}, err
	}
	shooter := card.Entry.Shooter
	summary := ScorecardSummary{Shooter: shooter.FirstName + " " + shooter.LastName, Lane: lane, Scorers: []ScorecardScorer{}}
	for _, st := range card.Entry.Stages {
		if st.Status == repository.StatusAccepted {
			summary.Scorers = append(summary.Scorers, ScorecardScorer{Stage: st.Number, Scorer: report.ScorerLabel(st)})
		}
	}
	return summary, nil
}

func (s *Service) ScorecardPNG(ctx context.Context, competitionID int64, lane int) (Download, error) {
	card, err := s.loadScorecard(ctx, competitionID, lane)
	if err != nil {
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
	base := fmt.Sprintf("%s-lane-%d", downloadBaseName(card.Competition.Name, card.Competition.Date), lane)
	return Download{BaseName: base, Data: png}, nil
}
