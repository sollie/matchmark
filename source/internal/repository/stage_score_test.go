package repository_test

import (
	"testing"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
)

func TestRingCountsScore(t *testing.T) {
	tests := []struct {
		name   string
		counts repository.RingCounts
		want   int
	}{
		{"zero value", repository.RingCounts{}, 0},
		{"zeros and misses score nothing", repository.RingCounts{Zero: 3, Miss: 2}, 0},
		{"X counts as ten", repository.RingCounts{X: 2}, 20},
		{"X and ten are equal", repository.RingCounts{X: 1, Ten: 1}, 20},
		{"mixed", repository.RingCounts{X: 1, Ten: 2, Nine: 3, Eight: 4, Seven: 5, Zero: 1, Miss: 1}, 30 + 27 + 32 + 35},
	}
	for _, tc := range tests {
		if got := tc.counts.Score(); got != tc.want {
			t.Errorf("%s: Score() = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestRingCountsScoreMatchesRulebook(t *testing.T) {
	counts := map[string]repository.RingCounts{
		"X":  {X: 1},
		"10": {Ten: 1},
		"9":  {Nine: 1},
		"8":  {Eight: 1},
		"7":  {Seven: 1},
		"0":  {Zero: 1},
		"M":  {Miss: 1},
	}
	for _, ring := range rulebook.RingValues() {
		want, err := rulebook.RingValue(ring)
		if err != nil {
			t.Fatal(err)
		}
		if got := counts[ring].Score(); got != want {
			t.Errorf("ring %q: Score() = %d, rulebook = %d", ring, got, want)
		}
	}
}
