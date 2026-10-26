package report

import (
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
	"github.com/sollie/matchmark/internal/repository"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	pdfFamily     = "go"
	pdfFontSize   = 9.0
	pdfRowHeight  = 6.5
	pdfLogoHeight = 18.0
)

var (
	pdfHeaderFill = &props.Color{Red: 238, Green: 238, Blue: 238}
	pdfMuted      = &props.Color{Red: 92, Green: 92, Blue: 92}
)

type ResultsPDFData struct {
	ClubName  string
	Logo      []byte
	LogoType  string
	Generated time.Time
	Results   Results
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
	vis := d.Results.Visibility
	columns, cells := resultsColumns(d.Results.Stages, vis)
	newColumns, newCells := newShooterColumns(vis)
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
		WithTitle(d.Results.Competition.Name, true).
		WithSequentialMode().
		Build()
	m := maroto.New(cfg)
	if err := m.RegisterHeader(headerRows(d, grid)...); err != nil {
		return nil, err
	}
	if err := m.RegisterFooter(footerRow(d, grid)); err != nil {
		return nil, err
	}

	type newShooter struct {
		category string
		row      Row
	}
	var newShooters []newShooter
	for _, c := range d.Results.Categories {
		for _, r := range c.New {
			newShooters = append(newShooters, newShooter{c.Name, r})
		}
		p := page.New()
		p.Add(sectionRow(c.Name, grid), tableHeader(columns, grid))
		for _, r := range c.Rows {
			p.Add(tableRow(columns, cells(r), grid, false))
		}
		m.AddPages(p)
	}
	if len(d.Results.Categories) == 0 {
		m.AddPages(page.New().Add(textRow("No results.", grid, props.Text{Size: pdfFontSize, Color: pdfMuted})))
	}
	if len(newShooters) > 0 {
		p := page.New()
		p.Add(sectionRow("New shooters", grid), tableHeader(newColumns, grid))
		for _, s := range newShooters {
			p.Add(tableRow(newColumns, newCells(s.category, s.row), grid, false))
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

func resultsColumns(stages []int, v repository.Visibility) ([]pdfColumn, func(Row) []string) {
	columns := []pdfColumn{{"#", 2, true}}
	if v.Names {
		columns = append(columns, pdfColumn{"Shooter", 7, false}, pdfColumn{"WAID", 3, false})
	}
	if v.Clubs {
		columns = append(columns, pdfColumn{"Club", 6, false})
	}
	if v.Countries {
		columns = append(columns, pdfColumn{"Country", 3, false})
	}
	if v.StageScores {
		for _, n := range stages {
			columns = append(columns, pdfColumn{"S" + strconv.Itoa(n), 2, true})
		}
	}
	columns = append(columns, pdfColumn{"Total", 3, true})
	if v.RingCounts {
		columns = append(columns, pdfColumn{"X", 2, true})
	}
	return columns, func(r Row) []string {
		cells := []string{strconv.Itoa(r.Rank)}
		if v.Names {
			cells = append(cells, r.Shooter.FirstName+" "+r.Shooter.LastName, r.Shooter.WAID)
		}
		if v.Clubs {
			cells = append(cells, r.Shooter.Club)
		}
		if v.Countries {
			cells = append(cells, r.Shooter.Country)
		}
		if v.StageScores {
			for _, s := range r.Stages {
				cells = append(cells, stageScoreCell(s))
			}
		}
		cells = append(cells, strconv.Itoa(r.Total))
		if v.RingCounts {
			cells = append(cells, strconv.Itoa(r.XCount))
		}
		return cells
	}
}

func stageScoreCell(s Stage) string {
	if !s.Accepted {
		return ""
	}
	return strconv.Itoa(s.Score)
}

func newShooterColumns(v repository.Visibility) ([]pdfColumn, func(string, Row) []string) {
	columns := []pdfColumn{{"Shooter", 8, false}, {"Category", 4, false}}
	if v.Clubs {
		columns = append(columns, pdfColumn{"Club", 6, false})
	}
	if v.Countries {
		columns = append(columns, pdfColumn{"Country", 3, false})
	}
	columns = append(columns, pdfColumn{"Total", 3, true})
	if v.RingCounts {
		columns = append(columns, pdfColumn{"X", 2, true})
	}
	return columns, func(category string, r Row) []string {
		cells := []string{r.Shooter.FirstName + " " + r.Shooter.LastName, category}
		if v.Clubs {
			cells = append(cells, r.Shooter.Club)
		}
		if v.Countries {
			cells = append(cells, r.Shooter.Country)
		}
		cells = append(cells, strconv.Itoa(r.Total))
		if v.RingCounts {
			cells = append(cells, strconv.Itoa(r.XCount))
		}
		return cells
	}
}

func headerRows(d ResultsPDFData, grid int) []core.Row {
	logoCols := 0
	var cols []core.Col
	if ext, ok := logoExtension(d.LogoType); ok && len(d.Logo) > 0 {
		logoCols = max(grid/6, 1)
		cols = append(cols, image.NewFromBytesCol(logoCols, d.Logo, ext, props.Rect{Center: true, Percent: 100}))
	}
	title := d.ClubName
	if title == "" {
		title = "Results"
	}
	c := d.Results.Competition
	info := c.Name + "  ·  " + c.Date + "  ·  " + d.Results.CourseLabel
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
	return row.New(8).Add(col.New(grid).Add(text.New(
		"Generated "+d.Generated.UTC().Format("2006-01-02 15:04")+" UTC",
		props.Text{Size: 8, Align: align.Right, Top: 3, Color: pdfMuted},
	)))
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
	r := row.New(pdfRowHeight).Add(cols...)
	if header {
		r.WithStyle(&props.Cell{BackgroundColor: pdfHeaderFill})
	}
	return r
}
