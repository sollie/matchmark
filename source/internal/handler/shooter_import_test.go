package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/sollie/matchmark/internal/apitest"
)

func importBody(extra map[string]any) map[string]any {
	body := map[string]any{"waid": "3289", "first_name": "Mike", "last_name": "ANDERSON", "country": "Canada", "club": "CCSC", "sex": "M"}
	for k, v := range extra {
		body[k] = v
	}
	return body
}

func TestImportShooterCreates(t *testing.T) {
	env := apitest.NewAdmin(t)
	status, out := env.Do("POST", "/api/shooters/import", importBody(nil))
	if status != http.StatusCreated || out["action"] != "created" {
		t.Fatalf("got %d %v", status, out)
	}
	_, got := env.Do("GET", "/api/shooters/"+itoa(out["id"]), nil)
	if got["waid"] != "3289" || got["sex"] != "M" || got["club"] != "CCSC" {
		t.Fatalf("stored %v", got)
	}
}

func TestImportShooterWAIDMatchUpdatesSilently(t *testing.T) {
	env := apitest.NewAdmin(t)
	_, first := env.Do("POST", "/api/shooters/import", importBody(nil))
	status, out := env.Do("POST", "/api/shooters/import", importBody(map[string]any{"club": "New Club", "last_name": "ANDERSON-SMITH"}))
	if status != http.StatusOK || out["action"] != "updated" || out["id"] != first["id"] {
		t.Fatalf("got %d %v", status, out)
	}
	_, got := env.Do("GET", "/api/shooters/"+itoa(first["id"]), nil)
	if got["club"] != "New Club" || got["last_name"] != "ANDERSON-SMITH" {
		t.Fatalf("stored %v", got)
	}
}

func TestImportShooterNameConflict(t *testing.T) {
	env := apitest.NewAdmin(t)
	manual := env.ID("POST", "/api/shooters", map[string]any{"waid": "", "first_name": "Mike", "last_name": "ANDERSON", "country": "Canada", "club": "Old", "sex": "M"})
	other := env.ID("POST", "/api/shooters", map[string]any{"waid": "999", "first_name": "Mike", "last_name": "ANDERSON", "country": "Sweden", "club": "X", "sex": "M"})

	status, out := env.Do("POST", "/api/shooters/import", importBody(nil))
	candidates, _ := out["candidates"].([]any)
	if status != http.StatusConflict || len(candidates) != 2 {
		t.Fatalf("got %d %v", status, out)
	}
	first := candidates[0].(map[string]any)
	if int(first["id"].(float64)) != manual || first["club"] != "Old" || first["country"] != "Canada" {
		t.Fatalf("candidate %v", first)
	}

	if status, _ = env.Do("POST", "/api/shooters/import", importBody(map[string]any{"update_shooter_id": 424242})); status != http.StatusUnprocessableEntity {
		t.Fatalf("foreign candidate status %d", status)
	}
	if status, _ = env.Do("POST", "/api/shooters/import", importBody(map[string]any{"update_shooter_id": manual, "create_new": true})); status != http.StatusUnprocessableEntity {
		t.Fatalf("both flags status %d", status)
	}

	status, out = env.Do("POST", "/api/shooters/import", importBody(map[string]any{"update_shooter_id": manual}))
	if status != http.StatusOK || int(out["id"].(float64)) != manual {
		t.Fatalf("update got %d %v", status, out)
	}
	_, got := env.Do("GET", "/api/shooters/"+itoa(manual), nil)
	if got["waid"] != "3289" || got["club"] != "CCSC" {
		t.Fatalf("stored %v", got)
	}
	_, untouched := env.Do("GET", "/api/shooters/"+itoa(other), nil)
	if untouched["waid"] != "999" {
		t.Fatalf("other changed %v", untouched)
	}
}

func TestImportShooterNameConflictNonASCII(t *testing.T) {
	env := apitest.NewAdmin(t)
	env.ID("POST", "/api/shooters", map[string]any{"waid": "", "first_name": "Åse", "last_name": "ØVREBØ", "country": "Norway", "club": "X", "sex": "F"})
	status, _ := env.Do("POST", "/api/shooters/import", importBody(map[string]any{"first_name": "åse", "last_name": "Øvrebø"}))
	if status != http.StatusConflict {
		t.Fatalf("status %d", status)
	}
}

func TestImportShooterCreateNew(t *testing.T) {
	env := apitest.NewAdmin(t)
	env.ID("POST", "/api/shooters", map[string]any{"waid": "999", "first_name": "Mike", "last_name": "ANDERSON", "country": "Sweden", "club": "X", "sex": "M"})
	status, out := env.Do("POST", "/api/shooters/import", importBody(map[string]any{"create_new": true}))
	if status != http.StatusCreated || out["action"] != "created" {
		t.Fatalf("got %d %v", status, out)
	}
}

func TestImportShooterValidation(t *testing.T) {
	env := apitest.NewAdmin(t)
	for name, extra := range map[string]map[string]any{
		"no sex":  {"sex": ""},
		"no waid": {"waid": ""},
		"no name": {"first_name": " "},
	} {
		if status, _ := env.Do("POST", "/api/shooters/import", importBody(extra)); status != http.StatusUnprocessableEntity {
			t.Fatalf("%s: status %d", name, status)
		}
	}
}

func TestImportShooterRequiresAdmin(t *testing.T) {
	env := apitest.NewWithBaseURL(t, "http://localhost")
	if status, _ := env.Do("POST", "/api/shooters/import", importBody(nil)); status != http.StatusUnauthorized {
		t.Fatalf("status %d", status)
	}
}

func itoa(v any) string { return fmt.Sprint(v) }

func TestImportShooterNormalizesNames(t *testing.T) {
	env := apitest.NewAdmin(t)
	status, out := env.Do("POST", "/api/shooters/import", importBody(map[string]any{"first_name": "  anna-lisa   o'neil ", "last_name": "øvrebø  hansen"}))
	if status != http.StatusCreated {
		t.Fatalf("got %d %v", status, out)
	}
	_, got := env.Do("GET", "/api/shooters/"+itoa(out["id"]), nil)
	if got["first_name"] != "Anna-Lisa O'Neil" || got["last_name"] != "ØVREBØ HANSEN" {
		t.Fatalf("stored %v", got)
	}
}

func TestShooterWriteRejectsNonNormalizedNames(t *testing.T) {
	env := apitest.NewAdmin(t)
	for _, c := range []struct{ first, last string }{
		{"Mike", "Anderson"},
		{"mike", "ANDERSON"},
		{"Mike", " ANDERSON"},
		{"Anna-lisa", "ANDERSON"},
		{"Mike  Jr", "ANDERSON"},
	} {
		body := map[string]any{"first_name": c.first, "last_name": c.last, "country": "NOR", "club": "Club", "sex": "M"}
		if status, _ := env.Do("POST", "/api/shooters", body); status != http.StatusUnprocessableEntity {
			t.Errorf("create %q %q: got %d", c.first, c.last, status)
		}
	}
	id := env.ID("POST", "/api/shooters", map[string]any{"first_name": "Mike", "last_name": "ANDERSON", "country": "NOR", "club": "Club", "sex": "M"})
	body := map[string]any{"first_name": "Mike", "last_name": "Anderson", "country": "NOR", "club": "Club", "sex": "M"}
	if status, _ := env.Do("PUT", "/api/shooters/"+itoa(id), body); status != http.StatusUnprocessableEntity {
		t.Errorf("update: got %d", status)
	}
}
