package report

import (
	"testing"

	"github.com/sollie/matchmark/internal/repository"
)

func TestSortedEntriesTiebreaksByMatchDateThenID(t *testing.T) {
	shooter := repository.Shooter{LastName: "BERG", FirstName: "Ola"}
	mk := func(matchID int64, date string) repository.ReportEntry {
		return repository.ReportEntry{Entry: repository.Entry{MatchID: matchID, Category: "Revolver"}, Shooter: shooter, MatchDate: date}
	}
	got := sortedEntries([]repository.ReportEntry{mk(3, "2026-05-02"), mk(2, "2026-05-01"), mk(1, "2026-05-02")})
	want := []int64{2, 1, 3}
	for i, id := range want {
		if got[i].MatchID != id {
			t.Fatalf("position %d: match %d, want %d", i, got[i].MatchID, id)
		}
	}
}
