package export

import (
	"encoding/json"
	"io"

	"github.com/sollie/matchmark/internal/repository"
)

type ringCounts struct {
	X     int `json:"X"`
	Ten   int `json:"10"`
	Nine  int `json:"9"`
	Eight int `json:"8"`
	Seven int `json:"7"`
	Zero  int `json:"0"`
	M     int `json:"M"`
}

type stageJSON struct {
	StageNumber int        `json:"stage_number"`
	Score       int        `json:"score"`
	XCount      int        `json:"x_count"`
	PerRing     ringCounts `json:"per_ring"`
}

type shooterJSON struct {
	Name        string      `json:"name"`
	WAID        string      `json:"wa_id"`
	Club        *string     `json:"club"`
	Discipline  string      `json:"discipline"`
	WeaponClass string      `json:"weapon_class"`
	TotalScore  int         `json:"total_score"`
	XCount      int         `json:"x_count"`
	PerRing     ringCounts  `json:"per_ring"`
	Stages      []stageJSON `json:"stages"`
}

func WriteJSON(w io.Writer, shooters []repository.ExportShooterRow) error {
	out := make([]shooterJSON, len(shooters))
	for i, s := range shooters {
		stages := make([]stageJSON, len(s.Stages))
		for j, st := range s.Stages {
			stages[j] = stageJSON{
				StageNumber: st.StageNumber,
				Score:       st.Score,
				XCount:      st.XCount,
				PerRing:     ringCounts{st.RingX, st.Ring10, st.Ring9, st.Ring8, st.Ring7, st.Ring0, st.RingM},
			}
		}
		out[i] = shooterJSON{
			Name:        s.Name,
			WAID:        s.WAID,
			Club:        s.Club,
			Discipline:  s.Discipline,
			WeaponClass: s.WeaponClass,
			TotalScore:  s.TotalScore,
			XCount:      s.XCount,
			PerRing:     ringCounts{s.RingX, s.Ring10, s.Ring9, s.Ring8, s.Ring7, s.Ring0, s.RingM},
			Stages:      stages,
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
