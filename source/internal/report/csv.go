package report

import (
	"bytes"
	"encoding/csv"
	"strconv"
)

var csvHeader = []string{"category", "lane", "last_name", "first_name", "waid", "withdrawn", "stage", "X", "10", "9", "8", "7", "0", "miss", "total", "locked", "scorer_lane", "scorer_name"}

func CSV(d Data) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(csvHeader); err != nil {
		return nil, err
	}
	for _, e := range sortedEntries(d.Entries) {
		for _, s := range e.Stages {
			c := s.Counts
			scorerLane := ""
			if s.ScorerLane != nil {
				scorerLane = strconv.Itoa(*s.ScorerLane)
			}
			rec := []string{
				e.Category, strconv.Itoa(e.Lane), e.Shooter.LastName, e.Shooter.FirstName, e.Shooter.WAID,
				strconv.FormatBool(e.Withdrawn), strconv.Itoa(s.Number),
				strconv.Itoa(c.X), strconv.Itoa(c.Ten), strconv.Itoa(c.Nine), strconv.Itoa(c.Eight),
				strconv.Itoa(c.Seven), strconv.Itoa(c.Zero), strconv.Itoa(c.Miss),
				strconv.Itoa(Score(c)), strconv.FormatBool(s.Locked), scorerLane, s.ScorerName,
			}
			if err := w.Write(rec); err != nil {
				return nil, err
			}
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}
