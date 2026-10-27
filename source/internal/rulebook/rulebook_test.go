package rulebook_test

import (
	"slices"
	"testing"

	"github.com/sollie/matchmark/internal/rulebook"
)

func TestCourseByKey(t *testing.T) {
	tests := []struct {
		key            string
		wantTotalShots int
		wantStages     int
		wantErr        bool
	}{
		{"150", 150, 10, false},
		{"60", 60, 4, false},
		{"48", 48, 4, false},
		{"40", 40, 4, false},
		{"unknown", 0, 0, true},
	}
	for _, tc := range tests {
		course, err := rulebook.CourseByKey(tc.key)
		if (err != nil) != tc.wantErr {
			t.Errorf("CourseByKey(%q) error = %v, wantErr %v", tc.key, err, tc.wantErr)
			continue
		}
		if tc.wantErr {
			continue
		}
		if course.TotalShots != tc.wantTotalShots {
			t.Errorf("CourseByKey(%q) TotalShots = %d, want %d", tc.key, course.TotalShots, tc.wantTotalShots)
		}
		if len(course.Stages) != tc.wantStages {
			t.Errorf("CourseByKey(%q) stages = %d, want %d", tc.key, len(course.Stages), tc.wantStages)
		}
		sum := 0
		for i, stage := range course.Stages {
			if stage.StageNumber != i+1 {
				t.Errorf("CourseByKey(%q) stage[%d].StageNumber = %d, want %d", tc.key, i, stage.StageNumber, i+1)
			}
			sum += stage.ExpectedShots
		}
		if sum != course.TotalShots {
			t.Errorf("CourseByKey(%q) sum of stage shots = %d, TotalShots = %d", tc.key, sum, course.TotalShots)
		}
	}
}

func TestRingValue(t *testing.T) {
	tests := []struct {
		ring    string
		want    int
		wantErr bool
	}{
		{"X", 10, false},
		{"10", 10, false},
		{"9", 9, false},
		{"8", 8, false},
		{"7", 7, false},
		{"0", 0, false},
		{"M", 0, false},
		{"11", 0, true},
		{"", 0, true},
		{"x", 0, true},
		{"miss", 0, true},
	}
	for _, tc := range tests {
		got, err := rulebook.RingValue(tc.ring)
		if (err != nil) != tc.wantErr {
			t.Errorf("RingValue(%q) error = %v, wantErr %v", tc.ring, err, tc.wantErr)
			continue
		}
		if !tc.wantErr && got != tc.want {
			t.Errorf("RingValue(%q) = %d, want %d", tc.ring, got, tc.want)
		}
	}
}

func TestCourseLabelsAndKeys(t *testing.T) {
	want := map[string]string{
		"150": "150 shots (WA1500)",
		"60":  "60 shots",
		"48":  "48 shots",
		"40":  "40 shots (5-shot revolver)",
	}
	keys := rulebook.CourseKeys()
	if len(keys) != len(want) {
		t.Fatalf("CourseKeys() = %v", keys)
	}
	for _, key := range keys {
		c, err := rulebook.CourseByKey(key)
		if err != nil {
			t.Fatalf("CourseByKey(%q): %v", key, err)
		}
		if c.Label != want[key] {
			t.Errorf("label for %q = %q, want %q", key, c.Label, want[key])
		}
	}
}

func TestCategoriesPerCourse(t *testing.T) {
	cases := map[string][]string{
		"150": {"Revolver", "Pistol", "Revolver OS", "Pistol OS"},
		"60":  {"Revolver", "Pistol", "Revolver OS", "Pistol OS"},
		"48":  {"Revolver", "Pistol"},
		"40":  {"Revolver"},
	}
	for key, want := range cases {
		got, err := rulebook.CategoriesForCourse(key)
		if err != nil {
			t.Fatalf("CategoriesForCourse(%q): %v", key, err)
		}
		if !slices.Equal(got, want) {
			t.Errorf("CategoriesForCourse(%q) = %v, want %v", key, got, want)
		}
		if err := rulebook.ValidateCategory(key, "Rifle"); err == nil {
			t.Errorf("ValidateCategory(%q, Rifle) accepted", key)
		}
	}
	if _, err := rulebook.CategoriesForCourse("99"); err == nil {
		t.Error("unknown course accepted")
	}
	if err := rulebook.ValidateCategory("40", "Pistol"); err == nil {
		t.Error("Pistol accepted for 40")
	}
	if err := rulebook.ValidateCategory("48", "Pistol"); err != nil {
		t.Errorf("Pistol rejected for 48: %v", err)
	}
}
