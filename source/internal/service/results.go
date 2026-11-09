package service

import (
	"context"
	"sort"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
	"github.com/sollie/matchmark/internal/wa1500"
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
	Rank      int                    `json:"rank"`
	LastName  *string                `json:"last_name,omitempty"`
	FirstName *string                `json:"first_name,omitempty"`
	WAID      *string                `json:"waid,omitempty"`
	Club      *string                `json:"club,omitempty"`
	Country   *string                `json:"country,omitempty"`
	Class     *wa1500.Classification `json:"classification,omitempty"`
	CompeteAs *wa1500.Classification `json:"class,omitempty"`
	Segments  []ResultsSegment       `json:"segments"`
	Total     int                    `json:"total"`
	XCount    *int                   `json:"x_count,omitempty"`
}

type ResultsDNFRow struct {
	LastName  *string          `json:"last_name,omitempty"`
	FirstName *string          `json:"first_name,omitempty"`
	WAID      *string          `json:"waid,omitempty"`
	Club      *string          `json:"club,omitempty"`
	Country   *string          `json:"country,omitempty"`
	Segments  []ResultsSegment `json:"segments"`
}

type ResultsGroup struct {
	Class wa1500.Classification `json:"class"`
	Rows  []ResultsRow          `json:"rows"`
}

type ResultsCategory struct {
	Category     string          `json:"category"`
	Rows         []ResultsRow    `json:"rows"`
	Groups       []ResultsGroup  `json:"groups"`
	DidNotFinish []ResultsDNFRow `json:"did_not_finish"`
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
	dnfByCategory := map[string][]rankedEntry{}
	for _, e := range entries {
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
		if e.Withdrawn {
			dnfByCategory[e.Category] = append(dnfByCategory[e.Category], r)
			continue
		}
		byCategory[e.Category] = append(byCategory[e.Category], r)
	}
	for _, category := range rulebook.Categories() {
		ranked := byCategory[category]
		dnf := dnfByCategory[category]
		if len(ranked) == 0 && len(dnf) == 0 {
			continue
		}
		sortRanked(ranked)
		cat := ResultsCategory{Category: category, Rows: []ResultsRow{}, Groups: []ResultsGroup{}, DidNotFinish: make([]ResultsDNFRow, len(dnf))}
		if groups, ok := classGroups(comp.Classified, category, ranked); ok {
			for _, class := range classOrder {
				members := groups[class]
				if len(members) == 0 {
					continue
				}
				cat.Groups = append(cat.Groups, ResultsGroup{Class: class, Rows: rankRows(members, vis)})
			}
		} else {
			cat.Rows = rankRows(ranked, vis)
		}
		for _, r := range ranked {
			if vis.Names && r.entry.Shooter.WAID == "" {
				out.NewShooters = append(out.NewShooters, resultsNewShooter(r, category, vis))
			}
		}
		for i, r := range dnf {
			cat.DidNotFinish[i] = resultsDNFRow(r, vis)
		}
		out.Categories = append(out.Categories, cat)
	}
	return out, nil
}

var classOrder = []wa1500.Classification{wa1500.HighMaster, wa1500.Master, wa1500.Expert, wa1500.Sharpshooter, wa1500.Marksman, wa1500.Unclassified}

func sortRanked(ranked []rankedEntry) {
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].total != ranked[j].total {
			return ranked[i].total > ranked[j].total
		}
		return ranked[i].x > ranked[j].x
	})
}

func rankRows(ranked []rankedEntry, vis repository.Visibility) []ResultsRow {
	rows := make([]ResultsRow, len(ranked))
	rank := 0
	for i, r := range ranked {
		if i == 0 || r.total != ranked[i-1].total || r.x != ranked[i-1].x {
			rank = i + 1
		}
		rows[i] = resultsRow(r, rank, vis)
	}
	return rows
}

func entryClass(e repository.ReportEntry) (wa1500.Classification, bool) {
	id := e.Identity(e.Shooter.Club, e.Shooter.Country, e.Shooter.Classes)
	class := id.Class
	if class == nil && !e.Frozen() {
		class = id.Classification
	}
	if class == nil {
		return 0, false
	}
	return wa1500.Classification(*class), true
}

func classGroups(classified bool, category string, ranked []rankedEntry) (map[wa1500.Classification][]rankedEntry, bool) {
	if !classified || !rulebook.IsClassified(category) || len(ranked) == 0 {
		return nil, false
	}
	groups := map[wa1500.Classification][]rankedEntry{}
	for _, r := range ranked {
		class, ok := entryClass(r.entry)
		if !ok {
			return nil, false
		}
		groups[class] = append(groups[class], r)
	}
	return groups, true
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
	id := r.entry.Identity(sh.Club, sh.Country, sh.Classes)
	if vis.Clubs {
		row.Club = &id.Club
	}
	if vis.Countries {
		row.Country = &id.Country
	}
	if vis.Names {
		row.Class, row.CompeteAs = classificationPtr(id.Classification), classificationPtr(id.Class)
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
	id := r.entry.Identity(sh.Club, sh.Country, sh.Classes)
	if vis.Clubs {
		n.Club = &id.Club
	}
	if vis.Countries {
		n.Country = &id.Country
	}
	if vis.RingCounts {
		x := r.x
		n.XCount = &x
	}
	return n
}

func resultsDNFRow(r rankedEntry, vis repository.Visibility) ResultsDNFRow {
	row := resultsRow(r, 0, vis)
	segments := make([]ResultsSegment, len(row.Segments))
	for i, st := range row.Segments {
		segments[i] = ResultsSegment{Segment: st.Segment, Score: st.Score}
	}
	return ResultsDNFRow{
		LastName: row.LastName, FirstName: row.FirstName, WAID: row.WAID,
		Club: row.Club, Country: row.Country, Segments: segments,
	}
}
