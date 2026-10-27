package service_test

import (
	"testing"

	"github.com/sollie/matchmark/internal/service"
)

func TestRulebookView(t *testing.T) {
	view, err := service.New(nil).Rulebook()
	if err != nil {
		t.Fatalf("Rulebook: %v", err)
	}
	want := []struct {
		key        string
		label      string
		discipline string
		total      int
		stages     int
		stageOne   int
	}{
		{"150", "150 shots (WA1500)", "1500", 150, 10, 12},
		{"60", "60 shots", "60", 60, 4, 12},
		{"48", "48 shots", "48", 48, 4, 6},
		{"40", "40 shots (5-shot revolver)", "48", 40, 4, 5},
	}
	if len(view.Courses) != len(want) {
		t.Fatalf("courses = %d, want %d", len(view.Courses), len(want))
	}
	for i, w := range want {
		c := view.Courses[i]
		if c.Key != w.key || c.Label != w.label || c.Discipline != w.discipline || c.TotalShots != w.total || len(c.Stages) != w.stages || c.Stages[0].ExpectedShots != w.stageOne {
			t.Errorf("course %d = %+v, want %+v", i, c, w)
		}
		sum := 0
		for _, st := range c.Stages {
			sum += st.ExpectedShots
		}
		if sum != c.TotalShots {
			t.Errorf("course %s stage shots = %d, want %d", c.Key, sum, c.TotalShots)
		}
	}
	if len(view.Rings) != 7 || view.Rings[0].Ring != "X" || view.Rings[0].Value != 10 {
		t.Errorf("rings = %+v", view.Rings)
	}
}
