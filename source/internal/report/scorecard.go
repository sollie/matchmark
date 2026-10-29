package report

import (
	"bytes"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"strconv"
	"strings"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
)

const (
	cardWidth       = 1200
	cardMargin      = 48
	logoHeight      = 96
	rowHeight       = 46
	headerRowHeight = 74
	adminScorer     = "Range admin"
)

var (
	colorInk    = color.RGBA{R: 0x1a, G: 0x1a, B: 0x1a, A: 0xff}
	colorMuted  = color.RGBA{R: 0x5c, G: 0x5c, B: 0x5c, A: 0xff}
	colorRule   = color.RGBA{R: 0xc8, G: 0xc8, B: 0xc8, A: 0xff}
	colorHeader = color.RGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff}
)

type ScorecardData struct {
	Match    repository.Match
	Entry    repository.ReportEntry
	Course   rulebook.Course
	ClubName string
	Logo     []byte
}

type cardFonts struct {
	regular font.Face
	bold    font.Face
	title   font.Face
	small   font.Face
}

func loadCardFonts() (cardFonts, error) {
	regular, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return cardFonts{}, err
	}
	bold, err := opentype.Parse(gobold.TTF)
	if err != nil {
		return cardFonts{}, err
	}
	face := func(f *opentype.Font, size float64) (font.Face, error) {
		return opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	}
	var cf cardFonts
	if cf.regular, err = face(regular, 24); err != nil {
		return cardFonts{}, err
	}
	if cf.bold, err = face(bold, 24); err != nil {
		return cardFonts{}, err
	}
	if cf.title, err = face(bold, 38); err != nil {
		return cardFonts{}, err
	}
	if cf.small, err = face(regular, 20); err != nil {
		return cardFonts{}, err
	}
	return cf, nil
}

type cardColumn struct {
	title string
	width int
	left  bool
}

func ringCounts(c repository.RingCounts) []int {
	return []int{c.X, c.Ten, c.Nine, c.Eight, c.Seven, c.Zero, c.Miss}
}

func ScorecardPNG(d ScorecardData) ([]byte, error) {
	fonts, err := loadCardFonts()
	if err != nil {
		return nil, err
	}
	grid := scorecardGrid(d)
	labelWidth, totalWidth := 140, 110
	segWidth := (cardWidth - 2*cardMargin - labelWidth - totalWidth) / max(len(d.Course.Segments), 1)
	columns := []cardColumn{{"", labelWidth, true}}
	for range d.Course.Segments {
		columns = append(columns, cardColumn{"", segWidth, false})
	}
	columns = append(columns, cardColumn{"", totalWidth, false})

	const infoRows = 3
	headerBottom := cardMargin + logoHeight + 24
	infoTop := headerBottom + 20
	tableTop := infoTop + infoRows*40 + 24
	height := tableTop + headerRowHeight + (len(grid)-1)*rowHeight + 36 + cardMargin
	img := image.NewRGBA(image.Rect(0, 0, cardWidth, height))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)

	textX := cardMargin
	if logo := scaledLogo(d.Logo); logo != nil {
		draw.Draw(img, image.Rect(cardMargin, cardMargin, cardMargin+logo.Bounds().Dx(), cardMargin+logo.Bounds().Dy()), logo, logo.Bounds().Min, draw.Over)
		textX += logo.Bounds().Dx() + 24
	}
	club := d.ClubName
	if club == "" {
		club = "Scorecard"
	}
	drawText(img, fonts.title, colorInk, textX, cardMargin+44, fitText(fonts.title, club, cardWidth-cardMargin-textX))
	drawText(img, fonts.regular, colorMuted, textX, cardMargin+84, fitText(fonts.regular, d.Match.Name+"  ·  "+d.Match.Date, cardWidth-cardMargin-textX))
	hline(img, cardMargin, cardWidth-cardMargin, headerBottom)

	shooter := d.Entry.Shooter
	name := shooter.FirstName + " " + shooter.LastName
	waid := shooter.WAID
	if waid == "" {
		waid = "-"
	}
	half := (cardWidth - 2*cardMargin) / 2
	infoLines := []struct{ label, value string }{
		{"Shooter", name}, {"WAID", waid},
		{"Category", d.Entry.Category}, {"Course", d.Course.Label},
		{"Lane", strconv.Itoa(d.Entry.Lane)}, {"Club", shooter.Club},
	}
	for i, line := range infoLines {
		x := cardMargin + (i%2)*half
		y := infoTop + (i/2)*40 + 28
		drawText(img, fonts.small, colorMuted, x, y, line.label)
		drawText(img, fonts.bold, colorInk, x+150, y, fitText(fonts.bold, line.value, half-170))
	}

	y := tableTop
	fillRect(img, image.Rect(cardMargin, y, cardWidth-cardMargin, y+headerRowHeight), colorHeader)
	drawHeaderRow(img, fonts, columns, y, grid[0])
	y += headerRowHeight - rowHeight
	for i, cells := range grid[1:] {
		y += rowHeight
		hline(img, cardMargin, cardWidth-cardMargin, y)
		face := fonts.regular
		switch {
		case i == len(grid)-4:
			face = fonts.bold
		case i >= len(grid)-3:
			face = fonts.small
		}
		drawRow(img, face, colorInk, columns, y, cells)
	}
	drawText(img, fonts.small, colorMuted, cardMargin, y+rowHeight+36, "Only accepted segments are counted.")

	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func scorecardGrid(d ScorecardData) [][]string {
	byNumber := map[int]repository.ReportSegment{}
	for _, s := range d.Entry.Segments {
		byNumber[s.Number] = s
	}
	rings := rulebook.RingValues()
	header := []string{""}
	body := make([][]string, len(rings)+3)
	for i, ring := range rings {
		body[i] = []string{ring}
	}
	score, scorer, shooter := len(rings), len(rings)+1, len(rings)+2
	body[score], body[scorer], body[shooter] = []string{"Score"}, []string{"Scorer"}, []string{"Shooter"}
	name := d.Entry.Shooter.FirstName + " " + d.Entry.Shooter.LastName
	var totals repository.RingCounts
	totalScore := 0
	for _, segment := range d.Course.Segments {
		header = append(header, segment.Name)
		body[shooter] = append(body[shooter], name)
		st, ok := byNumber[segment.Number]
		if !ok || st.Status != repository.StatusAccepted {
			for i := range rings {
				body[i] = append(body[i], "")
			}
			body[score] = append(body[score], "")
			body[scorer] = append(body[scorer], "")
			continue
		}
		for i, n := range ringCounts(st.Counts) {
			body[i] = append(body[i], strconv.Itoa(n))
		}
		body[score] = append(body[score], strconv.Itoa(st.Counts.Score()))
		body[scorer] = append(body[scorer], ScorerLabel(st))
		totals = addCounts(totals, st.Counts)
		totalScore += st.Counts.Score()
	}
	header = append(header, "Total")
	for i, n := range ringCounts(totals) {
		body[i] = append(body[i], strconv.Itoa(n))
	}
	body[score] = append(body[score], strconv.Itoa(totalScore))
	body[scorer] = append(body[scorer], "")
	body[shooter] = append(body[shooter], "")
	return append([][]string{header}, body...)
}

func drawHeaderRow(img *image.RGBA, fonts cardFonts, columns []cardColumn, top int, cells []string) {
	x := cardMargin
	for i, col := range columns {
		lines := strings.Split(cells[i], " · ")
		for j, line := range lines {
			text := fitText(fonts.bold, line, col.width-16)
			baseline := top + rowHeight - 14 + j*28 - (len(lines)-1)*14
			if col.left {
				drawText(img, fonts.bold, colorInk, x+8, baseline, text)
			} else {
				drawText(img, fonts.bold, colorInk, x+col.width-8-textWidth(fonts.bold, text), baseline, text)
			}
		}
		x += col.width
	}
}

func ScorerLabel(s repository.ReportSegment) string {
	if s.ScorerLane == nil {
		return adminScorer
	}
	return s.ScorerName
}

func addCounts(a, b repository.RingCounts) repository.RingCounts {
	return repository.RingCounts{
		X: a.X + b.X, Ten: a.Ten + b.Ten, Nine: a.Nine + b.Nine, Eight: a.Eight + b.Eight,
		Seven: a.Seven + b.Seven, Zero: a.Zero + b.Zero, Miss: a.Miss + b.Miss,
	}
}

func drawRow(img *image.RGBA, face font.Face, c color.Color, columns []cardColumn, top int, cells []string) {
	x := cardMargin
	baseline := top + rowHeight - 14
	for i, col := range columns {
		text := fitText(face, cells[i], col.width-16)
		if col.left {
			drawText(img, face, c, x+8, baseline, text)
		} else {
			drawText(img, face, c, x+col.width-8-textWidth(face, text), baseline, text)
		}
		x += col.width
	}
}

func scaledLogo(data []byte) image.Image {
	if len(data) == 0 {
		return nil
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	b := src.Bounds()
	if b.Dx() < 1 || b.Dy() < 1 {
		return nil
	}
	h := logoHeight
	w := b.Dx() * h / b.Dy()
	if w > 240 {
		w = 240
		h = b.Dy() * w / b.Dx()
	}
	if w < 1 || h < 1 {
		return nil
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}

func textWidth(face font.Face, s string) int {
	return font.MeasureString(face, s).Ceil()
}

func fitText(face font.Face, s string, max int) string {
	if textWidth(face, s) <= max {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		if candidate := string(runes) + "…"; textWidth(face, candidate) <= max {
			return candidate
		}
	}
	return ""
}

func drawText(img *image.RGBA, face font.Face, c color.Color, x, baseline int, s string) {
	d := font.Drawer{Dst: img, Src: image.NewUniform(c), Face: face, Dot: fixed.P(x, baseline)}
	d.DrawString(s)
}

func hline(img *image.RGBA, x0, x1, y int) {
	fillRect(img, image.Rect(x0, y, x1, y+1), colorRule)
}

func fillRect(img *image.RGBA, r image.Rectangle, c color.Color) {
	draw.Draw(img, r, image.NewUniform(c), image.Point{}, draw.Src)
}
