package rulebook_test

import (
	"slices"
	"testing"

	"github.com/sollie/matchmark/internal/rulebook"
)

func TestValidateWeaponClass(t *testing.T) {
	tests := []struct {
		discipline  string
		weaponClass string
		wantErr     bool
	}{
		{"1500", "Revolver 1500", false},
		{"1500", "Pistol 1500", false},
		{"1500", "Revolver Optical Sight 1500", false},
		{"1500", "Pistol Optical Sight 1500", false},
		{"60", "Distinguished Revolver", false},
		{"60", "Distinguished Pistol", false},
		{"60", "Open Match", false},
		{"48", "Standard Revolver 4.25\"", false},
		{"48", "Standard Revolver 2.75\"", false},
		{"48", "Standard Revolver 2.75\" 5-Shot", false},
		{"48", "Standard Semi-Automatic Pistol 5.5\" Production", false},
		{"48", "Standard Semi-Automatic Pistol 5.5\"", false},
		{"48", "Standard Semi-Automatic Pistol 3.7\"", false},
		{"1500", "Distinguished Revolver", true},
		{"60", "Revolver 1500", true},
		{"48", "Revolver 1500", true},
		{"unknown", "Revolver 1500", true},
		{"1500", "", true},
	}
	for _, tc := range tests {
		err := rulebook.ValidateWeaponClass(tc.discipline, tc.weaponClass)
		if (err != nil) != tc.wantErr {
			t.Errorf("ValidateWeaponClass(%q, %q) error = %v, wantErr %v", tc.discipline, tc.weaponClass, err, tc.wantErr)
		}
	}
}

func TestCourseOfFire(t *testing.T) {
	tests := []struct {
		discipline     string
		weaponClass    string
		wantTotalShots int
		wantStages     int
		wantErr        bool
	}{
		{"1500", "Revolver 1500", 150, 10, false},
		{"60", "Distinguished Revolver", 60, 4, false},
		{"48", "Standard Revolver 4.25\"", 48, 4, false},
		{"48", "Standard Revolver 2.75\" 5-Shot", 40, 4, false},
		{"unknown", "anything", 0, 0, true},
	}
	for _, tc := range tests {
		course, err := rulebook.CourseOfFire(tc.discipline, tc.weaponClass)
		if (err != nil) != tc.wantErr {
			t.Errorf("CourseOfFire(%q, %q) error = %v, wantErr %v", tc.discipline, tc.weaponClass, err, tc.wantErr)
			continue
		}
		if tc.wantErr {
			continue
		}
		if course.TotalShots != tc.wantTotalShots {
			t.Errorf("CourseOfFire(%q, %q) TotalShots = %d, want %d", tc.discipline, tc.weaponClass, course.TotalShots, tc.wantTotalShots)
		}
		if len(course.Stages) != tc.wantStages {
			t.Errorf("CourseOfFire(%q, %q) stages = %d, want %d", tc.discipline, tc.weaponClass, len(course.Stages), tc.wantStages)
		}
	}
}

func TestCourseOfFire_StageNumbers(t *testing.T) {
	disciplines := []struct {
		discipline  string
		weaponClass string
	}{
		{"1500", "Revolver 1500"},
		{"60", "Distinguished Revolver"},
		{"48", "Standard Revolver 4.25\""},
		{"48", "Standard Revolver 2.75\" 5-Shot"},
	}
	for _, d := range disciplines {
		course, err := rulebook.CourseOfFire(d.discipline, d.weaponClass)
		if err != nil {
			t.Fatalf("CourseOfFire(%q, %q): %v", d.discipline, d.weaponClass, err)
		}
		for i, stage := range course.Stages {
			if stage.StageNumber != i+1 {
				t.Errorf("CourseOfFire(%q, %q) stage[%d].StageNumber = %d, want %d",
					d.discipline, d.weaponClass, i, stage.StageNumber, i+1)
			}
		}
	}
}

func TestCourseOfFire_ShotCountsMatchTotal(t *testing.T) {
	disciplines := []struct {
		discipline  string
		weaponClass string
	}{
		{"1500", "Revolver 1500"},
		{"60", "Distinguished Revolver"},
		{"48", "Standard Revolver 4.25\""},
		{"48", "Standard Revolver 2.75\" 5-Shot"},
	}
	for _, d := range disciplines {
		course, err := rulebook.CourseOfFire(d.discipline, d.weaponClass)
		if err != nil {
			t.Fatalf("CourseOfFire(%q, %q): %v", d.discipline, d.weaponClass, err)
		}
		sum := 0
		for _, stage := range course.Stages {
			sum += stage.ExpectedShots
		}
		if sum != course.TotalShots {
			t.Errorf("CourseOfFire(%q, %q) sum of stage shots = %d, TotalShots = %d",
				d.discipline, d.weaponClass, sum, course.TotalShots)
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
