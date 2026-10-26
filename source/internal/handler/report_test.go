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
	last, first, sex, waid, category string
	lane                             int
	counts                           map[string]any
}

func seedReport(t *testing.T, env *apitest.Env, name string, shooters []reportShooter) int {
	t.Helper()
	comp := env.ID("POST", "/api/competitions", map[string]any{"name": name, "date": "2026-05-01", "course": "150"})
	entryIDs := make([]int, len(shooters))
	for i, s := range shooters {
		body := map[string]any{"last_name": s.last, "first_name": s.first, "sex": s.sex, "country": "NOR", "club": "Oslo PPC"}
		if s.waid != "" {
			body["waid"] = s.waid
		}
		sid := env.ID("POST", "/api/shooters", body)
		entryIDs[i] = env.ID("POST", fmt.Sprintf("/api/competitions/%d/entries", comp), map[string]any{"shooter_id": sid, "category": s.category, "lane": s.lane})
	}
	for i, s := range shooters {
		if s.counts != nil {
			path := fmt.Sprintf("/api/competitions/%d/entries/%d/stages/1", comp, entryIDs[i])
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
	status, header, body := env.Raw("GET", fmt.Sprintf("/api/competitions/%d/results.xlsx", comp))
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

func TestResultsWorkbookSmoke(t *testing.T) {
	env := apitest.NewAdmin(t)
	comp := seedReport(t, env, "Spring Match", []reportShooter{
		{"BERG", "Ola", "M", "W100", "Revolver", 1, counts112},
		{"NYHUS", "Kari", "F", "", "Revolver", 2, counts112},
	})
	f := openReport(t, env, comp)
	if got := len(f.GetSheetList()); got != 5 {
		t.Fatalf("sheets %d", got)
	}
	if cell(t, f, "Revolver", "B1") != "Spring Match" || cell(t, f, "Revolver", "A3") != "W100" || cell(t, f, "Revolver", "B3") != "112" {
		t.Fatal("workbook not filled from results")
	}
}

func TestResultsCSVSmoke(t *testing.T) {
	env := apitest.NewAdmin(t)
	comp := seedReport(t, env, "Spring Match", []reportShooter{
		{"BERG", "Ola", "M", "W100", "Revolver", 1, counts112},
		{"NYHUS", "Kari", "F", "", "Pistol", 2, nil},
	})
	status, header, body := env.Raw("GET", fmt.Sprintf("/api/competitions/%d/results.csv", comp))
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
	if len(rows) != 21 {
		t.Fatalf("rows %d, want 1 header + 2 entries x 10 stages", len(rows))
	}
	if got := strings.Join(rows[1], ","); got != "Revolver,1,BERG,Ola,W100,false,1,3,4,3,1,1,0,0,112,true,,"+apitest.AdminUsername {
		t.Fatalf("first row %q", got)
	}
}

func TestResultsKeepNonASCIINames(t *testing.T) {
	env := apitest.NewAdmin(t)
	comp := seedReport(t, env, "Spring Match", []reportShooter{
		{"HANSEN", "Pål", "M", "W100", "Revolver", 1, nil},
		{"ÅSEN", "Östen", "M", "", "Revolver", 2, nil},
	})
	_, _, body := env.Raw("GET", fmt.Sprintf("/api/competitions/%d/results.csv", comp))
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
				_, header, _ := env.Raw("GET", fmt.Sprintf("/api/competitions/%d/results.%s", comp, ext))
				want := strings.ReplaceAll(tt.want, "%s", ext)
				if got := header.Get("Content-Disposition"); got != want {
					t.Fatalf("got %q, want %q", got, want)
				}
			})
		}
	}
}

func TestResultsUnknownCompetition(t *testing.T) {
	env := apitest.NewAdmin(t)
	for _, p := range []string{"results.xlsx", "results.csv"} {
		if status, _, _ := env.Raw("GET", "/api/competitions/999/"+p); status != http.StatusNotFound {
			t.Fatalf("%s status %d", p, status)
		}
	}
}
