package report

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
)

func TestScorecardGridHasOneColumnPerSegment(t *testing.T) {
	course, err := rulebook.CourseByKey("48")
	if err != nil {
		t.Fatal(err)
	}
	lane := 2
	d := ScorecardData{
		Course: course,
		Entry: repository.ReportEntry{
			Shooter: repository.Shooter{FirstName: "Ola", LastName: "Berg"},
			Segments: []repository.ReportSegment{
				{Number: 1, Status: repository.StatusAccepted, ScorerLane: &lane, ScorerName: "Kari",
					Counts: repository.RingCounts{X: 3, Ten: 5, Nine: 4, Eight: 3, Seven: 2, Zero: 1}},
				{Number: 2, Status: repository.StatusOpen, Counts: repository.RingCounts{X: 30}},
			},
		},
	}
	score := strconv.Itoa(d.Entry.Segments[0].Counts.Score())
	got := scorecardGrid(d)
	want := [][]string{
		{"", "Stage 1+2", "Stage 3+4", "Total"},
		{"X", "3", "", "3"},
		{"10", "5", "", "5"},
		{"9", "4", "", "4"},
		{"8", "3", "", "3"},
		{"7", "2", "", "2"},
		{"0", "1", "", "1"},
		{"M", "0", "", "0"},
		{"Score", score, "", score},
		{"Scorer", "Kari", "", ""},
		{"Shooter", "Ola Berg", "Ola Berg", ""},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("grid\n got %q\nwant %q", got, want)
	}
}
