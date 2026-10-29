package service

import (
	"reflect"
	"testing"

	"github.com/sollie/matchmark/internal/repository"
)

func spectatorEntry(lane int, category string, withdrawn bool, segments ...repository.SpectatorSegment) repository.SpectatorEntry {
	e := repository.SpectatorEntry{LastName: "L", FirstName: "F", Club: "C", Country: "NOR", Segments: segments}
	e.Lane, e.Category, e.Withdrawn = lane, category, withdrawn
	return e
}

func accepted(n int, c repository.RingCounts) repository.SpectatorSegment {
	return repository.SpectatorSegment{Number: n, Counts: c, Status: repository.StatusAccepted}
}

func TestProjectStandingsVisibility(t *testing.T) {
	counts := repository.RingCounts{X: 2, Ten: 4, Nine: 6}
	entry := spectatorEntry(1, "Pistol", false, accepted(1, counts))
	tests := []struct {
		name string
		vis  repository.Visibility
		want func(StandingView) bool
	}{
		{"nothing visible", repository.Visibility{}, func(v StandingView) bool {
			return v.LastName == nil && v.Club == nil && v.Country == nil && v.RingCounts == nil && v.XCount == nil &&
				v.Segments[0].Score == nil && v.Segments[0].Counts == nil
		}},
		{"names clubs countries", repository.Visibility{Names: true, Clubs: true, Countries: true}, func(v StandingView) bool {
			return *v.LastName == "L" && *v.FirstName == "F" && *v.Club == "C" && *v.Country == "NOR" && v.Segments[0].Score == nil
		}},
		{"segment scores without ring counts", repository.Visibility{SegmentScores: true}, func(v StandingView) bool {
			return *v.Segments[0].Score == counts.Score() && v.Segments[0].Counts == nil && v.RingCounts == nil
		}},
		{"ring counts without segment scores", repository.Visibility{RingCounts: true}, func(v StandingView) bool {
			return v.Segments[0].Score == nil && v.Segments[0].Counts == nil && *v.XCount == 2 && v.RingCounts.Ten == 4
		}},
		{"segment scores and ring counts", repository.Visibility{SegmentScores: true, RingCounts: true}, func(v StandingView) bool {
			return *v.Segments[0].Score == counts.Score() && v.Segments[0].Counts.Nine == 6 && v.RingCounts.X == 2
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := projectStandings(1, tt.vis, []repository.SpectatorEntry{entry})
			if len(got) != 1 || got[0].Total != counts.Score() || !tt.want(got[0]) {
				t.Fatalf("got %+v", got)
			}
		})
	}
}

func TestProjectStandingsUnacceptedSegments(t *testing.T) {
	counts := repository.RingCounts{X: 1, Ten: 5, Nine: 6}
	tests := []struct {
		name        string
		status      repository.SegmentStatus
		number      int
		wantState   SegmentState
		wantTotal   int
		wantAccept  bool
		wantNoScore bool
	}{
		{"accepted", repository.StatusAccepted, 1, SegmentAccepted, counts.Score(), true, false},
		{"awaiting acceptance", repository.StatusAwaitingAcceptance, 1, SegmentAwaitingAcceptance, 0, false, true},
		{"open current segment", repository.StatusOpen, 1, SegmentInProgress, 0, false, true},
		{"open future segment", repository.StatusOpen, 2, SegmentNotStarted, 0, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := spectatorEntry(1, "Pistol", false, repository.SpectatorSegment{Number: tt.number, Counts: counts, Status: tt.status})
			got := projectStandings(1, repository.Visibility{SegmentScores: true, RingCounts: true}, []repository.SpectatorEntry{entry})[0]
			sv := got.Segments[0]
			if got.Total != tt.wantTotal || sv.State != tt.wantState || sv.Accepted != tt.wantAccept || (sv.Score == nil) != tt.wantNoScore {
				t.Fatalf("got total %d segment %+v", got.Total, sv)
			}
			if tt.wantNoScore && got.XCount != nil && *got.XCount != 0 {
				t.Fatalf("x_count = %d, want 0 for unaccepted segment", *got.XCount)
			}
		})
	}
}

func TestProjectStandingsOrderingAndGrouping(t *testing.T) {
	high := repository.RingCounts{X: 3, Ten: 3, Nine: 6}
	moreX := repository.RingCounts{X: 5, Ten: 1, Nine: 6}
	tests := []struct {
		name    string
		entries []repository.SpectatorEntry
		want    []int
	}{
		{"higher total first", []repository.SpectatorEntry{
			spectatorEntry(1, "Pistol", false, accepted(1, repository.RingCounts{Nine: 6})),
			spectatorEntry(2, "Pistol", false, accepted(1, high)),
		}, []int{2, 1}},
		{"equal total ranks more X first", []repository.SpectatorEntry{
			spectatorEntry(1, "Pistol", false, accepted(1, repository.RingCounts{X: 1, Ten: 5, Nine: 6})),
			spectatorEntry(2, "Pistol", false, accepted(1, moreX)),
		}, []int{2, 1}},
		{"full tie keeps input order", []repository.SpectatorEntry{
			spectatorEntry(3, "Pistol", false, accepted(1, high)),
			spectatorEntry(1, "Revolver", false, accepted(1, high)),
		}, []int{3, 1}},
		{"categories ranked together and kept", []repository.SpectatorEntry{
			spectatorEntry(1, "Revolver", false, accepted(1, repository.RingCounts{Nine: 6})),
			spectatorEntry(2, "Pistol", false, accepted(1, high)),
			spectatorEntry(3, "Revolver", true, accepted(1, repository.RingCounts{Ten: 6})),
		}, []int{2, 3, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := projectStandings(1, repository.Visibility{}, tt.entries)
			var lanes []int
			for _, v := range got {
				lanes = append(lanes, v.Lane)
				for _, e := range tt.entries {
					if e.Lane == v.Lane && (e.Category != v.Category || e.Withdrawn != v.Withdrawn) {
						t.Fatalf("category or withdrawn lost for lane %d: %+v", v.Lane, v)
					}
				}
			}
			if !reflect.DeepEqual(lanes, tt.want) {
				t.Fatalf("lanes = %v, want %v", lanes, tt.want)
			}
		})
	}
}
