package report

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/sollie/matchmark/internal/repository"
)

func renderFixture() Results {
	pending := repository.ReportStage{Number: 2, Counts: repository.RingCounts{Ten: 9}, Status: repository.StatusAwaitingAcceptance}
	return Build(repository.Competition{Name: "Spring", Date: "2026-05-01", Course: "wa1500"}, []repository.ReportEntry{
		entry("Revolver", 1, "ZED", "W1", false, accepted(1, 2, 8), pending),
		entry("Revolver", 2, "ALF", "W2", false, accepted(1, 0, 1)),
		entry("Revolver", 3, "MOE", "", true, accepted(1, 1, 1)),
	})
}

func TestCSVSortsAlphabeticallyAndKeepsWithdrawn(t *testing.T) {
	out, err := CSV(renderFixture())
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(out, []byte("\xef\xbb\xbf")))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, r := range rows[1:] {
		if r[6] == "1" {
			order = append(order, r[2]+":"+r[5])
		}
		if r[2] == "ZED" && r[6] == "2" && strings.Join(r[7:16], ",") != "0,0,0,0,0,0,0,0,false" {
			t.Fatalf("unaccepted stage not zeroed: %v", r)
		}
	}
	if got := strings.Join(order, ","); got != "ALF:false,MOE:true,ZED:false" {
		t.Fatalf("order = %s", got)
	}
}

func TestWorkbookSplitsKnownAndNewShooters(t *testing.T) {
	out, err := Workbook(renderFixture())
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	get := func(ref string) string {
		v, _ := f.GetCellValue("Revolver", ref)
		return v
	}
	if get("A3") != "W2" || get("B3") != "10" || get("A4") != "W1" || get("B4") != "100" {
		t.Fatalf("known rows: %q %q %q %q", get("A3"), get("B3"), get("A4"), get("B4"))
	}
	if get("A14") != "MOE" || get("G14") != "20" {
		t.Fatalf("new row: %q %q", get("A14"), get("G14"))
	}
}

func TestCSVNeutralisesFormulas(t *testing.T) {
	res := Build(repository.Competition{Name: "Spring", Date: "2026-05-01", Course: "wa1500"}, []repository.ReportEntry{
		entry("Revolver", 1, "=HYPERLINK(\"X\")", "", false, accepted(1, 0, 1)),
	})
	out, err := CSV(res)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte("'=HYPERLINK(")) {
		t.Fatalf("formula not neutralised: %s", out)
	}
}

func TestWorkbookInsertsRowsForManyShooters(t *testing.T) {
	var entries []repository.ReportEntry
	for i := 1; i <= 10; i++ {
		entries = append(entries, entry("Revolver", i, "K"+strings.Repeat("a", i), "W"+strings.Repeat("1", i), false, accepted(1, 0, 1)))
	}
	for i := 11; i <= 14; i++ {
		entries = append(entries, entry("Revolver", i, "N"+strings.Repeat("a", i), "", false, accepted(1, 0, 1)))
	}
	out, err := Workbook(Build(repository.Competition{Name: "Big", Date: "2026-05-01", Course: "wa1500"}, entries))
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	get := func(ref string) string {
		v, _ := f.GetCellValue("Revolver", ref)
		return v
	}
	if get("A14") != "New shooters:" || get("A15") != "Last name" || get("A19") == "" || get("A20") != "" {
		t.Fatalf("block not shifted: %q %q %q", get("A14"), get("A15"), get("A19"))
	}
}
