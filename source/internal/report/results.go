package report

import (
	"sort"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
)

type Stage struct {
	Number      int
	Accepted    bool
	Counts      repository.RingCounts
	Score       int
	ScorerLane  *int
	ScorerName  string
	ScorerLabel string
}

type Row struct {
	Rank      int
	Lane      int
	Shooter   repository.Shooter
	Withdrawn bool
	Stages    []Stage
	Total     int
	XCount    int
}

type Category struct {
	Name string
	Rows []Row
	New  []Row
}

type Results struct {
	Competition repository.Competition
	CourseLabel string
	Stages      []int
	Categories  []Category
	Visibility  repository.Visibility
}

func Build(comp repository.Competition, entries []repository.ReportEntry) Results {
	res := Results{Competition: comp, Categories: []Category{}, Visibility: repository.AllVisible()}
	if course, err := rulebook.CourseByKey(comp.Course); err == nil {
		res.CourseLabel = course.Label
		for _, s := range course.Stages {
			res.Stages = append(res.Stages, s.StageNumber)
		}
	}
	byCategory := map[string][]Row{}
	for _, e := range entries {
		byCategory[e.Category] = append(byCategory[e.Category], buildRow(e))
	}
	for _, name := range rulebook.Categories() {
		rows := byCategory[name]
		if len(rows) == 0 {
			continue
		}
		rank(rows)
		cat := Category{Name: name, Rows: rows}
		for _, r := range rows {
			if r.Shooter.WAID == "" {
				cat.New = append(cat.New, r)
			}
		}
		res.Categories = append(res.Categories, cat)
	}
	return res
}

func buildRow(e repository.ReportEntry) Row {
	r := Row{Lane: e.Lane, Shooter: e.Shooter, Withdrawn: e.Withdrawn, Stages: make([]Stage, len(e.Stages))}
	for i, s := range e.Stages {
		st := Stage{Number: s.Number, Accepted: s.Status == repository.StatusAccepted, ScorerLane: s.ScorerLane, ScorerName: s.ScorerName}
		st.ScorerLabel = ScorerLabel(s)
		if st.Accepted {
			st.Counts = s.Counts
			st.Score = s.Counts.Score()
			r.Total += st.Score
			r.XCount += s.Counts.X
		}
		r.Stages[i] = st
	}
	return r
}

func rank(rows []Row) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.Withdrawn != b.Withdrawn {
			return !a.Withdrawn
		}
		if a.Total != b.Total {
			return a.Total > b.Total
		}
		return a.XCount > b.XCount
	})
	current := 0
	for i := range rows {
		if rows[i].Withdrawn {
			rows[i].Rank = 0
			continue
		}
		if i == 0 || rows[i].Total != rows[i-1].Total || rows[i].XCount != rows[i-1].XCount {
			current = i + 1
		}
		rows[i].Rank = current
	}
}

func (r Results) Redact(v repository.Visibility) Results {
	out := r
	out.Visibility = v
	out.Categories = make([]Category, len(r.Categories))
	for i, c := range r.Categories {
		cat := Category{Name: c.Name, Rows: redactRows(c.Rows, v)}
		if v.Names {
			cat.New = redactRows(c.New, v)
		}
		out.Categories[i] = cat
	}
	return out
}

func redactRows(rows []Row, v repository.Visibility) []Row {
	if rows == nil {
		return nil
	}
	out := make([]Row, len(rows))
	for i, r := range rows {
		r.Stages = append([]Stage(nil), r.Stages...)
		if !v.Names {
			r.Shooter.LastName, r.Shooter.FirstName, r.Shooter.WAID = "", "", ""
		}
		if !v.Clubs {
			r.Shooter.Club = ""
		}
		if !v.Countries {
			r.Shooter.Country = ""
		}
		if !v.RingCounts {
			r.XCount = 0
		}
		for j := range r.Stages {
			s := &r.Stages[j]
			if !v.StageScores {
				s.Score = 0
			}
			if !v.RingCounts {
				s.Counts = repository.RingCounts{}
			}
			s.ScorerLane, s.ScorerName, s.ScorerLabel = nil, "", ""
		}
		out[i] = r
	}
	return out
}
