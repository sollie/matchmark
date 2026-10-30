package report

import (
	"bytes"
	_ "embed"
	"fmt"
	"sort"

	"github.com/xuri/excelize/v2"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
)

//go:embed template.xlsx
var template []byte

var categories = rulebook.Categories()

const (
	firstKnownRow     = 3
	knownCapacity     = 8
	newShootersOffset = 12
)

type Data struct {
	Match   repository.Match
	Entries []repository.ReportEntry
}

func total(e repository.ReportEntry) int {
	sum := 0
	for _, s := range e.Segments {
		sum += s.Counts.Score()
	}
	return sum
}

func sortedEntries(entries []repository.ReportEntry) []repository.ReportEntry {
	out := append([]repository.ReportEntry(nil), entries...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Category != b.Category {
			return categoryIndex(a.Category) < categoryIndex(b.Category)
		}
		if a.Shooter.LastName != b.Shooter.LastName {
			return a.Shooter.LastName < b.Shooter.LastName
		}
		if a.Shooter.FirstName != b.Shooter.FirstName {
			return a.Shooter.FirstName < b.Shooter.FirstName
		}
		return a.MatchID < b.MatchID
	})
	return out
}

func categoryIndex(category string) int {
	for i, c := range categories {
		if c == category {
			return i
		}
	}
	return len(categories)
}

func Workbook(d Data) ([]byte, error) {
	return build(d.Match.Date, d.Match.Name, d.Entries)
}

func CombinedWorkbook(c repository.Competition, entries []repository.ReportEntry) ([]byte, error) {
	return build(c.StartDate, c.Name, entries)
}

func build(date, name string, all []repository.ReportEntry) ([]byte, error) {
	f, err := excelize.OpenReader(bytes.NewReader(template))
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	sheets := f.GetSheetList()
	if len(sheets) < len(categories) {
		return nil, fmt.Errorf("template has %d sheets, need %d", len(sheets), len(categories))
	}
	entries := sortedEntries(all)
	for i, category := range categories {
		var known, fresh []repository.ReportEntry
		for _, e := range entries {
			if e.Category != category {
				continue
			}
			if e.Shooter.WAID != "" {
				known = append(known, e)
			} else {
				fresh = append(fresh, e)
			}
		}
		if err := fillSheet(f, sheets[i], date, name, known, fresh); err != nil {
			return nil, err
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func fillSheet(f *excelize.File, sheet, date, name string, known, fresh []repository.ReportEntry) error {
	if err := f.SetCellValue(sheet, "A1", date); err != nil {
		return err
	}
	if err := f.SetCellValue(sheet, "B1", name); err != nil {
		return err
	}
	extra := len(known) - knownCapacity
	if extra > 0 {
		if err := f.InsertRows(sheet, firstKnownRow+knownCapacity, extra); err != nil {
			return err
		}
		if err := copyRowStyle(f, sheet, firstKnownRow+knownCapacity-1, firstKnownRow+knownCapacity, extra, 4); err != nil {
			return err
		}
	} else {
		extra = 0
	}
	for i, e := range known {
		row := firstKnownRow + i
		if err := setRow(f, sheet, row, 1, e.Shooter.WAID, total(e)); err != nil {
			return err
		}
	}
	firstNewRow := newShootersOffset + extra + 2
	if err := f.SetCellValue(sheet, cellName(6, firstNewRow), nil); err != nil {
		return err
	}
	if len(fresh) > 3 {
		if err := copyRowStyle(f, sheet, firstNewRow, firstNewRow+3, len(fresh)-3, 7); err != nil {
			return err
		}
	}
	for i, e := range fresh {
		if err := setRow(f, sheet, firstNewRow+i, 1, e.Shooter.LastName, e.Shooter.FirstName, "", e.Shooter.Country, e.Shooter.Club, "nn", total(e)); err != nil {
			return err
		}
	}
	return nil
}

func setRow(f *excelize.File, sheet string, row, col int, values ...any) error {
	for i, v := range values {
		if err := f.SetCellValue(sheet, cellName(col+i, row), v); err != nil {
			return err
		}
	}
	return nil
}

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

func copyRowStyle(f *excelize.File, sheet string, srcRow, dstRow, count, cols int) error {
	for c := 1; c <= cols; c++ {
		style, err := f.GetCellStyle(sheet, cellName(c, srcRow))
		if err != nil {
			return err
		}
		for r := dstRow; r < dstRow+count; r++ {
			if err := f.SetCellStyle(sheet, cellName(c, r), cellName(c, r), style); err != nil {
				return err
			}
		}
	}
	return nil
}
