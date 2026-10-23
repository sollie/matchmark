package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/sollie/matchmark/internal/apitest"
)

func shooterBody(last string) map[string]any {
	return map[string]any{"last_name": last, "first_name": "A", "sex": "M", "country": "NOR", "club": "Club"}
}

func TestShooterRegistry(t *testing.T) {
	env := apitest.NewAdmin(t)
	id := env.ID("POST", "/api/shooters", shooterBody("Hansen"))

	t.Run("get", func(t *testing.T) {
		status, out := env.Do("GET", fmt.Sprintf("/api/shooters/%d", id), nil)
		if status != http.StatusOK || out["last_name"] != "Hansen" {
			t.Fatalf("got %d %v", status, out)
		}
	})
	t.Run("update", func(t *testing.T) {
		body := shooterBody("Olsen")
		body["waid"] = "W123"
		status, out := env.Do("PUT", fmt.Sprintf("/api/shooters/%d", id), body)
		if status != http.StatusOK || out["last_name"] != "Olsen" || out["waid"] != "W123" {
			t.Fatalf("got %d %v", status, out)
		}
	})
	t.Run("list", func(t *testing.T) {
		env.ID("POST", "/api/shooters", shooterBody("Berg"))
		status, out := env.Do("GET", "/api/shooters", nil)
		if status != http.StatusOK || len(out["shooters"].([]any)) != 2 {
			t.Fatalf("got %d %v", status, out)
		}
	})
	t.Run("duplicate waid conflicts", func(t *testing.T) {
		body := shooterBody("Dup")
		body["waid"] = "W123"
		if status, _ := env.Do("POST", "/api/shooters", body); status != http.StatusConflict {
			t.Fatalf("status %d", status)
		}
	})
	t.Run("unknown shooter", func(t *testing.T) {
		if status, _ := env.Do("GET", "/api/shooters/999", nil); status != http.StatusNotFound {
			t.Fatalf("status %d", status)
		}
	})
	t.Run("delete", func(t *testing.T) {
		if status, _ := env.Do("DELETE", fmt.Sprintf("/api/shooters/%d", id), nil); status != http.StatusOK {
			t.Fatalf("status %d", status)
		}
		if status, _ := env.Do("GET", fmt.Sprintf("/api/shooters/%d", id), nil); status != http.StatusNotFound {
			t.Fatalf("status %d", status)
		}
	})
	t.Run("shooter with entry cannot be deleted", func(t *testing.T) {
		f := newFixture(t, "150")
		f.addEntry(t, "Entered", 1)
		_, out := f.env.Do("GET", "/api/shooters", nil)
		sid := int(out["shooters"].([]any)[0].(map[string]any)["id"].(float64))
		if status, _ := f.env.Do("DELETE", fmt.Sprintf("/api/shooters/%d", sid), nil); status != http.StatusConflict {
			t.Fatalf("status %d", status)
		}
	})
}

func TestCompetitionLifecycle(t *testing.T) {
	env := apitest.NewAdmin(t)
	id := env.ID("POST", "/api/competitions", map[string]any{"name": "Spring", "date": "2026-05-01", "course": "150"})
	path := fmt.Sprintf("/api/competitions/%d", id)

	t.Run("new competition is draft", func(t *testing.T) {
		status, out := env.Do("GET", path, nil)
		if status != http.StatusOK || out["status"] != "draft" || out["course"] != "150" {
			t.Fatalf("got %d %v", status, out)
		}
	})
	t.Run("update while draft", func(t *testing.T) {
		status, out := env.Do("PUT", path, map[string]any{"name": "Autumn", "date": "2026-10-01", "course": "60"})
		if status != http.StatusOK || out["name"] != "Autumn" || out["course"] != "60" {
			t.Fatalf("got %d %v", status, out)
		}
	})
	t.Run("list", func(t *testing.T) {
		status, out := env.Do("GET", "/api/competitions", nil)
		if status != http.StatusOK || len(out["competitions"].([]any)) != 1 {
			t.Fatalf("got %d %v", status, out)
		}
	})
	t.Run("transitions", func(t *testing.T) {
		tests := []struct {
			to   string
			want int
		}{
			{"finished", http.StatusConflict},
			{"bogus", http.StatusUnprocessableEntity},
			{"active", http.StatusOK},
			{"draft", http.StatusConflict},
			{"finished", http.StatusOK},
			{"active", http.StatusConflict},
		}
		for _, tt := range tests {
			status, out := env.Do("POST", path+"/status", map[string]any{"status": tt.to})
			if status != tt.want {
				t.Fatalf("to %s: got %d %v, want %d", tt.to, status, out, tt.want)
			}
		}
		_, out := env.Do("GET", path, nil)
		if out["status"] != "finished" {
			t.Fatalf("got %v", out)
		}
	})
	t.Run("update after draft is rejected", func(t *testing.T) {
		status, _ := env.Do("PUT", path, map[string]any{"name": "X", "date": "2026-10-01", "course": "60"})
		if status != http.StatusConflict {
			t.Fatalf("status %d", status)
		}
	})
	t.Run("delete", func(t *testing.T) {
		if status, _ := env.Do("DELETE", path, nil); status != http.StatusOK {
			t.Fatalf("status %d", status)
		}
		if status, _ := env.Do("GET", path, nil); status != http.StatusNotFound {
			t.Fatalf("status %d", status)
		}
	})
}

func TestCourseChangeRejectedWithEntries(t *testing.T) {
	f := newFixture(t, "150")
	f.addEntry(t, "One", 1)
	status, _ := f.env.Do("PUT", fmt.Sprintf("/api/competitions/%d", f.comp), map[string]any{"name": "Club Match", "date": "2026-05-01", "course": "60"})
	if status != http.StatusConflict {
		t.Fatalf("status %d", status)
	}
}

func TestEntriesListWithStages(t *testing.T) {
	tests := []struct {
		course string
		shots  []int
	}{
		{"150", []int{12, 12, 18, 24, 12, 12, 12, 18, 24, 6}},
		{"60", []int{12, 18, 24, 6}},
		{"48", []int{6, 12, 12, 18}},
		{"40", []int{5, 10, 10, 15}},
	}
	for _, tt := range tests {
		t.Run(tt.course, func(t *testing.T) {
			f := newFixture(t, tt.course)
			f.addEntry(t, "One", 2)
			status, out := f.env.Do("GET", fmt.Sprintf("/api/competitions/%d/entries", f.comp), nil)
			if status != http.StatusOK {
				t.Fatalf("status %d", status)
			}
			entries := out["entries"].([]any)
			if len(entries) != 1 {
				t.Fatalf("got %v", out)
			}
			e := entries[0].(map[string]any)
			if e["lane"].(float64) != 2 || e["category"] != "Revolver" || e["withdrawn"] != false {
				t.Fatalf("got %v", e)
			}
			stages := e["stages"].([]any)
			if len(stages) != len(tt.shots) {
				t.Fatalf("got %d stages", len(stages))
			}
			for i, s := range stages {
				m := s.(map[string]any)
				if int(m["stage_number"].(float64)) != i+1 || int(m["expected_shots"].(float64)) != tt.shots[i] {
					t.Fatalf("stage %d: %v", i+1, m)
				}
			}
		})
	}
}

func TestEntriesListStageScores(t *testing.T) {
	f := newFixture(t, "150")
	entry := f.addEntry(t, "One", 1)
	base := fmt.Sprintf("/api/competitions/%d/entries/%d/stages", f.comp, entry)
	f.env.Do("POST", fmt.Sprintf("/api/competitions/%d/status", f.comp), map[string]any{"status": "active"})
	if status, out := f.env.Do("PUT", base+"/1", map[string]any{"x": 10, "9": 1, "8": 1}); status != http.StatusOK {
		t.Fatalf("set counts: %d %v", status, out)
	}
	if status, out := f.env.Do("PUT", base+"/2", map[string]any{"x": 12}); status != http.StatusOK {
		t.Fatalf("set counts: %d %v", status, out)
	}
	if status, out := f.env.Do("POST", base+"/2/lock", nil); status != http.StatusNoContent && status != http.StatusOK {
		t.Fatalf("lock: %d %v", status, out)
	}

	_, out := f.env.Do("GET", fmt.Sprintf("/api/competitions/%d/entries", f.comp), nil)
	stages := out["entries"].([]any)[0].(map[string]any)["stages"].([]any)
	want := []struct {
		score, shots int
		status       string
	}{{117, 12, "open"}, {120, 12, "accepted"}, {0, 0, "open"}}
	for i, w := range want {
		m := stages[i].(map[string]any)
		if int(m["score"].(float64)) != w.score || int(m["shots_entered"].(float64)) != w.shots || m["status"] != w.status {
			t.Fatalf("stage %d: %v", i+1, m)
		}
	}
}

func TestEntryUpdate(t *testing.T) {
	f := newFixture(t, "150")
	e1 := f.addEntry(t, "One", 1)
	f.addEntry(t, "Two", 2)
	path := fmt.Sprintf("/api/competitions/%d/entries/%d", f.comp, e1)

	t.Run("change category and lane", func(t *testing.T) {
		status, out := f.env.Do("PUT", path, map[string]any{"category": "Rifle", "lane": 5})
		if status != http.StatusOK || out["category"] != "Rifle" || out["lane"].(float64) != 5 {
			t.Fatalf("got %d %v", status, out)
		}
	})
	t.Run("occupied lane conflicts", func(t *testing.T) {
		if status, _ := f.env.Do("PUT", path, map[string]any{"category": "Rifle", "lane": 2}); status != http.StatusConflict {
			t.Fatalf("status %d", status)
		}
	})
	t.Run("invalid category", func(t *testing.T) {
		if status, _ := f.env.Do("PUT", path, map[string]any{"category": "Shotgun", "lane": 5}); status != http.StatusUnprocessableEntity {
			t.Fatalf("status %d", status)
		}
	})
	t.Run("unknown entry", func(t *testing.T) {
		p := fmt.Sprintf("/api/competitions/%d/entries/999", f.comp)
		if status, _ := f.env.Do("PUT", p, map[string]any{"category": "Rifle", "lane": 7}); status != http.StatusNotFound {
			t.Fatalf("status %d", status)
		}
	})
	t.Run("withdrawn flag is listed", func(t *testing.T) {
		f.env.Do("POST", path+"/withdraw", nil)
		_, out := f.env.Do("GET", fmt.Sprintf("/api/competitions/%d/entries", f.comp), nil)
		withdrawn := 0
		for _, e := range out["entries"].([]any) {
			if e.(map[string]any)["withdrawn"] == true {
				withdrawn++
			}
		}
		if withdrawn != 1 {
			t.Fatalf("got %v", out)
		}
	})
}

func TestEntryDeleteOnlyInDraft(t *testing.T) {
	f := newFixture(t, "150")
	e1 := f.addEntry(t, "One", 1)
	e2 := f.addEntry(t, "Two", 2)
	base := fmt.Sprintf("/api/competitions/%d", f.comp)

	if status, _ := f.env.Do("DELETE", fmt.Sprintf("%s/entries/%d", base, e1), nil); status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	f.env.Do("POST", base+"/status", map[string]any{"status": "active"})
	if status, _ := f.env.Do("DELETE", fmt.Sprintf("%s/entries/%d", base, e2), nil); status != http.StatusConflict {
		t.Fatalf("status %d", status)
	}
}

func lockStage(t *testing.T, f *fixture, entry, stage, shots int) {
	t.Helper()
	p := fmt.Sprintf("/api/competitions/%d/entries/%d/stages/%d", f.comp, entry, stage)
	if status, out := f.env.Do("PUT", p, map[string]any{"x": shots}); status != http.StatusOK {
		t.Fatalf("set counts: %d %v", status, out)
	}
}

func TestEntryChangesOnlyBetweenStages(t *testing.T) {
	f := newFixture(t, "150")
	e1 := f.addEntry(t, "One", 1)
	e2 := f.addEntry(t, "Two", 2)
	base := fmt.Sprintf("/api/competitions/%d", f.comp)
	f.env.Do("POST", base+"/status", map[string]any{"status": "active"})

	t.Run("between stages entries can be added", func(t *testing.T) {
		f.addEntry(t, "Three", 3)
	})

	lockStage(t, f, e1, 1, 12)

	t.Run("mid stage blocks add, update and withdraw", func(t *testing.T) {
		shooter := f.env.ID("POST", "/api/shooters", shooterBody("Late"))
		status, _ := f.env.Do("POST", base+"/entries", map[string]any{"shooter_id": shooter, "category": "Pistol", "lane": 9})
		if status != http.StatusConflict {
			t.Fatalf("add status %d", status)
		}
		status, _ = f.env.Do("PUT", fmt.Sprintf("%s/entries/%d", base, e2), map[string]any{"category": "Pistol", "lane": 8})
		if status != http.StatusConflict {
			t.Fatalf("update status %d", status)
		}
		status, _ = f.env.Do("POST", fmt.Sprintf("%s/entries/%d/withdraw", base, e2), nil)
		if status != http.StatusConflict {
			t.Fatalf("withdraw status %d", status)
		}
	})
}

func TestEntriesFrozenWhenFinished(t *testing.T) {
	f := newFixture(t, "150")
	e1 := f.addEntry(t, "One", 1)
	base := fmt.Sprintf("/api/competitions/%d", f.comp)
	f.env.Do("POST", base+"/status", map[string]any{"status": "active"})
	f.env.Do("POST", base+"/status", map[string]any{"status": "finished"})
	status, _ := f.env.Do("PUT", fmt.Sprintf("%s/entries/%d", base, e1), map[string]any{"category": "Pistol", "lane": 4})
	if status != http.StatusConflict {
		t.Fatalf("status %d", status)
	}
}

func TestAdminRoutesRequireLogin(t *testing.T) {
	env := apitest.New(t)
	for _, path := range []string{"/api/shooters", "/api/competitions"} {
		if status, _ := env.Do("GET", path, nil); status != http.StatusUnauthorized {
			t.Fatalf("%s: status %d", path, status)
		}
	}
}
