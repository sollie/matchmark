package handler_test

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/sollie/matchmark/internal/apitest"
)

type reportShooter struct {
	last, first, waid, category string
	lane                        int
	counts                      map[string]any
}

func seedReport(t *testing.T, env *apitest.Env, name string, shooters []reportShooter) int {
	t.Helper()
	comp := env.ID("POST", "/api/matches", map[string]any{"name": name, "date": "2026-05-01", "course": "150"})
	entryIDs := make([]int, len(shooters))
	for i, s := range shooters {
		body := map[string]any{"last_name": s.last, "first_name": s.first, "country": "NOR", "club": "Oslo PPC"}
		if s.waid != "" {
			body["waid"] = s.waid
		}
		sid := env.ID("POST", "/api/shooters", body)
		entryIDs[i] = env.ID("POST", fmt.Sprintf("/api/matches/%d/entries", comp), map[string]any{"shooter_id": sid, "category": s.category, "lane": s.lane})
	}
	for i, s := range shooters {
		if s.counts != nil {
			path := fmt.Sprintf("/api/matches/%d/entries/%d/stages/1", comp, entryIDs[i])
			if status, out := env.Do("PUT", path, s.counts); status != http.StatusOK {
				t.Fatalf("set counts: %d %v", status, out)
			}
			if status, out := env.Do("POST", path+"/lock", nil); status != http.StatusOK {
				t.Fatalf("lock: %d %v", status, out)
			}
		}
	}
	return comp
}

var counts112 = map[string]any{"x": 3, "10": 4, "9": 3, "8": 1, "7": 1, "0": 0, "miss": 0}

func openReport(t *testing.T, env *apitest.Env, comp int) *excelize.File {
	t.Helper()
	status, header, body := env.Raw("GET", fmt.Sprintf("/api/matches/%d/results.xlsx", comp))
	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	if ct := header.Get("Content-Type"); ct != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("content type %q", ct)
	}
	f, err := excelize.OpenReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("open workbook: %v", err)
	}
	return f
}

func cell(t *testing.T, f *excelize.File, sheet, ref string) string {
	t.Helper()
	v, err := f.GetCellValue(sheet, ref)
	if err != nil {
		t.Fatalf("cell %s!%s: %v", sheet, ref, err)
	}
	return v
}

func TestResultsWorkbook(t *testing.T) {
	env := apitest.NewAdmin(t)
	comp := seedReport(t, env, "Spring Match", []reportShooter{
		{"BERG", "Ola", "W100", "Revolver", 1, counts112},
		{"DAHL", "Eva", "W200", "Revolver", 2, nil},
		{"NYHUS", "Kari", "", "Revolver", 3, counts112},
		{"MOE", "Per", "W300", "Pistol", 4, counts112},
		{"LIE", "Tor", "W400", "Carbine", 5, counts112},
	})
	f := openReport(t, env, comp)

	t.Run("one sheet per category", func(t *testing.T) {
		if got := len(f.GetSheetList()); got != 5 {
			t.Fatalf("sheets %d", got)
		}
	})
	t.Run("header", func(t *testing.T) {
		if cell(t, f, "Revolver", "A1") != "2026-05-01" || cell(t, f, "Revolver", "B1") != "Spring Match" {
			t.Fatalf("header %q %q", cell(t, f, "Revolver", "A1"), cell(t, f, "Revolver", "B1"))
		}
	})
	t.Run("known shooters with total", func(t *testing.T) {
		want := [][2]string{{"W100", "112"}, {"W200", "0"}}
		for i, w := range want {
			row := 3 + i
			if got := [2]string{cell(t, f, "Revolver", fmt.Sprintf("A%d", row)), cell(t, f, "Revolver", fmt.Sprintf("B%d", row))}; got != w {
				t.Fatalf("row %d: %v want %v", row, got, w)
			}
		}
	})
	t.Run("new shooter block", func(t *testing.T) {
		want := []string{"NYHUS", "Kari", "", "NOR", "Oslo PPC", "nn", "112"}
		for i, w := range want {
			ref := fmt.Sprintf("%c14", 'A'+i)
			if got := cell(t, f, "Revolver", ref); got != w {
				t.Fatalf("%s = %q want %q", ref, got, w)
			}
		}
		if cell(t, f, "Revolver", "A12") != "New shooters:" || cell(t, f, "Revolver", "A13") != "Last name" {
			t.Fatal("new shooters header moved")
		}
	})
	t.Run("other categories", func(t *testing.T) {
		if cell(t, f, "Pistol", "A3") != "W300" || cell(t, f, "Pistol", "B3") != "112" {
			t.Fatal("pistol row")
		}
		if cell(t, f, "Pistol", "F14") != "" {
			t.Fatal("placeholder not cleared")
		}
		if cell(t, f, "Carbine", "A3") != "W400" || cell(t, f, "Carbine", "B3") != "112" {
			t.Fatal("carbine row")
		}
	})
}

func TestResultsWorkbookInsertsRows(t *testing.T) {
	env := apitest.NewAdmin(t)
	var shooters []reportShooter
	for i := 1; i <= 10; i++ {
		shooters = append(shooters, reportShooter{fmt.Sprintf("K%02d", i), "A", fmt.Sprintf("W%03d", i), "Revolver", i, counts112})
	}
	for i := 1; i <= 4; i++ {
		shooters = append(shooters, reportShooter{fmt.Sprintf("N%02d", i), "B", "", "Revolver", 10 + i, counts112})
	}
	comp := seedReport(t, env, "Big", shooters)
	f := openReport(t, env, comp)

	if cell(t, f, "Revolver", "A12") != "W010" || cell(t, f, "Revolver", "B12") != "112" {
		t.Fatalf("last known row: %q", cell(t, f, "Revolver", "A12"))
	}
	if cell(t, f, "Revolver", "A14") != "New shooters:" || cell(t, f, "Revolver", "A15") != "Last name" {
		t.Fatalf("block not shifted: %q %q", cell(t, f, "Revolver", "A14"), cell(t, f, "Revolver", "A15"))
	}
	for i := 1; i <= 4; i++ {
		if got := cell(t, f, "Revolver", fmt.Sprintf("A%d", 15+i)); got != fmt.Sprintf("N%02d", i) {
			t.Fatalf("new shooter %d: %q", i, got)
		}
	}
	if got := cell(t, f, "Revolver", "G19"); got != "112" {
		t.Fatalf("last new shooter result %q", got)
	}
}

func TestResultsCSV(t *testing.T) {
	env := apitest.NewAdmin(t)
	comp := seedReport(t, env, "Spring Match", []reportShooter{
		{"BERG", "Ola", "W100", "Revolver", 1, counts112},
		{"NYHUS", "Kari", "", "Pistol", 2, nil},
	})
	status, header, body := env.Raw("GET", fmt.Sprintf("/api/matches/%d/results.csv", comp))
	if status != http.StatusOK || !strings.HasPrefix(header.Get("Content-Type"), "text/csv") {
		t.Fatalf("status %d type %q", status, header.Get("Content-Type"))
	}
	if !bytes.HasPrefix(body, []byte("\xef\xbb\xbf")) {
		t.Fatalf("missing byte-order mark")
	}
	rows, err := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(body, []byte("\xef\xbb\xbf")))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	wantHeader := "category,lane,last_name,first_name,waid,withdrawn,stage,part,X,10,9,8,7,0,miss,total,accepted,scorer_lane,scorer_name"
	if strings.Join(rows[0], ",") != wantHeader {
		t.Fatalf("header %v", rows[0])
	}
	if len(rows) != 21 {
		t.Fatalf("rows %d, want 1 header + 2 entries x 10 stages", len(rows))
	}
	if got := strings.Join(rows[1], ","); got != "Revolver,1,BERG,Ola,W100,false,1,Match 1,3,4,3,1,1,0,0,112,true,,"+apitest.AdminUsername {
		t.Fatalf("first row %q", got)
	}
	if got := strings.Join(rows[11], ","); got != "Pistol,2,NYHUS,Kari,,false,1,Match 1,0,0,0,0,0,0,0,0,false,," {
		t.Fatalf("pistol row %q", got)
	}
}

func TestResultsCSVNeutralisesFormulas(t *testing.T) {
	env := apitest.NewAdmin(t)
	comp := seedReport(t, env, "Spring Match", []reportShooter{
		{"=HYPERLINK(\"X\")", "+1", "", "Revolver", 1, nil},
		{"-2", "@sum", "", "Revolver", 2, nil},
	})
	_, _, body := env.Raw("GET", fmt.Sprintf("/api/matches/%d/results.csv", comp))
	rows, err := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(body, []byte("\xef\xbb\xbf")))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	want := [][2]string{{"'=HYPERLINK(\"X\")", "'+1"}, {"'-2", "'@sum"}}
	got := map[string][2]string{}
	for _, row := range rows[1:] {
		got[row[1]] = [2]string{row[2], row[3]}
	}
	for i, w := range want {
		if got[fmt.Sprint(i+1)] != w {
			t.Fatalf("lane %d names %q, want %q", i+1, got[fmt.Sprint(i+1)], w)
		}
	}
}

func TestResultsKeepNonASCIINames(t *testing.T) {
	env := apitest.NewAdmin(t)
	comp := seedReport(t, env, "Spring Match", []reportShooter{
		{"HANSEN", "Pål", "W100", "Revolver", 1, nil},
		{"ÅSEN", "Östen", "", "Revolver", 2, nil},
	})
	_, _, body := env.Raw("GET", fmt.Sprintf("/api/matches/%d/results.csv", comp))
	if !bytes.HasPrefix(body, []byte("\xef\xbb\xbf")) || !bytes.Contains(body, []byte("HANSEN,Pål,W100")) || !bytes.Contains(body, []byte("ÅSEN,Östen,")) {
		t.Fatalf("names garbled in csv")
	}
	f := openReport(t, env, comp)
	found := map[string]bool{}
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range rows {
			for _, c := range row {
				found[c] = true
			}
		}
	}
	if !found["ÅSEN"] || !found["Östen"] {
		t.Fatalf("workbook lacks new shooter names: %v", found)
	}
}

func TestResultsDownloadFilename(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Test2", `attachment; filename="Test2-2026-05-01.%s"`},
		{"Spring Match: A/B  \"final\"", `attachment; filename="Spring-Match-A-B-final-2026-05-01.%s"`},
		{"Åre cup", `attachment; filename="_re-cup-2026-05-01.%s"; filename*=UTF-8''%C3%85re-cup-2026-05-01.%s`},
		{"///", `attachment; filename="results-2026-05-01.%s"`},
		{strings.Repeat("a", 100), `attachment; filename="` + strings.Repeat("a", 60) + `-2026-05-01.%s"`},
	}
	for _, tt := range tests {
		for _, ext := range []string{"xlsx", "csv"} {
			t.Run(tt.name+"/"+ext, func(t *testing.T) {
				env := apitest.NewAdmin(t)
				comp := seedReport(t, env, tt.name, nil)
				_, header, _ := env.Raw("GET", fmt.Sprintf("/api/matches/%d/results.%s", comp, ext))
				want := strings.ReplaceAll(tt.want, "%s", ext)
				if got := header.Get("Content-Disposition"); got != want {
					t.Fatalf("got %q, want %q", got, want)
				}
			})
		}
	}
}

func TestResultsUnknownMatch(t *testing.T) {
	env := apitest.NewAdmin(t)
	for _, p := range []string{"results.xlsx", "results.csv"} {
		if status, _, _ := env.Raw("GET", "/api/matches/999/"+p); status != http.StatusNotFound {
			t.Fatalf("%s status %d", p, status)
		}
	}
}
