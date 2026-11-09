package report

import (
	"bytes"
	"encoding/csv"
	"strconv"
	"strings"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
	"github.com/sollie/matchmark/internal/wa1500"
)

var csvHeader = []string{"category", "lane", "last_name", "first_name", "waid", "withdrawn", "segment", "segment_name", "X", "10", "9", "8", "7", "0", "miss", "total", "accepted", "scorer_lane", "scorer_name", "classification_at_match", "class", "club", "country"}

func safeCell(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}

func CSV(d Data) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("\ufeff")
	w := csv.NewWriter(&buf)
	if err := w.Write(csvHeader); err != nil {
		return nil, err
	}
	names := map[int]string{}
	if course, err := rulebook.CourseByKey(d.Match.Course); err == nil {
		for _, st := range course.Segments {
			names[st.Number] = st.Name
		}
	}
	for _, e := range sortedEntries(d.Entries) {
		id := e.Identity(e.Shooter.Club, e.Shooter.Country, e.Shooter.Classes)
		for _, s := range e.Segments {
			c := s.Counts
			scorerLane := ""
			if s.ScorerLane != nil {
				scorerLane = strconv.Itoa(*s.ScorerLane)
			}
			rec := []string{
				safeCell(e.Category), strconv.Itoa(e.Lane), safeCell(e.Shooter.LastName), safeCell(e.Shooter.FirstName), safeCell(e.Shooter.WAID),
				strconv.FormatBool(e.Withdrawn), strconv.Itoa(s.Number), names[s.Number],
				strconv.Itoa(c.X), strconv.Itoa(c.Ten), strconv.Itoa(c.Nine), strconv.Itoa(c.Eight),
				strconv.Itoa(c.Seven), strconv.Itoa(c.Zero), strconv.Itoa(c.Miss),
				strconv.Itoa(c.Score()), strconv.FormatBool(s.Status == repository.StatusAccepted), scorerLane, safeCell(s.ScorerName),
				classificationName(id.Classification), classificationName(id.Class), safeCell(id.Club), safeCell(id.Country),
			}
			if err := w.Write(rec); err != nil {
				return nil, err
			}
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func classificationName(v *int) string {
	if v == nil {
		return ""
	}
	return wa1500.Classification(*v).Name()
}
