package report

import (
	"strings"
	"testing"
)

func TestSizeColumnsFillPageAndKeepNumbersWhole(t *testing.T) {
	m, err := newMeasurer()
	if err != nil {
		t.Fatal(err)
	}
	cols := []pdfColumn{{title: "#", right: true}, {title: "Shooter", flex: true}, {title: "Total", right: true}}
	rows := [][]string{{"12", "Josefine Margreta Krogh Berntzen", "560"}}
	m.sizeColumns(cols, rows)
	sum := 0
	for _, c := range cols {
		sum += c.weight
	}
	if sum != pdfGrid {
		t.Fatalf("weights sum to %d, want %d", sum, pdfGrid)
	}
	for _, i := range []int{0, 2} {
		width := float64(cols[i].weight)*pdfUnit - 2*pdfCellPadX
		if got := m.wrap(rows[0][i], pdfFontSize, false, width, pdfMaxLines); len(got) != 1 {
			t.Fatalf("column %d wraps to %v", i, got)
		}
	}
}

func TestWrapEllipsisAfterMaxLines(t *testing.T) {
	m, err := newMeasurer()
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Repeat("Kristiansand ", 10)
	lines := m.wrap(text, pdfFontSize, false, 20, pdfMaxLines)
	if len(lines) != pdfMaxLines {
		t.Fatalf("got %d lines, want %d", len(lines), pdfMaxLines)
	}
	if !strings.HasSuffix(lines[len(lines)-1], pdfEllipsis) {
		t.Fatalf("last line %q has no ellipsis", lines[len(lines)-1])
	}
	for _, l := range lines {
		if m.width(l, pdfFontSize, false) > 20 {
			t.Fatalf("line %q is wider than the cell", l)
		}
	}
}

func TestHeaderMinFitsTwoLines(t *testing.T) {
	m, err := newMeasurer()
	if err != nil {
		t.Fatal(err)
	}
	w := m.headerMin("Match 5 · Stage 1+2", pdfFontSize)
	if got := m.wrap("Match 5 · Stage 1+2", pdfFontSize, true, w/pdfFitMargin, pdfHeaderLines+1); len(got) > pdfHeaderLines {
		t.Fatalf("header wraps to %d lines: %v", len(got), got)
	}
}
