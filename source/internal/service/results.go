package service

import (
	"context"

	"github.com/sollie/matchmark/internal/report"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
)

type ResultsCompetition struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Date        string `json:"date"`
	Course      string `json:"course"`
	CourseLabel string `json:"course_label"`
}

type ResultsStage struct {
	Stage int  `json:"stage"`
	Score *int `json:"score,omitempty"`
}

type ResultsRow struct {
	Rank      int            `json:"rank"`
	LastName  *string        `json:"last_name,omitempty"`
	FirstName *string        `json:"first_name,omitempty"`
	WAID      *string        `json:"waid,omitempty"`
	Club      *string        `json:"club,omitempty"`
	Country   *string        `json:"country,omitempty"`
	Stages    []ResultsStage `json:"stages"`
	Total     int            `json:"total"`
	XCount    *int           `json:"x_count,omitempty"`
}

type ResultsCategory struct {
	Category string       `json:"category"`
	Rows     []ResultsRow `json:"rows"`
}

type ResultsNewShooter struct {
	LastName  string  `json:"last_name"`
	FirstName string  `json:"first_name"`
	Category  string  `json:"category"`
	Club      *string `json:"club,omitempty"`
	Country   *string `json:"country,omitempty"`
	Total     int     `json:"total"`
	XCount    *int    `json:"x_count,omitempty"`
}

type ResultsView struct {
	Competition ResultsCompetition  `json:"competition"`
	Stages      []int               `json:"stages"`
	Categories  []ResultsCategory   `json:"categories"`
	NewShooters []ResultsNewShooter `json:"new_shooters"`
}

var showAll = repository.Visibility{Names: true, Clubs: true, Countries: true, StageScores: true, RingCounts: true}

func (s *Service) PublicResults(ctx context.Context, competitionID int64) (ResultsView, error) {
	comp, err := s.repo.GetCompetition(ctx, competitionID)
	if err != nil {
		return ResultsView{}, mapRepoErr(err)
	}
	if CompetitionStatus(comp.Status) != StatusFinished {
		return ResultsView{}, ErrNotFound
	}
	vis, err := s.repo.GetVisibility(ctx, competitionID)
	if err != nil {
		return ResultsView{}, mapRepoErr(err)
	}
	return s.results(ctx, comp, vis)
}

func (s *Service) AdminResults(ctx context.Context, competitionID int64) (ResultsView, error) {
	comp, err := s.repo.GetCompetition(ctx, competitionID)
	if err != nil {
		return ResultsView{}, mapRepoErr(err)
	}
	return s.results(ctx, comp, showAll)
}

func (s *Service) resultsProjection(ctx context.Context, comp repository.Competition, vis repository.Visibility) (report.Results, error) {
	if _, err := rulebook.CourseByKey(comp.Course); err != nil {
		return report.Results{}, err
	}
	entries, err := s.repo.ReportEntries(ctx, comp.ID)
	if err != nil {
		return report.Results{}, err
	}
	active := make([]repository.ReportEntry, 0, len(entries))
	for _, e := range entries {
		if !e.Withdrawn {
			active = append(active, e)
		}
	}
	return report.Build(comp, active).Redact(vis), nil
}

func (s *Service) results(ctx context.Context, comp repository.Competition, vis repository.Visibility) (ResultsView, error) {
	res, err := s.resultsProjection(ctx, comp, vis)
	if err != nil {
		return ResultsView{}, err
	}
	return resultsView(res), nil
}

func resultsView(res report.Results) ResultsView {
	vis := *res.Visibility
	comp := res.Competition
	out := ResultsView{
		Competition: ResultsCompetition{ID: comp.ID, Name: comp.Name, Date: comp.Date, Course: comp.Course, CourseLabel: res.CourseLabel},
		Stages:      res.Stages,
		Categories:  []ResultsCategory{},
		NewShooters: []ResultsNewShooter{},
	}
	for _, c := range res.Categories {
		cat := ResultsCategory{Category: c.Name, Rows: make([]ResultsRow, len(c.Rows))}
		for i, r := range c.Rows {
			cat.Rows[i] = resultsRow(r, vis)
		}
		for _, r := range c.New {
			out.NewShooters = append(out.NewShooters, resultsNewShooter(r, c.Name, vis))
		}
		out.Categories = append(out.Categories, cat)
	}
	return out
}

func resultsRow(r report.Row, vis repository.Visibility) ResultsRow {
	sh := r.Shooter
	row := ResultsRow{Rank: r.Rank, Stages: make([]ResultsStage, len(r.Stages)), Total: r.Total}
	for i, st := range r.Stages {
		row.Stages[i] = ResultsStage{Stage: st.Number}
		if vis.StageScores && st.Accepted {
			score := st.Score
			row.Stages[i].Score = &score
		}
	}
	if vis.Names {
		row.LastName, row.FirstName = &sh.LastName, &sh.FirstName
		if sh.WAID != "" {
			row.WAID = &sh.WAID
		}
	}
	if vis.Clubs {
		row.Club = &sh.Club
	}
	if vis.Countries {
		row.Country = &sh.Country
	}
	if vis.RingCounts {
		x := r.XCount
		row.XCount = &x
	}
	return row
}

func resultsNewShooter(r report.Row, category string, vis repository.Visibility) ResultsNewShooter {
	sh := r.Shooter
	n := ResultsNewShooter{LastName: sh.LastName, FirstName: sh.FirstName, Category: category, Total: r.Total}
	if vis.Clubs {
		n.Club = &sh.Club
	}
	if vis.Countries {
		n.Country = &sh.Country
	}
	if vis.RingCounts {
		x := r.XCount
		n.XCount = &x
	}
	return n
}
