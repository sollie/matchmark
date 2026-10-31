package report

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"regexp"
	"slices"
	"testing"
	"time"

	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/core"
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
			Segments: []ResultsPDFSegment{{Segment: 1, Score: ptr(90)}, {Segment: 2, Score: ptr(88)}}, Total: 178, XCount: ptr(9),
		}
	}
	return ResultsPDFData{
		ClubName:    "Test Club",
		Generated:   time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC),
		Name:        "Autumn Match",
		Date:        "2026-10-24",
		CourseLabel: "WA1500 150",
		Segments:    []int{1, 2},
		Categories: []ResultsPDFCategory{
			{Category: "Revolver", Rows: []ResultsPDFRow{row(1, "Ørn"), row(2, "Berg")}},
			{Category: "Pistol", Rows: []ResultsPDFRow{row(1, "Dahl")}},
		},
		NewShooters: []ResultsPDFNewShooter{{LastName: "Ny", FirstName: "Nina", Category: "Pistol", Club: ptr("Łódź SK"), Total: 170, XCount: ptr(4)}},
	}
}

func TestResultsPDFFlowsCategoriesBeforeNewShootersPage(t *testing.T) {
	out, err := ResultsPDF(sampleResults())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Fatalf("not a PDF: %q", out[:8])
	}
	if got := pageCount(out); got != 2 {
		t.Fatalf("pages = %d, want one page of categories and one of new shooters", got)
	}
}

func TestResultsPDFWithHiddenColumnsAndLogo(t *testing.T) {
	d := sampleResults()
	for i := range d.Categories {
		for j := range d.Categories[i].Rows {
			r := &d.Categories[i].Rows[j]
			r.LastName, r.FirstName, r.WAID, r.Club, r.Country, r.XCount = nil, nil, nil, nil, nil, nil
			for k := range r.Segments {
				r.Segments[k].Score = nil
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
	if got := pageCount(out); got != 1 {
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

func TestResultsPDFWithoutClubSettings(t *testing.T) {
	d := sampleResults()
	d.ClubName, d.Logo, d.LogoType = "", nil, ""
	out, err := ResultsPDF(d)
	if err != nil {
		t.Fatal(err)
	}
	if got := pageCount(out); got != 2 {
		t.Fatalf("pages = %d", got)
	}
}

func TestResultsColumnsUseSegmentNames(t *testing.T) {
	d := sampleResults()
	d.SegmentNames = []string{"Match 1", "Match 2"}
	columns, _ := resultsColumns(d)
	var titles []string
	for _, c := range columns {
		titles = append(titles, c.title)
	}
	for _, want := range d.SegmentNames {
		if !slices.Contains(titles, want) {
			t.Fatalf("titles %q lack %q", titles, want)
		}
	}
	if slices.Contains(titles, "S1") {
		t.Fatalf("titles %q still numbered", titles)
	}
}

func TestResultsColumnsShowDNFRowsAsDNF(t *testing.T) {
	d := sampleResults()
	d.Categories[0].Rows = append(d.Categories[0].Rows, ResultsPDFRow{
		LastName: ptr("Slow"), FirstName: ptr("Sam"), Club: ptr("Łódź SK"), Country: ptr("NOR"), DNF: true,
		Segments: []ResultsPDFSegment{{Segment: 1, Score: ptr(80)}, {Segment: 2}},
	})
	_, cells := resultsColumns(d)
	got := cells(d.Categories[0].Rows[2])
	want := []string{"", "Sam Slow", "", "Łódź SK", "NOR", "80", "", "DNF", ""}
	if !slices.Equal(got, want) {
		t.Fatalf("cells %q, want %q", got, want)
	}
	if got := d.participants(); got != 4 {
		t.Fatalf("participants = %d, want ranked and DNF rows", got)
	}
}

func TestPDFFlowKeepsBlockOnOnePage(t *testing.T) {
	var added []float64
	f := &pdfFlow{add: func(rows ...core.Row) {
		for range rows {
			added = append(added, 0)
		}
	}, used: pdfLogoHeight + pdfHeadGap}
	f.keep(50)
	if len(added) != 0 {
		t.Fatal("block that fits must not be pushed down")
	}
	f.used = pdfPageHeight - pdfFooterHeight - 10
	f.keep(50)
	if len(added) != 1 {
		t.Fatalf("filler rows = %d, want 1 to force a new page", len(added))
	}
	f.push(pdfSectionHeight, row.New(pdfSectionHeight))
	if want := pdfLogoHeight + pdfHeadGap + pdfSectionHeight; f.used != want {
		t.Fatalf("used = %v, want %v on the new page", f.used, want)
	}
}

func TestResultsPDFFlowsCategoriesOnSharedPages(t *testing.T) {
	d := sampleResults()
	d.NewShooters = nil
	out, err := ResultsPDF(d)
	if err != nil {
		t.Fatal(err)
	}
	if got := pageCount(out); got != 1 {
		t.Fatalf("pages = %d, want short categories on one page", got)
	}
}
