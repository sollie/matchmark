package report

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"regexp"
	"testing"
	"time"
)

var pdfPage = regexp.MustCompile(`/Type /Page[^s]`)

func pageCount(pdf []byte) int {
	return len(pdfPage.FindAll(pdf, -1))
}

func ptr[T any](v T) *T { return &v }

func sampleResults() ResultsPDFData {
	row := func(rank int, last string) ResultsPDFRow {
		return ResultsPDFRow{
			Rank: rank, LastName: ptr(last), FirstName: ptr("Åse"), WAID: ptr("1001"), Club: ptr("Łódź SK"), Country: ptr("NOR"),
			Stages: []ResultsPDFStage{{Stage: 1, Score: ptr(90)}, {Stage: 2, Score: ptr(88)}}, Total: 178, XCount: ptr(9),
		}
	}
	return ResultsPDFData{
		ClubName:    "Test Club",
		Generated:   time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC),
		Name:        "Autumn Match",
		Date:        "2026-10-24",
		CourseLabel: "WA1500 150",
		Stages:      []int{1, 2},
		Categories: []ResultsPDFCategory{
			{Category: "Revolver", Rows: []ResultsPDFRow{row(1, "Ørn"), row(2, "Berg")}},
			{Category: "Pistol", Rows: []ResultsPDFRow{row(1, "Dahl")}},
		},
		NewShooters: []ResultsPDFNewShooter{{LastName: "Ny", FirstName: "Nina", Category: "Pistol", Club: ptr("Łódź SK"), Total: 170, XCount: ptr(4)}},
	}
}

func TestResultsPDFHasOnePagePerCategory(t *testing.T) {
	out, err := ResultsPDF(sampleResults())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Fatalf("not a PDF: %q", out[:8])
	}
	if got := pageCount(out); got != 3 {
		t.Fatalf("pages = %d, want 2 categories + new shooters", got)
	}
}

func TestResultsPDFWithHiddenColumnsAndLogo(t *testing.T) {
	d := sampleResults()
	for i := range d.Categories {
		for j := range d.Categories[i].Rows {
			r := &d.Categories[i].Rows[j]
			r.LastName, r.FirstName, r.WAID, r.Club, r.Country, r.XCount = nil, nil, nil, nil, nil, nil
			for k := range r.Stages {
				r.Stages[k].Score = nil
			}
		}
	}
	d.NewShooters = nil
	var logo bytes.Buffer
	if err := png.Encode(&logo, image.NewRGBA(image.Rect(0, 0, 40, 20))); err != nil {
		t.Fatal(err)
	}
	d.Logo = logo.Bytes()
	d.LogoType = "image/png"
	out, err := ResultsPDF(d)
	if err != nil {
		t.Fatal(err)
	}
	if got := pageCount(out); got != 2 {
		t.Fatalf("pages = %d", got)
	}
}

func TestResultsPDFPaginatesLongCategories(t *testing.T) {
	d := sampleResults()
	var rows []ResultsPDFRow
	for i := 1; i <= 120; i++ {
		r := d.Categories[0].Rows[0]
		r.Rank = i
		r.LastName = ptr(fmt.Sprintf("Shooter%d", i))
		rows = append(rows, r)
	}
	d.Categories = []ResultsPDFCategory{{Category: "Revolver", Rows: rows}}
	d.NewShooters = nil
	out, err := ResultsPDF(d)
	if err != nil {
		t.Fatal(err)
	}
	if got := pageCount(out); got < 3 {
		t.Fatalf("pages = %d, want the category to continue over several pages", got)
	}
}

func TestResultsPDFWithoutCategories(t *testing.T) {
	d := sampleResults()
	d.Categories, d.NewShooters = nil, nil
	out, err := ResultsPDF(d)
	if err != nil {
		t.Fatal(err)
	}
	if got := pageCount(out); got != 1 {
		t.Fatalf("pages = %d", got)
	}
}
