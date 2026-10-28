package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/sollie/matchmark/internal/apitest"
)

func listedIDs(env *apitest.Env, query string) []int {
	_, out := env.Do("GET", "/api/shooters"+query, nil)
	var ids []int
	for _, s := range out["shooters"].([]any) {
		ids = append(ids, int(s.(map[string]any)["id"].(float64)))
	}
	return ids
}

func contains(ids []int, id int) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func TestArchivedShooterIsHiddenFromTheDefaultListButSearchable(t *testing.T) {
	env := apitest.NewAdmin(t)
	id := env.ID("POST", "/api/shooters", shooterBody("GONE"))
	if status, out := env.Do("POST", fmt.Sprintf("/api/shooters/%d/archive", id), nil); status != http.StatusOK || out["archived"] != true {
		t.Fatalf("archive %d %v", status, out)
	}
	if contains(listedIDs(env, ""), id) {
		t.Fatal("archived shooter in default list")
	}
	if !contains(listedIDs(env, "?include_archived=true"), id) {
		t.Fatal("archived shooter missing from include_archived list")
	}
	_, got := env.Do("GET", fmt.Sprintf("/api/shooters/%d", id), nil)
	if got["archived"] != true {
		t.Fatalf("detail %v", got)
	}

	if status, out := env.Do("POST", fmt.Sprintf("/api/shooters/%d/unarchive", id), nil); status != http.StatusOK || out["archived"] != false {
		t.Fatalf("unarchive %d %v", status, out)
	}
	if !contains(listedIDs(env, ""), id) {
		t.Fatal("unarchived shooter missing from default list")
	}
	if status, _ := env.Do("POST", "/api/shooters/999/archive", nil); status != http.StatusNotFound {
		t.Fatalf("unknown shooter status %d", status)
	}
}

func TestArchivingTwiceIsANoOpWithoutAudit(t *testing.T) {
	env := apitest.NewAdmin(t)
	id := env.ID("POST", "/api/shooters", shooterBody("TWICE"))
	count := func(action string) int {
		n := 0
		for _, row := range auditEntries(t, env, "?page_size=200") {
			if row["action"] == action {
				n++
			}
		}
		return n
	}
	for i := 0; i < 2; i++ {
		if status, out := env.Do("POST", fmt.Sprintf("/api/shooters/%d/archive", id), nil); status != http.StatusOK || out["archived"] != true {
			t.Fatalf("archive %d %v", status, out)
		}
	}
	if n := count("shooter.archive"); n != 1 {
		t.Fatalf("archive audited %d times", n)
	}
	for i := 0; i < 2; i++ {
		if status, out := env.Do("POST", fmt.Sprintf("/api/shooters/%d/unarchive", id), nil); status != http.StatusOK || out["archived"] != false {
			t.Fatalf("unarchive %d %v", status, out)
		}
	}
	if n := count("shooter.unarchive"); n != 1 {
		t.Fatalf("unarchive audited %d times", n)
	}
	if status, _ := env.Do("POST", "/api/shooters/9999/archive", nil); status != http.StatusNotFound {
		t.Fatalf("unknown shooter %d", status)
	}
}

func TestArchivedShooterKeepsEntriesAndCannotBeEnteredAgain(t *testing.T) {
	f := newFixture(t, "150")
	f.addEntry(t, "Kept", 1)
	kept := listedIDs(f.env, "")[0]
	other := f.env.ID("POST", "/api/shooters", shooterBody("OTHER"))
	f.env.Do("POST", fmt.Sprintf("/api/shooters/%d/archive", kept), nil)
	f.env.Do("POST", fmt.Sprintf("/api/shooters/%d/archive", other), nil)

	_, entries := f.env.Do("GET", fmt.Sprintf("/api/matches/%d/entries", f.comp), nil)
	if len(entries["entries"].([]any)) != 1 {
		t.Fatalf("existing entry changed %v", entries)
	}
	path := fmt.Sprintf("/api/matches/%d/entries", f.comp)
	status, out := f.env.Do("POST", path, map[string]any{"shooter_id": other, "category": "Revolver", "lane": 2})
	if status != http.StatusConflict || out["error"] != "shooter_archived" {
		t.Fatalf("got %d %v", status, out)
	}
	if status, _ := f.env.Do("DELETE", fmt.Sprintf("/api/shooters/%d", kept), nil); status != http.StatusConflict {
		t.Fatalf("delete status %d", status)
	}

	f.env.Do("POST", fmt.Sprintf("/api/shooters/%d/unarchive", other), nil)
	if status, out := f.env.Do("POST", path, map[string]any{"shooter_id": other, "category": "Revolver", "lane": 2}); status != http.StatusCreated {
		t.Fatalf("after unarchive %d %v", status, out)
	}
}

func TestImportMatchingArchivedShooterAsksForUnarchive(t *testing.T) {
	env := apitest.NewAdmin(t)
	_, first := env.Do("POST", "/api/shooters/import", importBody(nil))
	id := int(first["id"].(float64))
	env.Do("POST", fmt.Sprintf("/api/shooters/%d/archive", id), nil)

	status, out := env.Do("POST", "/api/shooters/import", importBody(map[string]any{"club": "New Club"}))
	shooter, _ := out["shooter"].(map[string]any)
	if status != http.StatusConflict || out["error"] != "shooter_archived" || int(shooter["id"].(float64)) != id {
		t.Fatalf("got %d %v", status, out)
	}
	_, stored := env.Do("GET", fmt.Sprintf("/api/shooters/%d", id), nil)
	if stored["club"] != "CCSC" || stored["archived"] != true {
		t.Fatalf("import changed archived shooter %v", stored)
	}

	env.Do("POST", fmt.Sprintf("/api/shooters/%d/unarchive", id), nil)
	status, out = env.Do("POST", "/api/shooters/import", importBody(map[string]any{"club": "New Club"}))
	if status != http.StatusOK || out["action"] != "updated" {
		t.Fatalf("after unarchive %d %v", status, out)
	}
}

func TestImportNameCandidateThatIsArchivedAsksForUnarchive(t *testing.T) {
	env := apitest.NewAdmin(t)
	manual := env.ID("POST", "/api/shooters", map[string]any{"waid": "", "first_name": "Mike", "last_name": "ANDERSON", "country": "Canada", "club": "Old"})
	env.Do("POST", fmt.Sprintf("/api/shooters/%d/archive", manual), nil)

	status, out := env.Do("POST", "/api/shooters/import", importBody(map[string]any{"update_shooter_id": manual}))
	if status != http.StatusConflict || out["error"] != "shooter_archived" {
		t.Fatalf("got %d %v", status, out)
	}
}
