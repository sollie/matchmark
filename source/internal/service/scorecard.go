package service

import (
	"context"
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
	comp, entries, err := s.loadReport(ctx, competitionID)
	if err != nil {
		return report.ScorecardData{}, err
	}
	if comp.Status != string(StatusFinished) {
		return report.ScorecardData{}, ErrNotFound
	}
	card := report.ScorecardData{Competition: comp}
	found := false
	for _, e := range entries {
		if e.Lane == lane && !e.Withdrawn {
			card.Entry = e
			found = true
			break
		}
	}
	if !found {
		return report.ScorecardData{}, ErrNotFound
	}
	if card.Course, err = rulebook.CourseByKey(comp.Course); err != nil {
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
	var logo ClubLogo
	if card.ClubName, logo, err = s.clubBranding(ctx); err != nil {
		return Download{}, err
	}
	card.Logo = logo.Data
	png, err := report.ScorecardPNG(card)
	if err != nil {
		return Download{}, err
	}
	base := fmt.Sprintf("%s-lane-%d", downloadBaseName(card.Competition.Name, card.Competition.Date), lane)
	return Download{BaseName: base, Data: png}, nil
}
