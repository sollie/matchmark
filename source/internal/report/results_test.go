package report

import (
	"reflect"
	"testing"

	"github.com/sollie/matchmark/internal/repository"
)

func entry(category string, lane int, last, waid string, withdrawn bool, stages ...repository.ReportStage) repository.ReportEntry {
	return repository.ReportEntry{
		Entry:   repository.Entry{Category: category, Lane: lane, Withdrawn: withdrawn},
		Shooter: repository.Shooter{LastName: last, FirstName: "F" + last, WAID: waid, Club: "Club " + last, Country: "NOR"},
		Stages:  stages,
	}
}

func accepted(n, x, ten int) repository.ReportStage {
	return repository.ReportStage{Number: n, Counts: repository.RingCounts{X: x, Ten: ten}, Status: repository.StatusAccepted}
}

func names(rows []Row) []string {
	var out []string
	for _, r := range rows {
		out = append(out, r.Shooter.LastName)
	}
	return out
}

func ranks(rows []Row) []int {
	var out []int
	for _, r := range rows {
		out = append(out, r.Rank)
	}
	return out
}

func TestBuildRanking(t *testing.T) {
	comp := repository.Competition{Course: "150", Name: "Cup", Date: "2026-01-01"}
	entries := []repository.ReportEntry{
		entry("Pistol", 1, "Low", "W1", false, accepted(1, 0, 5)),
		entry("Pistol", 2, "HighX", "W2", false, accepted(1, 3, 2)),
		entry("Pistol", 3, "HighTen", "W3", false, accepted(1, 0, 5)),
		entry("Pistol", 4, "Top", "W4", false, accepted(1, 0, 9)),
		entry("Revolver", 5, "Solo", "W5", false, accepted(1, 1, 0)),
	}
	res := Build(comp, entries)
	if len(res.Categories) != 2 || res.Categories[0].Name != "Revolver" || res.Categories[1].Name != "Pistol" {
		t.Fatalf("categories = %+v", res.Categories)
	}
	pistol := res.Categories[1]
	if got, want := names(pistol.Rows), []string{"Top", "HighX", "Low", "HighTen"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	if got, want := ranks(pistol.Rows), []int{1, 2, 3, 3}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ranks = %v, want %v", got, want)
	}
	if pistol.Rows[1].Total != 50 || pistol.Rows[1].XCount != 3 {
		t.Fatalf("row = %+v", pistol.Rows[1])
	}
}

func TestBuildTieBreaksOnXCount(t *testing.T) {
	entries := []repository.ReportEntry{
		entry("Pistol", 1, "NoX", "W1", false, accepted(1, 0, 5)),
		entry("Pistol", 2, "WithX", "W2", false, accepted(1, 2, 3)),
	}
	rows := Build(repository.Competition{Course: "150"}, entries).Categories[0].Rows
	if got, want := names(rows), []string{"WithX", "NoX"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	if got, want := ranks(rows), []int{1, 2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ranks = %v, want %v", got, want)
	}
}

func TestBuildNewSubset(t *testing.T) {
	entries := []repository.ReportEntry{
		entry("Pistol", 1, "Known", "W1", false, accepted(1, 0, 9)),
		entry("Pistol", 2, "Fresh", "", false, accepted(1, 0, 5)),
	}
	cat := Build(repository.Competition{Course: "150"}, entries).Categories[0]
	if len(cat.Rows) != 2 {
		t.Fatalf("rows = %v", names(cat.Rows))
	}
	if got := names(cat.New); !reflect.DeepEqual(got, []string{"Fresh"}) || cat.New[0].Rank != 2 {
		t.Fatalf("new = %+v", cat.New)
	}
}

func TestBuildAcceptedStagesOnly(t *testing.T) {
	lane := 4
	pending := repository.ReportStage{Number: 2, Counts: repository.RingCounts{X: 5, Ten: 5}, Status: repository.StatusOpen, ScorerLane: &lane, ScorerName: "Ann"}
	entries := []repository.ReportEntry{entry("Pistol", 1, "A", "W1", false, accepted(1, 1, 1), pending)}
	row := Build(repository.Competition{Course: "150"}, entries).Categories[0].Rows[0]
	if row.Total != 20 || row.XCount != 1 {
		t.Fatalf("total=%d x=%d", row.Total, row.XCount)
	}
	got := row.Stages[1]
	if got.Accepted || got.Counts != (repository.RingCounts{}) || got.Score != 0 {
		t.Fatalf("stage = %+v", got)
	}
	if got.ScorerLane == nil || *got.ScorerLane != 4 || got.ScorerName != "Ann" || got.ScorerLabel != "Ann" {
		t.Fatalf("scorer = %+v", got)
	}
	if !row.Stages[0].Accepted || row.Stages[0].Score != 20 || row.Stages[0].ScorerLabel != "Range admin" {
		t.Fatalf("stage 1 = %+v", row.Stages[0])
	}
}

func TestBuildWithdrawn(t *testing.T) {
	entries := []repository.ReportEntry{
		entry("Pistol", 1, "Out", "W1", true, accepted(1, 0, 9)),
		entry("Pistol", 2, "In", "W2", false, accepted(1, 0, 1)),
	}
	rows := Build(repository.Competition{Course: "150"}, entries).Categories[0].Rows
	if got, want := names(rows), []string{"In", "Out"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	if rows[0].Withdrawn || !rows[1].Withdrawn || rows[0].Rank != 1 || rows[1].Rank != 0 {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestBuildIgnoresCompetitionStatus(t *testing.T) {
	entries := []repository.ReportEntry{entry("Pistol", 1, "A", "W1", false, accepted(1, 0, 1))}
	res := Build(repository.Competition{Course: "150", Status: "draft"}, entries)
	if len(res.Categories) != 1 || len(res.Stages) == 0 || res.CourseLabel == "" {
		t.Fatalf("res = %+v", res)
	}
}

func TestRedact(t *testing.T) {
	base := Build(repository.Competition{Course: "150"}, []repository.ReportEntry{
		entry("Pistol", 1, "Known", "W1", false, accepted(1, 2, 1)),
		entry("Pistol", 2, "Fresh", "", false, accepted(1, 0, 1)),
	})
	all := repository.Visibility{Names: true, Clubs: true, Countries: true, StageScores: true, RingCounts: true}
	cases := []struct {
		name  string
		hide  func(*repository.Visibility)
		check func(t *testing.T, r Row)
	}{
		{"names", func(v *repository.Visibility) { v.Names = false }, func(t *testing.T, r Row) {
			if r.Shooter.LastName != "" || r.Shooter.FirstName != "" || r.Shooter.WAID != "" {
				t.Errorf("names leaked: %+v", r.Shooter)
			}
		}},
		{"clubs", func(v *repository.Visibility) { v.Clubs = false }, func(t *testing.T, r Row) {
			if r.Shooter.Club != "" {
				t.Errorf("club leaked")
			}
		}},
		{"countries", func(v *repository.Visibility) { v.Countries = false }, func(t *testing.T, r Row) {
			if r.Shooter.Country != "" {
				t.Errorf("country leaked")
			}
		}},
		{"stage scores", func(v *repository.Visibility) { v.StageScores = false }, func(t *testing.T, r Row) {
			if r.Stages[0].Score != 0 || r.Total == 0 {
				t.Errorf("row = %+v", r)
			}
		}},
		{"ring counts", func(v *repository.Visibility) { v.RingCounts = false }, func(t *testing.T, r Row) {
			if r.XCount != 0 || r.Stages[0].Counts != (repository.RingCounts{}) {
				t.Errorf("counts leaked: %+v", r)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vis := all
			tc.hide(&vis)
			res := base.Redact(vis)
			if res.Visibility == nil || *res.Visibility != vis {
				t.Fatalf("visibility = %+v", res.Visibility)
			}
			cat := res.Categories[0]
			for _, r := range append(append([]Row(nil), cat.Rows...), cat.New...) {
				tc.check(t, r)
			}
			if base.Categories[0].Rows[0].Shooter.LastName == "" {
				t.Fatal("Redact mutated receiver")
			}
		})
	}

	t.Run("all visible", func(t *testing.T) {
		res := base.Redact(all)
		if got, want := names(res.Categories[0].Rows), names(base.Categories[0].Rows); !reflect.DeepEqual(got, want) || res.Categories[0].Rows[0].Total != base.Categories[0].Rows[0].Total {
			t.Fatal("visible results changed")
		}
	})

	t.Run("scorers hidden", func(t *testing.T) {
		res := base.Redact(all)
		for _, s := range res.Categories[0].Rows[0].Stages {
			if s.ScorerName != "" || s.ScorerLane != nil || s.ScorerLabel != "" {
				t.Fatalf("scorer leaked: %+v", s)
			}
		}
	})
}
