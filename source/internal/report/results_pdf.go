package report

import (
	_ "embed"
	"strconv"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/page"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/extension"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/orientation"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/fontrepository"
	"github.com/johnfercher/maroto/v2/pkg/props"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	pdfFamily       = "go"
	pdfFontSize     = 9.0
	pdfRowHeight    = 6.5
	pdfHeaderHeight = 13.0
	pdfLogoHeight   = 18.0
)

//go:embed matchmark-icon.png
var matchMarkIcon []byte

var (
	pdfHeaderFill = &props.Color{Red: 238, Green: 238, Blue: 238}
	pdfMuted      = &props.Color{Red: 92, Green: 92, Blue: 92}
)

type ResultsPDFSegment struct {
	Segment int
	Score   *int
}

type ResultsPDFRow struct {
	Rank      int
	LastName  *string
	FirstName *string
	WAID      *string
	Club      *string
	Country   *string
	Segments  []ResultsPDFSegment
	Total     int
	XCount    *int
}

type ResultsPDFCategory struct {
	Category string
	Rows     []ResultsPDFRow
}

type ResultsPDFNewShooter struct {
	LastName  string
	FirstName string
	Category  string
	Club      *string
	Country   *string
	Total     int
	XCount    *int
}

type ResultsPDFData struct {
	ClubName     string
	Logo         []byte
	LogoType     string
	Generated    time.Time
	Name         string
	Date         string
	CourseLabel  string
	Segments     []int
	SegmentNames []string
	Categories   []ResultsPDFCategory
	NewShooters  []ResultsPDFNewShooter
}

type pdfColumn struct {
	title  string
	weight int
	right  bool
}

func ResultsPDF(d ResultsPDFData) ([]byte, error) {
	fonts, err := fontrepository.New().
		AddUTF8FontFromBytes(pdfFamily, fontstyle.Normal, goregular.TTF).
		AddUTF8FontFromBytes(pdfFamily, fontstyle.Bold, gobold.TTF).
		Load()
	if err != nil {
		return nil, err
	}
	columns, cells := resultsColumns(d)
	newColumns, newCells := newShooterColumns(d)
	grid := columnGrid(columns)
	if g := columnGrid(newColumns); g > grid {
		grid = g
	}
	cfg := config.NewBuilder().
		WithPageSize(pagesize.A4).
		WithOrientation(orientation.Vertical).
		WithLeftMargin(12).WithRightMargin(12).WithTopMargin(12).WithBottomMargin(12).
		WithMaxGridSize(grid).
		WithCustomFonts(fonts).
		WithDefaultFont(&props.Font{Family: pdfFamily, Size: pdfFontSize}).
		WithCreationDate(d.Generated).
		WithTitle(d.Name, true).
		WithSequentialMode().
		Build()
	m := maroto.New(cfg)
	if err := m.RegisterHeader(headerRows(d, grid)...); err != nil {
		return nil, err
	}
	if err := m.RegisterFooter(footerRow(d, grid)); err != nil {
		return nil, err
	}

	for _, c := range d.Categories {
		p := page.New()
		p.Add(sectionRow(c.Category, grid), tableHeader(columns, grid))
		for _, r := range c.Rows {
			p.Add(tableRow(columns, cells(r), grid, false))
		}
		m.AddPages(p)
	}
	if len(d.Categories) == 0 {
		m.AddPages(page.New().Add(textRow("No results.", grid, props.Text{Size: pdfFontSize, Color: pdfMuted})))
	}
	if len(d.NewShooters) > 0 {
		p := page.New()
		p.Add(sectionRow("New shooters", grid), tableHeader(newColumns, grid))
		for _, s := range d.NewShooters {
			p.Add(tableRow(newColumns, newCells(s), grid, false))
		}
		m.AddPages(p)
	}

	doc, err := m.Generate()
	if err != nil {
		return nil, err
	}
	return doc.GetBytes(), nil
}

func columnGrid(columns []pdfColumn) int {
	n := 0
	for _, c := range columns {
		n += c.weight
	}
	return n
}

func resultsColumns(d ResultsPDFData) ([]pdfColumn, func(ResultsPDFRow) []string) {
	var first ResultsPDFRow
	found := false
	for _, c := range d.Categories {
		if len(c.Rows) > 0 {
			first, found = c.Rows[0], true
			break
		}
	}
	showNames := found && first.LastName != nil
	showClubs := found && first.Club != nil
	showCountries := found && first.Country != nil
	showSegments := found && len(first.Segments) > 0 && first.Segments[0].Score != nil
	showX := found && first.XCount != nil

	columns := []pdfColumn{{"#", 2, true}}
	if showNames {
		columns = append(columns, pdfColumn{"Shooter", 7, false}, pdfColumn{"WAID", 3, false})
	}
	if showClubs {
		columns = append(columns, pdfColumn{"Club", 6, false})
	}
	if showCountries {
		columns = append(columns, pdfColumn{"Country", 3, false})
	}
	if showSegments {
		for i, n := range d.Segments {
			title := "S" + strconv.Itoa(n)
			if i < len(d.SegmentNames) {
				title = d.SegmentNames[i]
			}
			columns = append(columns, pdfColumn{title, 4, true})
		}
	}
	columns = append(columns, pdfColumn{"Total", 3, true})
	if showX {
		columns = append(columns, pdfColumn{"X", 2, true})
	}
	return columns, func(r ResultsPDFRow) []string {
		cells := []string{strconv.Itoa(r.Rank)}
		if showNames {
			cells = append(cells, derefString(r.FirstName)+" "+derefString(r.LastName), derefString(r.WAID))
		}
		if showClubs {
			cells = append(cells, derefString(r.Club))
		}
		if showCountries {
			cells = append(cells, derefString(r.Country))
		}
		if showSegments {
			for _, s := range r.Segments {
				cells = append(cells, derefInt(s.Score))
			}
		}
		cells = append(cells, strconv.Itoa(r.Total))
		if showX {
			cells = append(cells, derefInt(r.XCount))
		}
		return cells
	}
}

func newShooterColumns(d ResultsPDFData) ([]pdfColumn, func(ResultsPDFNewShooter) []string) {
	showClubs := len(d.NewShooters) > 0 && d.NewShooters[0].Club != nil
	showCountries := len(d.NewShooters) > 0 && d.NewShooters[0].Country != nil
	showX := len(d.NewShooters) > 0 && d.NewShooters[0].XCount != nil
	columns := []pdfColumn{{"Shooter", 8, false}, {"Category", 4, false}}
	if showClubs {
		columns = append(columns, pdfColumn{"Club", 6, false})
	}
	if showCountries {
		columns = append(columns, pdfColumn{"Country", 3, false})
	}
	columns = append(columns, pdfColumn{"Total", 3, true})
	if showX {
		columns = append(columns, pdfColumn{"X", 2, true})
	}
	return columns, func(s ResultsPDFNewShooter) []string {
		cells := []string{s.FirstName + " " + s.LastName, s.Category}
		if showClubs {
			cells = append(cells, derefString(s.Club))
		}
		if showCountries {
			cells = append(cells, derefString(s.Country))
		}
		cells = append(cells, strconv.Itoa(s.Total))
		if showX {
			cells = append(cells, derefInt(s.XCount))
		}
		return cells
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt(n *int) string {
	if n == nil {
		return ""
	}
	return strconv.Itoa(*n)
}

func headerRows(d ResultsPDFData, grid int) []core.Row {
	logoCols := 0
	var cols []core.Col
	logoCols = max(grid/6, 1)
	logo, logoExt := matchMarkIcon, extension.Png
	if ext, ok := logoExtension(d.LogoType); ok && len(d.Logo) > 0 {
		logo, logoExt = d.Logo, ext
	}
	cols = append(cols, image.NewFromBytesCol(logoCols, logo, logoExt, props.Rect{Center: true, Percent: 100}))
	title := d.ClubName
	if title == "" {
		title = "MatchMark"
	}
	info := d.Name + "  ·  " + d.Date + "  ·  " + d.CourseLabel
	textCol := col.New(grid-logoCols).Add(
		text.New(title, props.Text{Size: 16, Style: fontstyle.Bold, Top: 2, Left: 2}),
		text.New(info, props.Text{Size: 10, Top: 10, Left: 2, Color: pdfMuted}),
	)
	cols = append(cols, textCol)
	return []core.Row{
		row.New(pdfLogoHeight).Add(cols...),
		row.New(2),
	}
}

func footerRow(d ResultsPDFData, grid int) core.Row {
	iconCols := max(grid/16, 1)
	textCols := max(grid/4, 1)
	rest := max(grid-iconCols-textCols, 1)
	muted := props.Text{Size: 8, Top: 3, Color: pdfMuted}
	right := muted
	right.Align = align.Right
	return row.New(8).Add(
		image.NewFromBytesCol(iconCols, matchMarkIcon, extension.Png, props.Rect{Center: true, Percent: 80}),
		col.New(textCols).Add(text.New("Made with MatchMark", muted)),
		col.New(rest).Add(text.New("Generated "+d.Generated.UTC().Format("2006-01-02 15:04")+" UTC", right)),
	)
}

func logoExtension(contentType string) (extension.Type, bool) {
	switch contentType {
	case "image/png":
		return extension.Png, true
	case "image/jpeg":
		return extension.Jpeg, true
	}
	return "", false
}

func textRow(s string, grid int, p props.Text) core.Row {
	return row.New(pdfRowHeight).Add(col.New(grid).Add(text.New(s, p)))
}

func sectionRow(title string, grid int) core.Row {
	return row.New(pdfRowHeight + 2).Add(col.New(grid).Add(text.New(title, props.Text{Size: 12, Style: fontstyle.Bold, Top: 1})))
}

func tableHeader(columns []pdfColumn, grid int) core.Row {
	titles := make([]string, len(columns))
	for i, c := range columns {
		titles[i] = c.title
	}
	return tableRow(columns, titles, grid, true)
}

func tableRow(columns []pdfColumn, cells []string, grid int, header bool) core.Row {
	used := columnGrid(columns)
	cols := make([]core.Col, 0, len(columns)+1)
	for i, c := range columns {
		p := props.Text{Top: 1.5, Left: 1, Right: 1, Size: pdfFontSize}
		if header {
			p.Style = fontstyle.Bold
		}
		if c.right {
			p.Align = align.Right
		}
		cols = append(cols, col.New(c.weight).Add(text.New(cells[i], p)))
	}
	if used < grid {
		cols = append(cols, col.New(grid-used))
	}
	height := pdfRowHeight
	if header {
		height = pdfHeaderHeight
	}
	r := row.New(height).Add(cols...)
	if header {
		r.WithStyle(&props.Cell{BackgroundColor: pdfHeaderFill})
	}
	return r
}
