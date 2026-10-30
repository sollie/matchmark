package handler_test

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/sollie/matchmark/internal/apitest"
)

func seedCompetitionMatch(t *testing.T, env *apitest.Env, comp int, name, date, course string, finish bool, shooterIDs map[string]int, names ...string) {
	t.Helper()
	id := env.ID("POST", "/api/matches", map[string]any{"name": name, "date": date, "course": course, "competition_id": comp})
	for i, n := range names {
		entry := env.ID("POST", fmt.Sprintf("/api/matches/%d/entries", id), map[string]any{"shooter_id": shooterIDs[n], "category": "Revolver", "lane": i + 1})
		if course != "150" {
			continue
		}
		path := fmt.Sprintf("/api/matches/%d/entries/%d/segments/1", id, entry)
		if status, out := env.Do("PUT", path, counts112); status != http.StatusOK {
			t.Fatalf("counts: %d %v", status, out)
		}
		if status, out := env.Do("POST", path+"/lock", nil); status != http.StatusOK {
			t.Fatalf("lock: %d %v", status, out)
		}
	}
	steps := []string{}
	if finish {
		steps = []string{"active", "finished"}
	}
	for _, s := range steps {
		if status, out := env.Do("POST", fmt.Sprintf("/api/matches/%d/status", id), map[string]any{"status": s}); status != http.StatusOK {
			t.Fatalf("status %s: %d %v", s, status, out)
		}
	}
}

func TestCompetitionWorkbook1500(t *testing.T) {
	env := apitest.NewAdmin(t)
	comp := env.ID("POST", "/api/competitions", map[string]any{"name": "Nationals", "start_date": "2026-05-01", "end_date": "2026-05-03"})
	shooters := map[string]int{}
	for i := 1; i <= 10; i++ {
		n := fmt.Sprintf("K%02d", i)
		shooters[n] = env.ID("POST", "/api/shooters", map[string]any{"last_name": n, "first_name": "A", "country": "NOR", "club": "Oslo PPC", "waid": fmt.Sprintf("W%03d", i)})
	}
	url := fmt.Sprintf("/api/competitions/%d/results.xlsx", comp)

	if status, _, _ := env.Raw("GET", url); status != http.StatusUnprocessableEntity {
		t.Fatalf("empty competition: status %d", status)
	}
	if status, _, _ := env.Raw("GET", "/api/competitions/99999/results.xlsx"); status != http.StatusNotFound {
		t.Fatalf("unknown competition: status %d", status)
	}

	seedCompetitionMatch(t, env, comp, "Day two", "2026-05-02", "150", true, shooters, "K01", "K02", "K03", "K04", "K05", "K06")
	seedCompetitionMatch(t, env, comp, "Day one", "2026-05-01", "150", true, shooters, "K01", "K07", "K08", "K09", "K10")
	seedCompetitionMatch(t, env, comp, "Short", "2026-05-01", "40", true, shooters, "K03")
	seedCompetitionMatch(t, env, comp, "Open", "2026-05-03", "150", false, shooters, "K04")

	status, header, body := env.Raw("GET", url)
	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	if cd := header.Get("Content-Disposition"); cd != `attachment; filename="Nationals-2026-05-01-1500.xlsx"` {
		t.Fatalf("disposition %q", cd)
	}
	f, err := excelize.OpenReader(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if cell(t, f, "Revolver", "A1") != "2026-05-01" || cell(t, f, "Revolver", "B1") != "Nationals" {
		t.Fatalf("header %q %q", cell(t, f, "Revolver", "A1"), cell(t, f, "Revolver", "B1"))
	}
	want := []string{"W001", "W001", "W002", "W003", "W004", "W005", "W006", "W007", "W008", "W009", "W010"}
	for i, w := range want {
		if got := cell(t, f, "Revolver", fmt.Sprintf("A%d", 3+i)); got != w {
			t.Fatalf("known row %d = %q want %q", 3+i, got, w)
		}
	}
	if cell(t, f, "Revolver", "A3") != "W001" || cell(t, f, "Revolver", "B3") != "224" || cell(t, f, "Revolver", "B4") != "224" {
		t.Fatal("repeat shooter rows not kept per match")
	}
	if got := cell(t, f, "Revolver", "A15"); got != "New shooters:" {
		t.Fatalf("new shooter block at A15 = %q", got)
	}

	if lines := auditEntries(t, env, "?action=report.download"); len(lines) != 1 {
		t.Fatalf("audit entries %v", lines)
	}
}
