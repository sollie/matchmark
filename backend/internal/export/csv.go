package export

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/sollie/matchmark/internal/repository"
)

func WriteCSV(w io.Writer, shooters []repository.ExportShooterRow) error {
	cw := csv.NewWriter(w)

	header := []string{
		"Name", "WA-ID", "Club", "Discipline", "Weapon Class",
		"Total Score", "X Count",
		"X", "10", "9", "8", "7", "0", "M",
	}
	if err := cw.Write(header); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}

	for _, s := range shooters {
		club := ""
		if s.Club != nil {
			club = *s.Club
		}
		row := []string{
			s.Name,
			s.WAID,
			club,
			s.Discipline,
			s.WeaponClass,
			fmt.Sprintf("%d", s.TotalScore),
			fmt.Sprintf("%d", s.XCount),
			fmt.Sprintf("%d", s.RingX),
			fmt.Sprintf("%d", s.Ring10),
			fmt.Sprintf("%d", s.Ring9),
			fmt.Sprintf("%d", s.Ring8),
			fmt.Sprintf("%d", s.Ring7),
			fmt.Sprintf("%d", s.Ring0),
			fmt.Sprintf("%d", s.RingM),
		}
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}

	cw.Flush()
	return cw.Error()
}
