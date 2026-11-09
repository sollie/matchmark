package service

import (
	"sort"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
	"github.com/sollie/matchmark/internal/wa1500"
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

type SegmentScoreView struct {
	Segment  int             `json:"segment"`
	Score    *int            `json:"score,omitempty"`
	Accepted bool            `json:"accepted"`
	State    SegmentState    `json:"state"`
	Counts   *RingCountsView `json:"ring_counts,omitempty"`
}

type SegmentState string

const (
	SegmentNotStarted         SegmentState = "not_started"
	SegmentInProgress         SegmentState = "in_progress"
	SegmentAwaitingAcceptance SegmentState = "awaiting_acceptance"
	SegmentAccepted           SegmentState = "accepted"
)

func segmentState(st repository.SegmentStatus, number, currentSegment int) SegmentState {
	switch st {
	case repository.StatusAccepted:
		return SegmentAccepted
	case repository.StatusAwaitingAcceptance:
		return SegmentAwaitingAcceptance
	}
	if number > currentSegment {
		return SegmentNotStarted
	}
	return SegmentInProgress
}

type StandingView struct {
	Lane       int                    `json:"lane"`
	Category   string                 `json:"category"`
	Withdrawn  bool                   `json:"withdrawn"`
	Total      int                    `json:"total"`
	LastName   *string                `json:"last_name,omitempty"`
	FirstName  *string                `json:"first_name,omitempty"`
	Club       *string                `json:"club,omitempty"`
	Country    *string                `json:"country,omitempty"`
	Class      *wa1500.Classification `json:"classification,omitempty"`
	CompeteAs  *wa1500.Classification `json:"class,omitempty"`
	ClassGroup *wa1500.Classification `json:"class_group,omitempty"`
	XCount     *int                   `json:"x_count,omitempty"`
	RingCounts *RingCountsView        `json:"ring_counts,omitempty"`
	Segments   []SegmentScoreView     `json:"segments,omitempty"`
}

func classificationPtr(v *int) *wa1500.Classification {
	if v == nil {
		return nil
	}
	c := wa1500.Classification(*v)
	return &c
}

func ringCountsView(c repository.RingCounts) RingCountsView {
	return RingCountsView{X: c.X, Ten: c.Ten, Nine: c.Nine, Eight: c.Eight, Seven: c.Seven, Zero: c.Zero, Miss: c.Miss}
}

func standingClass(e repository.SpectatorEntry, id repository.EntryIdentity) *wa1500.Classification {
	class := id.Class
	if class == nil && !e.Frozen() {
		class = id.Classification
	}
	return classificationPtr(class)
}

func applyClassGroups(views []StandingView, classified bool) {
	incomplete := map[string]bool{}
	for _, v := range views {
		if v.ClassGroup == nil {
			incomplete[v.Category] = true
		}
	}
	for i := range views {
		if !classified || incomplete[views[i].Category] {
			views[i].ClassGroup = nil
		}
	}
}

func projectStandings(currentSegment int, vis repository.Visibility, entries []repository.SpectatorEntry) []StandingView {
	type ranked struct {
		view StandingView
		x    int
	}
	rows := make([]ranked, 0, len(entries))
	for _, e := range entries {
		var total repository.RingCounts
		v := StandingView{Lane: e.Lane, Category: e.Category, Withdrawn: e.Withdrawn}
		for _, st := range e.Segments {
			accepted := st.Status == repository.StatusAccepted
			if !accepted {
				st.Counts = repository.RingCounts{}
			}
			v.Total += st.Counts.Score()
			total.X += st.Counts.X
			total.Ten += st.Counts.Ten
			total.Nine += st.Counts.Nine
			total.Eight += st.Counts.Eight
			total.Seven += st.Counts.Seven
			total.Zero += st.Counts.Zero
			total.Miss += st.Counts.Miss
			sv := SegmentScoreView{Segment: st.Number, Accepted: accepted, State: segmentState(st.Status, st.Number, currentSegment)}
			if vis.SegmentScores && accepted {
				score := st.Counts.Score()
				sv.Score = &score
				if vis.RingCounts {
					c := ringCountsView(st.Counts)
					sv.Counts = &c
				}
			}
			v.Segments = append(v.Segments, sv)
		}
		if vis.Names {
			v.LastName, v.FirstName = &e.LastName, &e.FirstName
		}
		id := e.Identity(e.Club, e.Country, e.Classes)
		if class := standingClass(e, id); class != nil && rulebook.IsClassified(e.Category) {
			v.ClassGroup = class
		}
		if vis.Names {
			v.Class, v.CompeteAs = classificationPtr(id.Classification), classificationPtr(id.Class)
		}
		if vis.Clubs {
			v.Club = &id.Club
		}
		if vis.Countries {
			v.Country = &id.Country
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
	return out
}
