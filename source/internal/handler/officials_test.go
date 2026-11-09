package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/sollie/matchmark/internal/apitest"
	"github.com/sollie/matchmark/internal/wa1500"
)

func officialsEnv(t *testing.T) (*apitest.Env, *fakeRegister) {
	t.Helper()
	fake := &fakeRegister{}
	fake.set([]wa1500.Shooter{
		{WAID: "1001", FirstName: "Rita", LastName: "REFEREE", Country: "Norway", Club: "A"},
		{WAID: "2002", FirstName: "Ron", LastName: "RANGE", Country: "Norway", Club: "B"},
	}, nil)
	return newRegisterEnv(t, fake), fake
}

func matchBody(extra map[string]any) map[string]any {
	body := map[string]any{"name": "Club Match", "date": "2026-05-01", "course": "150"}
	for k, v := range extra {
		body[k] = v
	}
	return body
}

func getMatch(t *testing.T, env *apitest.Env, id int) map[string]any {
	t.Helper()
	status, out := env.Do("GET", fmt.Sprintf("/api/matches/%d", id), nil)
	if status != http.StatusOK {
		t.Fatalf("get: %d %v", status, out)
	}
	return out
}

func TestMatchOfficialsAreLookedUpAndSnapshotted(t *testing.T) {
	env, _ := officialsEnv(t)
	status, out := env.Do("POST", "/api/matches", matchBody(map[string]any{"referee_waid": "1001", "range_officer_waid": " 2002 "}))
	if status != http.StatusCreated {
		t.Fatalf("create: %d %v", status, out)
	}
	m := getMatch(t, env, int(out["id"].(float64)))
	if m["referee_waid"] != "1001" || m["referee_name"] != "Rita REFEREE" || m["range_officer_waid"] != "2002" || m["range_officer_name"] != "Ron RANGE" {
		t.Fatalf("officials: %v", m)
	}
}

func TestMatchOfficialsAreEmptyByDefault(t *testing.T) {
	env, fake := officialsEnv(t)
	m := getMatch(t, env, newMatch(env))
	if m["referee_waid"] != nil || m["referee_name"] != nil || m["range_officer_waid"] != nil || m["range_officer_name"] != nil {
		t.Fatalf("officials: %v", m)
	}
	if fake.calls() != 0 {
		t.Fatalf("lookups: %d", fake.calls())
	}
}

func TestMatchRefereeDefaultsToSignedInUserWithWAID(t *testing.T) {
	env, _ := officialsEnv(t)
	if status, out := env.Do("PUT", "/api/admin/waid", map[string]any{"waid": "1001"}); status != http.StatusOK || out["waid"] != "1001" {
		t.Fatalf("set waid: %d %v", status, out)
	}
	m := getMatch(t, env, newMatch(env))
	if m["referee_waid"] != "1001" || m["referee_name"] != "Rita REFEREE" || m["range_officer_waid"] != nil {
		t.Fatalf("default: %v", m)
	}
	m = getMatch(t, env, env.ID("POST", "/api/matches", matchBody(map[string]any{"referee_waid": ""})))
	if m["referee_waid"] != nil {
		t.Fatalf("explicit empty: %v", m)
	}
}

func TestMatchOfficialLookupFailureStoresNothing(t *testing.T) {
	env, fake := officialsEnv(t)
	id := env.ID("POST", "/api/matches", matchBody(map[string]any{"referee_waid": "1001"}))

	status, out := env.Do("POST", "/api/matches", matchBody(map[string]any{"range_officer_waid": "9999"}))
	if status != http.StatusUnprocessableEntity || out["error"] != "WAID not found in the WA1500 register" {
		t.Fatalf("not found: %d %v", status, out)
	}
	if _, list := env.Do("GET", "/api/matches", nil); len(list["matches"].([]any)) != 1 {
		t.Fatalf("partial match created: %v", list)
	}

	fake.set(nil, wa1500.ErrUpstream)
	path := fmt.Sprintf("/api/matches/%d", id)
	status, out = env.Do("PUT", path, matchBody(map[string]any{"name": "Renamed", "referee_waid": "1001", "range_officer_waid": "2002"}))
	if status != http.StatusServiceUnavailable {
		t.Fatalf("unreachable: %d %v", status, out)
	}
	m := getMatch(t, env, id)
	if m["name"] != "Club Match" || m["range_officer_waid"] != nil || m["referee_name"] != "Rita REFEREE" {
		t.Fatalf("partial update: %v", m)
	}
}

func TestMatchOfficialsKeepOnUnrelatedEditAndCanBeCleared(t *testing.T) {
	env, fake := officialsEnv(t)
	id := env.ID("POST", "/api/matches", matchBody(map[string]any{"referee_waid": "1001"}))
	fake.set(nil, wa1500.ErrUpstream)
	path := fmt.Sprintf("/api/matches/%d", id)
	if status, out := env.Do("PUT", path, matchBody(map[string]any{"name": "Renamed"})); status != http.StatusOK || out["referee_name"] != "Rita REFEREE" {
		t.Fatalf("omitted: %d %v", status, out)
	}
	if status, out := env.Do("PUT", path, matchBody(map[string]any{"name": "Renamed", "referee_waid": "1001"})); status != http.StatusOK || out["referee_name"] != "Rita REFEREE" {
		t.Fatalf("unchanged: %d %v", status, out)
	}
	if status, out := env.Do("PUT", path, matchBody(map[string]any{"name": "Renamed", "referee_waid": ""})); status != http.StatusOK || out["referee_waid"] != nil || out["referee_name"] != nil {
		t.Fatalf("cleared: %d %v", status, out)
	}
}

func TestMatchOfficialsEditableAfterMatchStarts(t *testing.T) {
	env, _ := officialsEnv(t)
	id := newMatch(env)
	if status, out := env.Do("POST", fmt.Sprintf("/api/matches/%d/status", id), map[string]any{"status": "active"}); status != http.StatusOK {
		t.Fatalf("activate: %d %v", status, out)
	}
	status, out := env.Do("PUT", fmt.Sprintf("/api/matches/%d", id), matchBody(map[string]any{"range_officer_waid": "2002"}))
	if status != http.StatusOK || out["range_officer_name"] != "Ron RANGE" {
		t.Fatalf("active edit: %d %v", status, out)
	}
}

func TestMatchOfficialsAreAudited(t *testing.T) {
	env, _ := officialsEnv(t)
	env.ID("POST", "/api/matches", matchBody(map[string]any{"referee_waid": "1001"}))
	found := false
	for _, line := range env.AuditLines() {
		if line["action"] == "match.create" && line["details"] == "Club Match, referee 1001" {
			found = true
		}
	}
	if !found {
		t.Fatalf("audit: %v", env.AuditLines())
	}
}

func TestUserWAIDIsOptionalUniqueAndAdminManaged(t *testing.T) {
	env, _ := officialsEnv(t)
	status, out := env.Do("GET", "/api/users", nil)
	if status != http.StatusOK || out["users"].([]any)[0].(map[string]any)["waid"] != nil {
		t.Fatalf("list: %d %v", status, out)
	}
	status, out = env.Do("POST", "/api/users", map[string]any{"username": "second", "role": "user", "password": "a-long-enough-password"})
	if status != http.StatusCreated {
		t.Fatalf("create: %d %v", status, out)
	}
	second := int(out["id"].(float64))
	path := fmt.Sprintf("/api/users/%d/waid", second)
	if status, out := env.Do("PUT", path, map[string]any{"waid": " 3003 "}); status != http.StatusOK || out["waid"] != "3003" {
		t.Fatalf("set: %d %v", status, out)
	}
	if status, _ := env.Do("PUT", "/api/admin/waid", map[string]any{"waid": "3003"}); status != http.StatusConflict {
		t.Fatalf("duplicate: %d", status)
	}
	if status, _ := env.Do("PUT", path, map[string]any{"waid": string(make([]byte, 33))}); status != http.StatusUnprocessableEntity {
		t.Fatalf("too long: %d", status)
	}
	if status, out := env.Do("PUT", path, map[string]any{"waid": ""}); status != http.StatusOK || out["waid"] != nil {
		t.Fatalf("clear: %d %v", status, out)
	}
	if status, out := env.Do("PUT", "/api/admin/waid", map[string]any{"waid": "3003"}); status != http.StatusOK || out["waid"] != "3003" {
		t.Fatalf("own: %d %v", status, out)
	}
	if _, session := env.Do("GET", "/api/admin/session", nil); session["waid"] != "3003" {
		t.Fatalf("session: %v", session)
	}
}
