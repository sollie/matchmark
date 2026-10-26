package report

import (
	"bytes"
	"encoding/csv"
	"strconv"
	"strings"
)

var csvHeader = []string{"category", "lane", "last_name", "first_name", "waid", "withdrawn", "stage", "X", "10", "9", "8", "7", "0", "miss", "total", "accepted", "scorer_lane", "scorer_name"}

func safeCell(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}

func CSV(res Results) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("\ufeff")
	w := csv.NewWriter(&buf)
	if err := w.Write(csvHeader); err != nil {
		return nil, err
	}
	for _, cat := range res.Categories {
		for _, e := range sortedRows(cat.Rows) {
			for _, s := range e.Stages {
				c := s.Counts
				scorerLane := ""
				if s.ScorerLane != nil {
					scorerLane = strconv.Itoa(*s.ScorerLane)
				}
				rec := []string{
					safeCell(cat.Name), strconv.Itoa(e.Lane), safeCell(e.Shooter.LastName), safeCell(e.Shooter.FirstName), safeCell(e.Shooter.WAID),
					strconv.FormatBool(e.Withdrawn), strconv.Itoa(s.Number),
					strconv.Itoa(c.X), strconv.Itoa(c.Ten), strconv.Itoa(c.Nine), strconv.Itoa(c.Eight),
					strconv.Itoa(c.Seven), strconv.Itoa(c.Zero), strconv.Itoa(c.Miss),
					strconv.Itoa(s.Score), strconv.FormatBool(s.Accepted), scorerLane, safeCell(s.ScorerName),
				}
				if err := w.Write(rec); err != nil {
					return nil, err
				}
			}
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}
