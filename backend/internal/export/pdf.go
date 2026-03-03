package export

import (
	"fmt"
	"io"

	"github.com/go-pdf/fpdf"
	"github.com/sollie/matchmark/internal/repository"
)

type matchGroup struct {
	label  string
	stages []int
}

func groupsForDiscipline(discipline string) []matchGroup {
	switch discipline {
	case "1500":
		return []matchGroup{
			{"Match 1", []int{1, 2}},
			{"Match 2", []int{3}},
			{"Match 3", []int{4}},
			{"Match 4", []int{5, 6}},
			{"Match 5", []int{7, 8, 9, 10}},
		}
	default:
		return []matchGroup{
			{"Subtotal 1", []int{1, 2}},
			{"Subtotal 2", []int{3, 4}},
		}
	}
}

func WritePDF(w io.Writer, competitionName, competitionDate string, shooters []repository.ExportShooterRow) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(true, 15)

	cols := []string{"Stage", "Points", "X", "10", "9", "8", "7", "0", "M"}
	colW := []float64{14, 18, 10, 10, 10, 10, 10, 10, 10}

	ringVal := func(st repository.ExportStageRow, col string) string {
		switch col {
		case "X":
			return fmt.Sprintf("%d", st.RingX)
		case "10":
			return fmt.Sprintf("%d", st.Ring10)
		case "9":
			return fmt.Sprintf("%d", st.Ring9)
		case "8":
			return fmt.Sprintf("%d", st.Ring8)
		case "7":
			return fmt.Sprintf("%d", st.Ring7)
		case "0":
			return fmt.Sprintf("%d", st.Ring0)
		case "M":
			return fmt.Sprintf("%d", st.RingM)
		}
		return ""
	}

	writeRow := func(cells []string, bold bool, fillColor [3]int) {
		if bold {
			pdf.SetFont("Helvetica", "B", 8)
		} else {
			pdf.SetFont("Helvetica", "", 8)
		}
		if fillColor[0] != 255 || fillColor[1] != 255 || fillColor[2] != 255 {
			pdf.SetFillColor(fillColor[0], fillColor[1], fillColor[2])
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		fill := fillColor[0] != 255 || fillColor[1] != 255 || fillColor[2] != 255
		for i, cell := range cells {
			align := "C"
			if i == 0 {
				align = "L"
			}
			pdf.CellFormat(colW[i], 6, cell, "1", 0, align, fill, 0, "")
		}
		pdf.Ln(-1)
	}

	stageMap := func(stages []repository.ExportStageRow) map[int]repository.ExportStageRow {
		m := make(map[int]repository.ExportStageRow, len(stages))
		for _, s := range stages {
			m[s.StageNumber] = s
		}
		return m
	}

	for _, sh := range shooters {
		pdf.AddPage()

		pdf.SetFont("Helvetica", "B", 11)
		pdf.CellFormat(0, 7, competitionName, "", 1, "C", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.CellFormat(0, 5, competitionDate, "", 1, "C", false, 0, "")
		pdf.Ln(2)

		club := ""
		if sh.Club != nil {
			club = *sh.Club
		}
		pdf.SetFont("Helvetica", "", 9)
		pdf.CellFormat(0, 5, fmt.Sprintf("Name: %s   WA-ID: %s   Club: %s", sh.Name, sh.WAID, club), "", 1, "L", false, 0, "")
		pdf.CellFormat(0, 5, fmt.Sprintf("Discipline: %s   Weapon Class: %s", sh.Discipline, sh.WeaponClass), "", 1, "L", false, 0, "")
		pdf.Ln(3)

		writeRow(cols, true, [3]int{220, 220, 220})

		sm := stageMap(sh.Stages)
		groups := groupsForDiscipline(sh.Discipline)

		for _, g := range groups {
			var groupScore, groupX, gRingX, gRing10, gRing9, gRing8, gRing7, gRing0, gRingM int
			for _, sn := range g.stages {
				st, ok := sm[sn]
				if !ok {
					continue
				}
				writeRow([]string{
					fmt.Sprintf("Stage %d", st.StageNumber),
					fmt.Sprintf("%d", st.Score),
					fmt.Sprintf("%d", st.XCount),
					ringVal(st, "10"),
					ringVal(st, "9"),
					ringVal(st, "8"),
					ringVal(st, "7"),
					ringVal(st, "0"),
					ringVal(st, "M"),
				}, false, [3]int{255, 255, 255})
				groupScore += st.Score
				groupX += st.XCount
				gRingX += st.RingX
				gRing10 += st.Ring10
				gRing9 += st.Ring9
				gRing8 += st.Ring8
				gRing7 += st.Ring7
				gRing0 += st.Ring0
				gRingM += st.RingM
			}
			writeRow([]string{
				g.label,
				fmt.Sprintf("%d", groupScore),
				fmt.Sprintf("%d", groupX),
				fmt.Sprintf("%d", gRingX),
				fmt.Sprintf("%d", gRing10),
				fmt.Sprintf("%d", gRing9),
				fmt.Sprintf("%d", gRing8),
				fmt.Sprintf("%d", gRing7),
				fmt.Sprintf("%d", gRing0),
				fmt.Sprintf("%d", gRingM),
			}, true, [3]int{235, 235, 235})
		}

		writeRow([]string{
			"Grand Total",
			fmt.Sprintf("%d", sh.TotalScore),
			fmt.Sprintf("%d", sh.XCount),
			fmt.Sprintf("%d", sh.RingX),
			fmt.Sprintf("%d", sh.Ring10),
			fmt.Sprintf("%d", sh.Ring9),
			fmt.Sprintf("%d", sh.Ring8),
			fmt.Sprintf("%d", sh.Ring7),
			fmt.Sprintf("%d", sh.Ring0),
			fmt.Sprintf("%d", sh.RingM),
		}, true, [3]int{200, 200, 200})

		pdf.Ln(8)
		pdf.SetFont("Helvetica", "", 9)
		lineW := (colW[0] + colW[1] + colW[2] + colW[3] + colW[4] + colW[5] + colW[6] + colW[7] + colW[8]) / 3
		pdf.CellFormat(lineW, 5, "Scorer: ____________________", "", 0, "L", false, 0, "")
		pdf.CellFormat(lineW, 5, "Shooter: ____________________", "", 0, "C", false, 0, "")
		pdf.CellFormat(lineW, 5, "Match Director: ____________________", "", 1, "R", false, 0, "")
	}

	return pdf.Output(w)
}
