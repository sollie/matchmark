package report

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"regexp"
	"testing"
	"time"

	"github.com/sollie/matchmark/internal/repository"
)

var pdfPage = regexp.MustCompile(`/Type /Page[^s]`)

func pageCount(pdf []byte) int {
	return len(pdfPage.FindAll(pdf, -1))
}

func sampleResults() ResultsPDFData {
	row := func(rank int, last, waid string) Row {
		return Row{
			Rank: rank, Shooter: repository.Shooter{LastName: last, FirstName: "Åse", WAID: waid, Club: "Łódź SK", Country: "NOR"},
			Stages: []Stage{{Number: 1, Accepted: true, Score: 90}, {Number: 2, Accepted: true, Score: 88}}, Total: 178, XCount: 9,
		}
	}
	fresh := row(2, "Ny", "")
	return ResultsPDFData{
		ClubName:  "Test Club",
		Generated: time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC),
		Results: Results{
			Competition: repository.Competition{Name: "Autumn Match", Date: "2026-10-24"},
			CourseLabel: "WA1500 150",
			Stages:      []int{1, 2},
			Categories: []Category{
				{Name: "Revolver", Rows: []Row{row(1, "Ørn", "1001"), row(2, "Berg", "1002")}},
				{Name: "Pistol", Rows: []Row{row(1, "Dahl", "1003"), fresh}, New: []Row{fresh}},
			},
		},
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
	d.Results = d.Results.Redact(repository.Visibility{})
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
	var rows []Row
	for i := 1; i <= 120; i++ {
		r := d.Results.Categories[0].Rows[0]
		r.Rank = i
		r.Shooter.LastName = fmt.Sprintf("Shooter%d", i)
		rows = append(rows, r)
	}
	d.Results.Categories = []Category{{Name: "Revolver", Rows: rows}}
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
	d.Results.Categories = nil
	out, err := ResultsPDF(d)
	if err != nil {
		t.Fatal(err)
	}
	if got := pageCount(out); got != 1 {
		t.Fatalf("pages = %d", got)
	}
}
