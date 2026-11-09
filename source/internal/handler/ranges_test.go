package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/sollie/matchmark/internal/apitest"
)

func newRange(t *testing.T, env *apitest.Env, name string) int {
	t.Helper()
	return env.ID("POST", "/api/ranges", map[string]any{"name": name})
}

func newCompetition(env *apitest.Env, extra map[string]any) (int, map[string]any) {
	body := map[string]any{"name": "Open", "start_date": "2026-05-01", "end_date": "2026-05-01"}
	for k, v := range extra {
		body[k] = v
	}
	status, out := env.Do("POST", "/api/competitions", body)
	if status != http.StatusCreated {
		return status, out
	}
	_, got := env.Do("GET", fmt.Sprintf("/api/competitions/%v", out["id"]), nil)
	return status, got
}

func TestRangeCrudAndValidation(t *testing.T) {
	env := apitest.NewAdmin(t)
	id := newRange(t, env, "  Indoor 25m  ")
	status, out := env.Do("GET", "/api/ranges", nil)
	list := out["ranges"].([]any)
	if status != http.StatusOK || len(list) != 1 || list[0].(map[string]any)["name"] != "Indoor 25m" || list[0].(map[string]any)["archived"] != false {
		t.Fatalf("list: %d %v", status, out)
	}
	if status, _ := env.Do("POST", "/api/ranges", map[string]any{"name": "indoor 25M"}); status != http.StatusConflict {
		t.Fatalf("duplicate: %d", status)
	}
	if status, _ := env.Do("POST", "/api/ranges", map[string]any{"name": "  "}); status != http.StatusUnprocessableEntity {
		t.Fatalf("blank: %d", status)
	}
	status, out = env.Do("PUT", fmt.Sprintf("/api/ranges/%d", id), map[string]any{"name": "Outdoor"})
	if status != http.StatusOK || out["name"] != "Outdoor" {
		t.Fatalf("rename: %d %v", status, out)
	}
	if status, _ := env.Do("DELETE", fmt.Sprintf("/api/ranges/%d", id), nil); status != http.StatusOK {
		t.Fatalf("delete: %d", status)
	}
	if status, _ := env.Do("DELETE", fmt.Sprintf("/api/ranges/%d", id), nil); status != http.StatusNotFound {
		t.Fatalf("delete again: %d", status)
	}
}

func TestRegularUserListsRangesButCannotChangeThem(t *testing.T) {
	admin := apitest.NewAdmin(t)
	id := newRange(t, admin, "Indoor")
	createUser(admin, "anna", "user", tempPassword)
	c := client(t)
	login(t, c, admin, "/api/admin/login", "anna", tempPassword)
	changePassword(t, c, admin, tempPassword, newPassword)
	resp, err := c.Get(admin.Server.URL + "/api/ranges")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: %d", resp.StatusCode)
	}
	if resp, _ := postBody(t, c, admin, "/api/ranges", `{"name":"Outdoor"}`); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("create: %d", resp.StatusCode)
	}
	if resp, _ := postBody(t, c, admin, fmt.Sprintf("/api/ranges/%d/archive", id), ``); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("archive: %d", resp.StatusCode)
	}
}

func TestDefaultRangePrefillsAndSingleRangeAutoSelects(t *testing.T) {
	env := apitest.NewAdmin(t)
	if _, c := newCompetition(env, nil); c["range_id"] != nil {
		t.Fatalf("no ranges: %v", c)
	}
	a := newRange(t, env, "A")
	if _, c := newCompetition(env, nil); c["range_id"] != float64(a) || c["range_name"] != "A" {
		t.Fatalf("single range: %v", c)
	}
	b := newRange(t, env, "B")
	if _, c := newCompetition(env, nil); c["range_id"] != nil {
		t.Fatalf("two ranges: %v", c)
	}
	if status, out := env.Do("PUT", "/api/club", map[string]any{"default_range_id": b}); status != http.StatusOK || out["default_range_id"] != float64(b) {
		t.Fatalf("default: %d %v", status, out)
	}
	if _, c := newCompetition(env, nil); c["range_id"] != float64(b) {
		t.Fatalf("default prefill: %v", c)
	}
	if _, c := newCompetition(env, map[string]any{"range_id": a}); c["range_id"] != float64(a) {
		t.Fatalf("explicit: %v", c)
	}
	if _, c := newCompetition(env, map[string]any{"range_id": nil}); c["range_id"] != nil {
		t.Fatalf("explicit none: %v", c)
	}
	if status, out := env.Do("PUT", "/api/club", map[string]any{"default_range_id": nil}); status != http.StatusOK || out["default_range_id"] != nil {
		t.Fatalf("clear default: %d %v", status, out)
	}
}

func TestUsedRangeCannotBeDeletedButCanBeArchived(t *testing.T) {
	env := apitest.NewAdmin(t)
	a := newRange(t, env, "A")
	b := newRange(t, env, "B")
	_, comp := newCompetition(env, map[string]any{"range_id": a})
	if status, _ := env.Do("DELETE", fmt.Sprintf("/api/ranges/%d", a), nil); status != http.StatusConflict {
		t.Fatalf("delete used: %d", status)
	}
	if status, out := env.Do("POST", fmt.Sprintf("/api/ranges/%d/archive", a), nil); status != http.StatusOK || out["archived"] != true {
		t.Fatalf("archive: %d %v", status, out)
	}
	_, got := env.Do("GET", fmt.Sprintf("/api/competitions/%v", comp["id"]), nil)
	if got["range_id"] != float64(a) || got["range_name"] != "A" {
		t.Fatalf("kept on competition: %v", got)
	}
	if status, _ := newCompetition(env, map[string]any{"range_id": a}); status != http.StatusUnprocessableEntity {
		t.Fatalf("archived pick: %d", status)
	}
	if status, out := env.Do("PUT", fmt.Sprintf("/api/competitions/%v", comp["id"]), map[string]any{"name": "Renamed", "start_date": "2026-05-01", "end_date": "2026-05-01"}); status != http.StatusOK || out["range_id"] != float64(a) {
		t.Fatalf("update keeps archived: %d %v", status, out)
	}
	if status, out := env.Do("PUT", fmt.Sprintf("/api/competitions/%v", comp["id"]), map[string]any{"name": "Renamed", "start_date": "2026-05-01", "end_date": "2026-05-01", "range_id": b}); status != http.StatusOK || out["range_id"] != float64(b) {
		t.Fatalf("update range: %d %v", status, out)
	}
	if status, _ := env.Do("DELETE", fmt.Sprintf("/api/ranges/%d", a), nil); status != http.StatusOK {
		t.Fatalf("delete unused: %d", status)
	}
}

func TestDefaultRangeCannotBeArchivedOrDeletedAndMustBeActive(t *testing.T) {
	env := apitest.NewAdmin(t)
	a := newRange(t, env, "A")
	b := newRange(t, env, "B")
	env.Do("PUT", "/api/club", map[string]any{"default_range_id": a})
	if status, _ := env.Do("POST", fmt.Sprintf("/api/ranges/%d/archive", a), nil); status != http.StatusConflict {
		t.Fatalf("archive default: %d", status)
	}
	if status, _ := env.Do("DELETE", fmt.Sprintf("/api/ranges/%d", a), nil); status != http.StatusConflict {
		t.Fatalf("delete default: %d", status)
	}
	env.Do("POST", fmt.Sprintf("/api/ranges/%d/archive", b), nil)
	if status, _ := env.Do("PUT", "/api/club", map[string]any{"default_range_id": b}); status != http.StatusUnprocessableEntity {
		t.Fatalf("archived default: %d", status)
	}
	if status, _ := env.Do("PUT", "/api/club", map[string]any{"default_range_id": 999}); status != http.StatusUnprocessableEntity {
		t.Fatalf("unknown default: %d", status)
	}
	if status, out := env.Do("POST", fmt.Sprintf("/api/ranges/%d/unarchive", b), nil); status != http.StatusOK || out["archived"] != false {
		t.Fatalf("unarchive: %d %v", status, out)
	}
}

func TestRangeChangesAreAudited(t *testing.T) {
	env := apitest.NewAdmin(t)
	id := newRange(t, env, "A")
	env.Do("PUT", fmt.Sprintf("/api/ranges/%d", id), map[string]any{"name": "B"})
	env.Do("POST", fmt.Sprintf("/api/ranges/%d/archive", id), nil)
	env.Do("POST", fmt.Sprintf("/api/ranges/%d/unarchive", id), nil)
	env.Do("PUT", "/api/club", map[string]any{"default_range_id": id})
	env.Do("PUT", "/api/club", map[string]any{"default_range_id": nil})
	env.Do("DELETE", fmt.Sprintf("/api/ranges/%d", id), nil)
	seen := map[any]bool{}
	for _, l := range env.AuditLines() {
		seen[l["action"]] = true
	}
	for _, want := range []string{"range.create", "range.update", "range.archive", "range.unarchive", "range.delete", "club.default_range"} {
		if !seen[want] {
			t.Errorf("missing %s in %v", want, seen)
		}
	}
}

func TestResultsExposeTheRangeName(t *testing.T) {
	f := resultsFixture(t)
	if _, out := publicGet(t, f, resultsPath(f)); out["match"].(map[string]any)["range"] != nil {
		t.Fatalf("no range: %v", out["match"])
	}
	id := newRange(t, f.env, "Outdoor")
	_, match := f.env.Do("GET", fmt.Sprintf("/api/matches/%d", f.comp), nil)
	status, out := f.env.Do("PUT", fmt.Sprintf("/api/competitions/%v", match["competition_id"]), map[string]any{"name": "Open", "start_date": "2026-01-01", "end_date": "2027-12-31", "range_id": id})
	if status != http.StatusOK {
		t.Fatalf("update: %d %v", status, out)
	}
	_, got := publicGet(t, f, resultsPath(f))
	if got["match"].(map[string]any)["range"] != "Outdoor" {
		t.Fatalf("range: %v", got["match"])
	}
}
