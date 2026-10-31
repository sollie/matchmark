package report

import (
	"math"
	"slices"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

const (
	pdfPageWidth   = 186.0
	pdfUnit        = 0.5
	pdfGrid        = int(pdfPageWidth / pdfUnit)
	pdfCellPadX    = 1.0
	pdfFitMargin   = 0.95
	pdfFlexMinMM   = 12.0
	pdfMaxLines    = 3
	pdfHeaderLines = 2
	pdfEllipsis    = "…"
)

type measurer struct {
	faces map[faceKey]font.Face
}

type faceKey struct {
	size float64
	bold bool
}

func newMeasurer() (*measurer, error) {
	m := &measurer{faces: map[faceKey]font.Face{}}
	for _, bold := range []bool{false, true} {
		ttf := goregular.TTF
		if bold {
			ttf = gobold.TTF
		}
		f, err := opentype.Parse(ttf)
		if err != nil {
			return nil, err
		}
		for _, size := range []float64{pdfFontSize, pdfMutedSize} {
			face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
			if err != nil {
				return nil, err
			}
			m.faces[faceKey{size, bold}] = face
		}
	}
	return m, nil
}

func (m *measurer) width(s string, size float64, bold bool) float64 {
	return float64(font.MeasureString(m.faces[faceKey{size, bold}], s)) / 64 * 25.4 / 72
}

func (m *measurer) wrap(s string, size float64, bold bool, widthMM float64, maxLines int) []string {
	limit := widthMM * pdfFitMargin
	var lines []string
	cur := ""
	for _, word := range strings.Fields(s) {
		for _, token := range m.chunks(word, size, bold, limit) {
			candidate := token
			if cur != "" {
				candidate = cur + " " + token
			}
			if cur == "" || m.width(candidate, size, bold) <= limit {
				cur = candidate
				continue
			}
			lines = append(lines, cur)
			cur = token
		}
	}
	if cur != "" || len(lines) == 0 {
		lines = append(lines, cur)
	}
	if len(lines) > maxLines {
		kept := lines[:maxLines]
		kept[maxLines-1] = m.truncate(strings.Join(lines[maxLines-1:], " "), size, bold, limit)
		return kept
	}
	return lines
}

func (m *measurer) chunks(word string, size float64, bold bool, limit float64) []string {
	var out []string
	runes := []rune(word)
	for m.width(string(runes), size, bold) > limit && len(runes) > 1 {
		n := 1
		for n < len(runes)-1 && m.width(string(runes[:n+1]), size, bold) <= limit {
			n++
		}
		out = append(out, string(runes[:n]))
		runes = runes[n:]
	}
	return append(out, string(runes))
}

func (m *measurer) truncate(s string, size float64, bold bool, limit float64) string {
	runes := []rune(s)
	for len(runes) > 0 && m.width(string(runes)+pdfEllipsis, size, bold) > limit {
		runes = runes[:len(runes)-1]
	}
	return strings.TrimRight(string(runes), " ") + pdfEllipsis
}

func (m *measurer) headerMin(title string, size float64) float64 {
	widest := 0.0
	for _, w := range strings.Fields(title) {
		widest = max(widest, m.width(w, size, true))
	}
	full := m.width(title, size, true)
	for w := widest; w < full; w += 0.25 {
		if len(m.wrap(title, size, true, w/pdfFitMargin, pdfHeaderLines+1)) <= pdfHeaderLines {
			return w
		}
	}
	return full
}

func (c pdfColumn) fontSize() float64 {
	if c.muted {
		return pdfMutedSize
	}
	return pdfFontSize
}

func (m *measurer) sizeColumns(columns []pdfColumn, rows [][]string) {
	natural := make([]float64, len(columns))
	for i, c := range columns {
		w := 0.0
		for _, r := range rows {
			w = max(w, m.width(r[i], c.fontSize(), false))
		}
		natural[i] = max(w, m.headerMin(c.title, pdfFontSize))/pdfFitMargin + 2*pdfCellPadX
	}
	equalise(columns, natural)
	widths := distribute(columns, natural, pdfPageWidth)
	units := toUnits(widths, pdfGrid)
	for i := range columns {
		columns[i].weight = units[i]
	}
}

func equalise(columns []pdfColumn, widths []float64) {
	widest := 0.0
	for i, c := range columns {
		if c.equal {
			widest = max(widest, widths[i])
		}
	}
	for i, c := range columns {
		if c.equal {
			widths[i] = widest
		}
	}
}

func distribute(columns []pdfColumn, natural []float64, avail float64) []float64 {
	out := slices.Clone(natural)
	total := 0.0
	flexTotal := 0.0
	flexRoom := 0.0
	for i, w := range natural {
		total += w
		if columns[i].flex {
			flexTotal += w
			flexRoom += max(w-pdfFlexMinMM, 0)
		}
	}
	spare := avail - total
	switch {
	case spare >= 0 && flexTotal > 0:
		for i := range out {
			if columns[i].flex {
				out[i] += spare * natural[i] / flexTotal
			}
		}
	case spare >= 0:
		for i := range out {
			out[i] += spare * natural[i] / total
		}
	case -spare <= flexRoom:
		level := flexLevel(columns, natural, flexTotal+spare)
		for i := range out {
			if columns[i].flex {
				out[i] = min(natural[i], level)
			}
		}
	default:
		scale := avail / total
		for i := range out {
			out[i] *= scale
		}
	}
	return out
}

func toUnits(widths []float64, grid int) []int {
	total := 0.0
	for _, w := range widths {
		total += w
	}
	units := make([]int, len(widths))
	used := 0
	widest := 0
	for i, w := range widths {
		units[i] = max(int(math.Ceil(w/total*float64(grid))), 1)
		used += units[i]
		if widths[i] > widths[widest] {
			widest = i
		}
	}
	units[widest] += grid - used
	return units
}

func flexLevel(columns []pdfColumn, natural []float64, budget float64) float64 {
	lo, hi := pdfFlexMinMM, 0.0
	for i, w := range natural {
		if columns[i].flex {
			hi = max(hi, w)
		}
	}
	for range 40 {
		mid := (lo + hi) / 2
		sum := 0.0
		for i, w := range natural {
			if columns[i].flex {
				sum += min(w, mid)
			}
		}
		if sum > budget {
			hi = mid
		} else {
			lo = mid
		}
	}
	return lo
}
