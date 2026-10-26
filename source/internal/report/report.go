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

func sortedRows(rows []Row) []Row {
	out := append([]Row(nil), rows...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Shooter.LastName != b.Shooter.LastName {
			return a.Shooter.LastName < b.Shooter.LastName
		}
		if a.Shooter.FirstName != b.Shooter.FirstName {
			return a.Shooter.FirstName < b.Shooter.FirstName
		}
		return a.Lane < b.Lane
	})
	return out
}

func Workbook(res Results) ([]byte, error) {
	f, err := excelize.OpenReader(bytes.NewReader(template))
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	sheets := f.GetSheetList()
	if len(sheets) < len(categories) {
		return nil, fmt.Errorf("template has %d sheets, need %d", len(sheets), len(categories))
	}
	byName := map[string]Category{}
	for _, c := range res.Categories {
		byName[c.Name] = c
	}
	for i, category := range categories {
		var known, fresh []Row
		for _, r := range sortedRows(byName[category].Rows) {
			if r.Shooter.WAID != "" {
				known = append(known, r)
			} else {
				fresh = append(fresh, r)
			}
		}
		if err := fillSheet(f, sheets[i], res.Competition, known, fresh); err != nil {
			return nil, err
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func fillSheet(f *excelize.File, sheet string, comp repository.Competition, known, fresh []Row) error {
	if err := f.SetCellValue(sheet, "A1", comp.Date); err != nil {
		return err
	}
	if err := f.SetCellValue(sheet, "B1", comp.Name); err != nil {
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
		if err := setRow(f, sheet, row, 1, e.Shooter.WAID, e.Total); err != nil {
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
		sex := "male"
		if e.Shooter.Sex == "F" {
			sex = "female"
		}
		if err := setRow(f, sheet, firstNewRow+i, 1, e.Shooter.LastName, e.Shooter.FirstName, sex, e.Shooter.Country, e.Shooter.Club, "nn", e.Total); err != nil {
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
