package service

import (
	"context"
	"sort"

	"github.com/sollie/matchmark/internal/repository"
)

type RingCountsView struct {
	X     int `json:"x"`
	Ten   int `json:"10"`
	Nine  int `json:"9"`
	Eight int `json:"8"`
	Seven int `json:"7"`
	Zero  int `json:"0"`
	Miss  int `json:"miss"`
}

type StageScoreView struct {
	Stage  int             `json:"stage"`
	Score  int             `json:"score"`
	Locked bool            `json:"locked"`
	Counts *RingCountsView `json:"ring_counts,omitempty"`
}

type StandingView struct {
	Lane       int              `json:"lane"`
	Category   string           `json:"category"`
	Withdrawn  bool             `json:"withdrawn"`
	Total      int              `json:"total"`
	LastName   *string          `json:"last_name,omitempty"`
	FirstName  *string          `json:"first_name,omitempty"`
	Club       *string          `json:"club,omitempty"`
	Country    *string          `json:"country,omitempty"`
	XCount     *int             `json:"x_count,omitempty"`
	RingCounts *RingCountsView  `json:"ring_counts,omitempty"`
	Stages     []StageScoreView `json:"stages,omitempty"`
}

type StageProgress struct {
	Stage   int `json:"stage"`
	Entries int `json:"entries"`
	Locked  int `json:"locked"`
}

func (s *Service) GetVisibility(ctx context.Context, competitionID int64) (repository.Visibility, error) {
	v, err := s.repo.GetVisibility(ctx, competitionID)
	return v, mapRepoErr(err)
}

func (s *Service) SetVisibility(ctx context.Context, competitionID int64, v repository.Visibility) error {
	return mapRepoErr(s.repo.SetVisibility(ctx, competitionID, v))
}

func (s *Service) ListPublicCompetitions(ctx context.Context) ([]repository.Competition, error) {
	return s.repo.ListPublicCompetitions(ctx)
}

func (s *Service) publicCompetition(ctx context.Context, id int64) error {
	c, err := s.repo.GetCompetition(ctx, id)
	if err != nil {
		return mapRepoErr(err)
	}
	if c.Status == "draft" {
		return ErrNotFound
	}
	return nil
}

func stagePoints(c repository.RingCounts) int {
	return (c.X+c.Ten)*10 + c.Nine*9 + c.Eight*8 + c.Seven*7
}

func ringCountsView(c repository.RingCounts) RingCountsView {
	return RingCountsView{X: c.X, Ten: c.Ten, Nine: c.Nine, Eight: c.Eight, Seven: c.Seven, Zero: c.Zero, Miss: c.Miss}
}

func (s *Service) Standings(ctx context.Context, competitionID int64) ([]StandingView, error) {
	if err := s.publicCompetition(ctx, competitionID); err != nil {
		return nil, err
	}
	vis, err := s.repo.GetVisibility(ctx, competitionID)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	entries, err := s.repo.ListSpectatorEntries(ctx, competitionID)
	if err != nil {
		return nil, err
	}
	type ranked struct {
		view StandingView
		x    int
	}
	rows := make([]ranked, 0, len(entries))
	for _, e := range entries {
		var total repository.RingCounts
		v := StandingView{Lane: e.Lane, Category: e.Category, Withdrawn: e.Withdrawn}
		for _, st := range e.Stages {
			v.Total += stagePoints(st.Counts)
			total.X += st.Counts.X
			total.Ten += st.Counts.Ten
			total.Nine += st.Counts.Nine
			total.Eight += st.Counts.Eight
			total.Seven += st.Counts.Seven
			total.Zero += st.Counts.Zero
			total.Miss += st.Counts.Miss
			if vis.StageScores {
				sv := StageScoreView{Stage: st.Number, Score: stagePoints(st.Counts), Locked: st.Locked}
				if vis.RingCounts {
					c := ringCountsView(st.Counts)
					sv.Counts = &c
				}
				v.Stages = append(v.Stages, sv)
			}
		}
		if vis.Names {
			v.LastName, v.FirstName = &e.LastName, &e.FirstName
		}
		if vis.Clubs {
			v.Club = &e.Club
		}
		if vis.Countries {
			v.Country = &e.Country
		}
		if vis.RingCounts {
			c := ringCountsView(total)
			x := total.X
			v.RingCounts, v.XCount = &c, &x
		}
		rows = append(rows, ranked{view: v, x: total.X})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].view.Total != rows[j].view.Total {
			return rows[i].view.Total > rows[j].view.Total
		}
		return rows[i].x > rows[j].x
	})
	out := make([]StandingView, len(rows))
	for i, r := range rows {
		out[i] = r.view
	}
	return out, nil
}

func (s *Service) Progress(ctx context.Context, competitionID int64) ([]StageProgress, error) {
	if err := s.publicCompetition(ctx, competitionID); err != nil {
		return nil, err
	}
	entries, err := s.repo.ListSpectatorEntries(ctx, competitionID)
	if err != nil {
		return nil, err
	}
	byStage := map[int]*StageProgress{}
	for _, e := range entries {
		for _, st := range e.Stages {
			p, ok := byStage[st.Number]
			if !ok {
				p = &StageProgress{Stage: st.Number}
				byStage[st.Number] = p
			}
			p.Entries++
			if st.Locked {
				p.Locked++
			}
		}
	}
	out := make([]StageProgress, 0, len(byStage))
	for _, p := range byStage {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Stage < out[j].Stage })
	return out, nil
}
