package service

import (
	"context"
	"sort"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
)

type ResultsMatch struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Date        string `json:"date"`
	Course      string `json:"course"`
	CourseLabel string `json:"course_label"`
}

type ResultsSegment struct {
	Segment    int             `json:"segment"`
	Score      *int            `json:"score,omitempty"`
	RingCounts *RingCountsView `json:"ring_counts,omitempty"`
}

type ResultsRow struct {
	Rank      int              `json:"rank"`
	LastName  *string          `json:"last_name,omitempty"`
	FirstName *string          `json:"first_name,omitempty"`
	WAID      *string          `json:"waid,omitempty"`
	Club      *string          `json:"club,omitempty"`
	Country   *string          `json:"country,omitempty"`
	Segments  []ResultsSegment `json:"segments"`
	Total     int              `json:"total"`
	XCount    *int             `json:"x_count,omitempty"`
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
	Match       ResultsMatch        `json:"match"`
	Segments    []int               `json:"segments"`
	Categories  []ResultsCategory   `json:"categories"`
	NewShooters []ResultsNewShooter `json:"new_shooters"`
}

var showAll = repository.Visibility{Names: true, Clubs: true, Countries: true, SegmentScores: true, RingCounts: true}

func (s *Service) PublicResults(ctx context.Context, matchID int64) (ResultsView, error) {
	comp, err := s.repo.GetMatch(ctx, matchID)
	if err != nil {
		return ResultsView{}, mapRepoErr(err)
	}
	if MatchStatus(comp.Status) != StatusFinished {
		return ResultsView{}, ErrNotFound
	}
	vis, err := s.repo.GetVisibility(ctx, matchID)
	if err != nil {
		return ResultsView{}, mapRepoErr(err)
	}
	return s.results(ctx, comp, vis)
}

func (s *Service) AdminResults(ctx context.Context, matchID int64) (ResultsView, error) {
	comp, err := s.repo.GetMatch(ctx, matchID)
	if err != nil {
		return ResultsView{}, mapRepoErr(err)
	}
	return s.results(ctx, comp, showAll)
}

type rankedEntry struct {
	entry repository.ReportEntry
	total int
	x     int
	view  []ResultsSegment
}

func (s *Service) results(ctx context.Context, comp repository.Match, vis repository.Visibility) (ResultsView, error) {
	course, err := rulebook.CourseByKey(comp.Course)
	if err != nil {
		return ResultsView{}, err
	}
	entries, err := s.repo.ReportEntries(ctx, comp.ID)
	if err != nil {
		return ResultsView{}, err
	}
	out := ResultsView{
		Match:       ResultsMatch{ID: comp.ID, Name: comp.Name, Date: comp.Date, Course: comp.Course, CourseLabel: course.Label},
		Segments:    make([]int, len(course.Segments)),
		Categories:  []ResultsCategory{},
		NewShooters: []ResultsNewShooter{},
	}
	for i := range course.Segments {
		out.Segments[i] = course.Segments[i].Number
	}
	byCategory := map[string][]rankedEntry{}
	for _, e := range entries {
		if e.Withdrawn {
			continue
		}
		r := rankedEntry{entry: e}
		for _, st := range e.Segments {
			rs := ResultsSegment{Segment: st.Number}
			if st.Status == repository.StatusAccepted {
				score := st.Counts.Score()
				r.total += score
				r.x += st.Counts.X
				if vis.SegmentScores {
					rs.Score = &score
				}
				if vis.RingCounts {
					counts := ringCountsView(st.Counts)
					rs.RingCounts = &counts
				}
			}
			r.view = append(r.view, rs)
		}
		byCategory[e.Category] = append(byCategory[e.Category], r)
	}
	for _, category := range rulebook.Categories() {
		ranked := byCategory[category]
		if len(ranked) == 0 {
			continue
		}
		sort.SliceStable(ranked, func(i, j int) bool {
			if ranked[i].total != ranked[j].total {
				return ranked[i].total > ranked[j].total
			}
			return ranked[i].x > ranked[j].x
		})
		cat := ResultsCategory{Category: category, Rows: make([]ResultsRow, len(ranked))}
		rank := 0
		for i, r := range ranked {
			if i == 0 || r.total != ranked[i-1].total || r.x != ranked[i-1].x {
				rank = i + 1
			}
			cat.Rows[i] = resultsRow(r, rank, vis)
			if vis.Names && r.entry.Shooter.WAID == "" {
				out.NewShooters = append(out.NewShooters, resultsNewShooter(r, category, vis))
			}
		}
		out.Categories = append(out.Categories, cat)
	}
	return out, nil
}

func resultsRow(r rankedEntry, rank int, vis repository.Visibility) ResultsRow {
	sh := r.entry.Shooter
	row := ResultsRow{Rank: rank, Segments: r.view, Total: r.total}
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
		x := r.x
		row.XCount = &x
	}
	return row
}

func resultsNewShooter(r rankedEntry, category string, vis repository.Visibility) ResultsNewShooter {
	sh := r.entry.Shooter
	n := ResultsNewShooter{LastName: sh.LastName, FirstName: sh.FirstName, Category: category, Total: r.total}
	if vis.Clubs {
		n.Club = &sh.Club
	}
	if vis.Countries {
		n.Country = &sh.Country
	}
	if vis.RingCounts {
		x := r.x
		n.XCount = &x
	}
	return n
}
