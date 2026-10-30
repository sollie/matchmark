package report

import (
	"testing"

	"github.com/sollie/matchmark/internal/repository"
)

func TestSortedEntriesTiebreaksByMatchID(t *testing.T) {
	shooter := repository.Shooter{LastName: "BERG", FirstName: "Ola"}
	mk := func(matchID int64) repository.ReportEntry {
		return repository.ReportEntry{Entry: repository.Entry{MatchID: matchID, Category: "Revolver"}, Shooter: shooter}
	}
	got := sortedEntries([]repository.ReportEntry{mk(3), mk(1), mk(2)})
	want := []int64{1, 2, 3}
	for i, id := range want {
		if got[i].MatchID != id {
			t.Fatalf("position %d: match %d, want %d", i, got[i].MatchID, id)
		}
	}
}
