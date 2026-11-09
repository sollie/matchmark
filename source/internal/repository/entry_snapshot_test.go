package repository_test

import (
	"testing"

	"github.com/sollie/matchmark/internal/repository"
)

func TestIdentityFallsBackToLiveValuesWhenNotFrozen(t *testing.T) {
	e := repository.Entry{Category: "Revolver 1500"}
	id := e.Identity("Live Club", "NOR", map[string]int{"Revolver 1500": 3})
	if id.Club != "Live Club" || id.Country != "NOR" || id.Classification == nil || *id.Classification != 3 || id.Class != nil {
		t.Fatalf("identity %+v", id)
	}
}

func TestIdentityReadsSnapshotWhenFrozen(t *testing.T) {
	club, country, classification := "Old Club", "SWE", 2
	e := repository.Entry{Category: "Revolver 1500", Club: &club, Country: &country, ClassificationAtMatch: &classification}
	id := e.Identity("Live Club", "NOR", map[string]int{"Revolver 1500": 5})
	if id.Club != "Old Club" || id.Country != "SWE" || *id.Classification != 2 || id.Class != nil {
		t.Fatalf("identity %+v", id)
	}
}
