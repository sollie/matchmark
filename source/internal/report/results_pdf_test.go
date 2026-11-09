package report

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"reflect"
	"regexp"
	"slices"
	"strings"
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
			{Category: "Revolver 1500", Groups: []ResultsPDFGroup{{Rows: []ResultsPDFRow{row(1, "Ørn"), row(2, "Berg")}}}},
			{Category: "Pistol 1500", Groups: []ResultsPDFGroup{{Rows: []ResultsPDFRow{row(1, "Dahl")}}}},
		},
		NewShooters: []ResultsPDFNewShooter{{LastName: "Ny", FirstName: "Nina", Category: "Pistol 1500", Club: ptr("Łódź SK"), Total: 170, XCount: ptr(4)}},
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
		for j := range d.Categories[i].Groups[0].Rows {
			r := &d.Categories[i].Groups[0].Rows[j]
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
		r := d.Categories[0].Groups[0].Rows[0]
		r.Rank = i
		r.LastName = ptr(fmt.Sprintf("Shooter%d", i))
		rows = append(rows, r)
	}
	d.Categories = []ResultsPDFCategory{{Category: "Revolver 1500", Groups: []ResultsPDFGroup{{Rows: rows}}}}
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

func TestResultsColumnsUseSegmentHeadings(t *testing.T) {
	d := sampleResults()
	d.Headings = []string{"1-2", "3-4"}
	columns, _ := resultsColumns(d)
	var titles []string
	for _, c := range columns {
		titles = append(titles, c.title)
	}
	for _, want := range d.Headings {
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
	d.Categories[0].Groups[0].Rows = append(d.Categories[0].Groups[0].Rows, ResultsPDFRow{
		LastName: ptr("Slow"), FirstName: ptr("Sam"), Club: ptr("Łódź SK"), Country: ptr("NOR"), DNF: true,
		Segments: []ResultsPDFSegment{{Segment: 1, Score: ptr(80)}, {Segment: 2}},
	})
	_, cells := resultsColumns(d)
	got := cells(d.Categories[0].Groups[0].Rows[2])
	want := []string{"", "", "Sam Slow", "Łódź SK", "NOR", "80", "", "", "DNF"}
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

func course150() ResultsPDFData {
	scores := []int{229, 167, 205, 227, 291, 259}
	var segs []ResultsPDFSegment
	for i, v := range scores {
		segs = append(segs, ResultsPDFSegment{Segment: i + 1, Score: ptr(v)})
	}
	d := sampleResults()
	d.Segments = []int{1, 2, 3, 4, 5, 6}
	d.Headings = []string{"1", "2", "3", "4", "5:1", "5:2"}
	d.SumGroups = []ResultsPDFSumGroup{{Heading: "1-4", Segments: []int{1, 2, 3, 4}}, {Heading: "5", Segments: []int{5, 6}}}
	r := ResultsPDFRow{Rank: 1, LastName: ptr("Ørn"), FirstName: ptr("Åse"), WAID: ptr("1001"), Club: ptr("Łódź SK"), Country: ptr("NOR"),
		Classification: ptr("Master"), Segments: segs, Total: 1378, XCount: ptr(9)}
	d.Categories = []ResultsPDFCategory{{Category: "Revolver 1500", Groups: []ResultsPDFGroup{{Class: "Master", Rows: []ResultsPDFRow{r}}}}}
	return d
}

func TestResultsColumnsAddMatchSumsForCourse150(t *testing.T) {
	d := course150()
	columns, cells := resultsColumns(d)
	var titles []string
	for _, c := range columns {
		titles = append(titles, c.title)
	}
	wantTitles := []string{"Rank", "WAID", "Name", "Club", "Country", "Classification", "1", "2", "3", "4", "5:1", "5:2", "Match Sum 1-4", "Match Sum 5", "X", "Total"}
	if !slices.Equal(titles, wantTitles) {
		t.Fatalf("titles %q, want %q", titles, wantTitles)
	}
	got := cells(d.Categories[0].Groups[0].Rows[0])
	want := []string{"1", "1001", "Åse Ørn", "Łódź SK", "NOR", "Master", "229", "167", "205", "227", "291", "259", "828", "550", "9", "1378"}
	if !slices.Equal(got, want) {
		t.Fatalf("cells %q, want %q", got, want)
	}
}

func TestResultsColumnsHideClassificationRemarksAndSums(t *testing.T) {
	d := course150()
	r := &d.Categories[0].Groups[0].Rows[0]
	r.Classification = nil
	for i := range r.Segments {
		r.Segments[i].Score = nil
	}
	columns, _ := resultsColumns(d)
	for _, c := range columns {
		if c.title == "Classification" || c.title == "Remarks" || c.title == "Match Sum 1-4" {
			t.Fatalf("column %q must be hidden", c.title)
		}
	}
}

func TestResultsPDFHeaderBlocks(t *testing.T) {
	d := sampleResults()
	blocks := d.headerBlocks()
	if len(blocks[1]) != 0 {
		t.Fatalf("officials %v, want none when unset", blocks[1])
	}
	d.Competition, d.Referee, d.RangeOfficer, d.Range = "Autumn Open", "Rita Ref", "Ola Officer", "Åsane Range"
	blocks = d.headerBlocks()
	want := [][]headerField{
		{{"Competition", "Autumn Open"}, {"Match", "Autumn Match"}, {"Course", "WA1500 150"}},
		{{"Referee", "Rita Ref"}, {"Range officer", "Ola Officer"}, {"Range", "Åsane Range"}},
		{{"Match date", "2026-10-24"}, {"Participants", "3"}},
	}
	if !reflect.DeepEqual(blocks, want) {
		t.Fatalf("blocks %v, want %v", blocks, want)
	}
}

var pdfMediaBox = regexp.MustCompile(`/MediaBox \[0 0 ([0-9.]+) ([0-9.]+)\]`)

func TestResultsPDFIsLandscapeA4(t *testing.T) {
	out, err := ResultsPDF(course150())
	if err != nil {
		t.Fatal(err)
	}
	m := pdfMediaBox.FindSubmatch(out)
	if m == nil {
		t.Fatal("no media box")
	}
	w, h := string(m[1]), string(m[2])
	if !strings.HasPrefix(w, "841") || !strings.HasPrefix(h, "595") {
		t.Fatalf("media box %s x %s, want landscape A4", w, h)
	}
}

func TestResultsPDFWithClassGroupsRestartsRanks(t *testing.T) {
	d := course150()
	second := d.Categories[0].Groups[0].Rows[0]
	d.NewShooters = nil
	d.Categories[0].Groups = append(d.Categories[0].Groups, ResultsPDFGroup{Class: "Marksman", Rows: []ResultsPDFRow{second}})
	if got := d.participants(); got != 2 {
		t.Fatalf("participants = %d", got)
	}
	if got := categoryTitle(d.Categories[0]); got != "Revolver 1500  ·  2 shooters" {
		t.Fatalf("title %q", got)
	}
	out, err := ResultsPDF(d)
	if err != nil {
		t.Fatal(err)
	}
	if got := pageCount(out); got != 1 {
		t.Fatalf("pages = %d", got)
	}
}
