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
	pdfFamily        = "go"
	pdfFontSize      = 7.0
	pdfMutedSize     = 6.0
	pdfLineHeight    = 3.0
	pdfPadY          = 0.6
	pdfRowHeight     = 4.2
	pdfSectionHeight = 8.0
	pdfLogoHeight    = 18.0
	pdfHeadGap       = 2.0
	pdfMargin        = 12.0
	pdfFooterHeight  = 8.0
	pdfPageHeight    = 297.0 - 2*pdfMargin
	pdfKeepRows      = 2
	pdfDNF           = "DNF"
)

//go:embed matchmark-icon.png
var matchMarkIcon []byte

var (
	pdfHeaderFill = &props.Color{Red: 238, Green: 238, Blue: 238}
	pdfMuted      = &props.Color{Red: 92, Green: 92, Blue: 92}
	pdfPodiumFill = &props.Color{Red: 255, Green: 248, Blue: 225}
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
	DNF       bool
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
	muted  bool
	flex   bool
	equal  bool
}

func ResultsPDF(d ResultsPDFData) ([]byte, error) {
	fonts, err := fontrepository.New().
		AddUTF8FontFromBytes(pdfFamily, fontstyle.Normal, goregular.TTF).
		AddUTF8FontFromBytes(pdfFamily, fontstyle.Bold, gobold.TTF).
		Load()
	if err != nil {
		return nil, err
	}
	ms, err := newMeasurer()
	if err != nil {
		return nil, err
	}
	columns, cells := resultsColumns(d)
	var rowCells [][]string
	for _, c := range d.Categories {
		for _, r := range c.Rows {
			rowCells = append(rowCells, cells(r))
		}
	}
	ms.sizeColumns(columns, rowCells)
	newColumns, newCells := newShooterColumns(d)
	var newRowCells [][]string
	for _, n := range d.NewShooters {
		newRowCells = append(newRowCells, newCells(n))
	}
	ms.sizeColumns(newColumns, newRowCells)
	grid := pdfGrid
	cfg := config.NewBuilder().
		WithPageSize(pagesize.A4).
		WithOrientation(orientation.Vertical).
		WithLeftMargin(12).WithRightMargin(12).WithTopMargin(12).WithBottomMargin(12).
		WithMaxGridSize(grid).
		WithCustomFonts(fonts).
		WithPageNumber(props.PageNumber{
			Pattern: "Page {current} of {total}", Place: props.RightBottom,
			Family: pdfFamily, Size: 8, Color: pdfMuted,
		}).
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

	flow := &pdfFlow{add: m.AddRows, used: pdfLogoHeight + pdfHeadGap}
	header, headerHeight := tableHeader(ms, columns, grid)
	for _, c := range d.Categories {
		flow.keep(pdfSectionHeight + headerHeight + pdfKeepRows*pdfRowHeight)
		flow.push(pdfSectionHeight, sectionRow(categoryTitle(c), grid))
		flow.push(headerHeight, header)
		for _, r := range c.Rows {
			tr, height := tableRow(ms, columns, cells(r), grid, false, r.Rank >= 1 && r.Rank <= 3 && !r.DNF)
			flow.push(height, tr)
		}
		flow.push(pdfRowHeight, row.New(pdfRowHeight))
	}
	if len(d.Categories) == 0 {
		m.AddPages(page.New().Add(textRow("No results.", grid, props.Text{Size: pdfFontSize, Color: pdfMuted})))
	}
	if len(d.NewShooters) > 0 {
		p := page.New()
		newHeader, _ := tableHeader(ms, newColumns, pdfGrid)
		p.Add(sectionRow("New shooters", grid), newHeader)
		for _, cells := range newRowCells {
			tr, _ := tableRow(ms, newColumns, cells, grid, false, false)
			p.Add(tr)
		}
		m.AddPages(p)
	}

	doc, err := m.Generate()
	if err != nil {
		return nil, err
	}
	return doc.GetBytes(), nil
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
	for _, c := range d.Categories {
		for _, r := range c.Rows {
			if !r.DNF {
				first = r
				break
			}
		}
	}
	showNames := found && first.LastName != nil
	showClubs := found && first.Club != nil
	showCountries := found && first.Country != nil
	showSegments := found && len(first.Segments) > 0 && first.Segments[0].Score != nil
	showX := found && first.XCount != nil

	columns := []pdfColumn{{title: "#", right: true}}
	if showNames {
		columns = append(columns, pdfColumn{title: "Shooter", flex: true}, pdfColumn{title: "WAID", muted: true})
	}
	if showClubs {
		columns = append(columns, pdfColumn{title: "Club", flex: true})
	}
	if showCountries {
		columns = append(columns, pdfColumn{title: "Country"})
	}
	if showSegments {
		for i, n := range d.Segments {
			title := "S" + strconv.Itoa(n)
			if i < len(d.SegmentNames) {
				title = d.SegmentNames[i]
			}
			columns = append(columns, pdfColumn{title: title, right: true, equal: true})
		}
	}
	columns = append(columns, pdfColumn{title: "Total", right: true})
	if showX {
		columns = append(columns, pdfColumn{title: "X", right: true})
	}
	return columns, func(r ResultsPDFRow) []string {
		cells := []string{strconv.Itoa(r.Rank)}
		if r.DNF {
			cells[0] = ""
		}
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
		if r.DNF {
			cells = append(cells, pdfDNF)
		} else {
			cells = append(cells, strconv.Itoa(r.Total))
		}
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
	columns := []pdfColumn{{title: "Shooter", flex: true}, {title: "Category"}}
	if showClubs {
		columns = append(columns, pdfColumn{title: "Club", flex: true})
	}
	if showCountries {
		columns = append(columns, pdfColumn{title: "Country"})
	}
	columns = append(columns, pdfColumn{title: "Total", right: true})
	if showX {
		columns = append(columns, pdfColumn{title: "X", right: true})
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
	info := d.Name + "  ·  " + d.Date + "  ·  " + d.CourseLabel + "  ·  " + shooterCount(d.participants())
	textCol := col.New(grid-logoCols).Add(
		text.New(title, props.Text{Size: 16, Style: fontstyle.Bold, Top: 2, Left: 2}),
		text.New(info, props.Text{Size: 10, Top: 10, Left: 2, Color: pdfMuted}),
	)
	cols = append(cols, textCol)
	return []core.Row{
		row.New(pdfLogoHeight).Add(cols...),
		row.New(pdfHeadGap),
	}
}

const RepoURL = "https://github.com/sollie/matchmark"

func footerRow(d ResultsPDFData, grid int) core.Row {
	iconCols := max(grid/16, 1)
	textCols := max(grid/2, 1)
	rest := max(grid-iconCols-textCols, 1)
	muted := props.Text{Size: 8, Top: 3, Color: pdfMuted}
	link := muted
	repoURL := RepoURL
	link.Hyperlink = &repoURL
	right := muted
	right.Align = align.Right
	return row.New(8).Add(
		image.NewFromBytesCol(iconCols, matchMarkIcon, extension.Png, props.Rect{Center: true, Percent: 80}),
		col.New(textCols).Add(text.New("Scored with MatchMark · "+RepoURL, link)),
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
	return row.New(pdfSectionHeight).Add(col.New(grid).Add(text.New(title, props.Text{Size: 10, Style: fontstyle.Bold, Top: 2})))
}

func tableHeader(ms *measurer, columns []pdfColumn, grid int) (core.Row, float64) {
	titles := make([]string, len(columns))
	for i, c := range columns {
		titles[i] = c.title
	}
	return tableRow(ms, columns, titles, grid, true, false)
}

func tableRow(ms *measurer, columns []pdfColumn, cells []string, grid int, header, podium bool) (core.Row, float64) {
	cols := make([]core.Col, 0, len(columns))
	lines := 1
	for i, c := range columns {
		width := float64(c.weight)*pdfUnit - 2*pdfCellPadX
		maxLines := pdfMaxLines
		if header {
			maxLines = pdfHeaderLines
		}
		size := c.fontSize()
		if header {
			size = pdfFontSize
		}
		wrapped := ms.wrap(cells[i], size, header, width, maxLines)
		lines = max(lines, len(wrapped))
		texts := make([]core.Component, len(wrapped))
		for n, line := range wrapped {
			p := props.Text{Top: pdfPadY + float64(n)*pdfLineHeight, Left: pdfCellPadX, Right: pdfCellPadX, Size: size}
			if header {
				p.Style = fontstyle.Bold
			}
			if c.right {
				p.Align = align.Right
			}
			if c.muted && !header {
				p.Color = pdfMuted
			}
			texts[n] = text.New(line, p)
		}
		cols = append(cols, col.New(c.weight).Add(texts...))
	}
	height := max(pdfRowHeight, 2*pdfPadY+float64(lines)*pdfLineHeight)
	r := row.New(height).Add(cols...)
	if header {
		r.WithStyle(&props.Cell{BackgroundColor: pdfHeaderFill})
	} else if podium {
		r.WithStyle(&props.Cell{BackgroundColor: pdfPodiumFill})
	}
	return r, height
}

func (d ResultsPDFData) participants() int {
	n := 0
	for _, c := range d.Categories {
		n += len(c.Rows)
	}
	return n
}

func shooterCount(n int) string {
	if n == 1 {
		return "1 shooter"
	}
	return strconv.Itoa(n) + " shooters"
}

func categoryTitle(c ResultsPDFCategory) string {
	return c.Category + "  ·  " + shooterCount(len(c.Rows))
}

type pdfFlow struct {
	add  func(...core.Row)
	used float64
}

func (f *pdfFlow) push(height float64, r core.Row) {
	if f.used+height > pdfPageHeight-pdfFooterHeight {
		f.used = pdfLogoHeight + pdfHeadGap
	}
	f.used += height
	f.add(r)
}

func (f *pdfFlow) keep(height float64) {
	limit := pdfPageHeight - pdfFooterHeight
	if f.used+height <= limit {
		return
	}
	if gap := limit - f.used - 0.05; gap > 0 {
		f.add(row.New(gap))
	}
	f.used = limit
}
