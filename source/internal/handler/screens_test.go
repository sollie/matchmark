package handler_test

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/sollie/matchmark/internal/apitest"
)

func registerScreen(t *testing.T, env *apitest.Env) (token, code string) {
	t.Helper()
	status, out := env.Do("POST", "/api/spectator/screens", nil)
	if status != http.StatusCreated {
		t.Fatalf("register: %d %v", status, out)
	}
	return out["token"].(string), out["code"].(string)
}

func TestScreenLifecycle(t *testing.T) {
	env := apitest.NewAdmin(t)
	token, code := registerScreen(t, env)
	if len(code) != 6 || len(token) < 20 {
		t.Fatalf("token %q code %q", token, code)
	}

	_, list := env.Do("GET", "/api/screens", nil)
	if screens := list["screens"].([]any); len(screens) != 0 {
		t.Fatalf("pending screen listed: %v", list)
	}
	if status, _ := env.Do("POST", "/api/screens/approve", map[string]any{"code": "NOPE00"}); status != http.StatusNotFound {
		t.Fatalf("unknown code: %d", status)
	}
	status, screen := env.Do("POST", "/api/screens/approve", map[string]any{"code": code})
	if status != http.StatusOK || screen["name"] != "Screen 1" || screen["online"] != true || screen["match_id"] != nil {
		t.Fatalf("approve: %d %v", status, screen)
	}
	if status, _ := env.Do("POST", "/api/screens/approve", map[string]any{"code": code}); status != http.StatusNotFound {
		t.Fatalf("second approve: %d", status)
	}
	id := strconv.Itoa(int(screen["id"].(float64)))

	draft := env.ID("POST", "/api/matches", map[string]any{"name": "Draft", "date": env.Today(), "course": "40"})
	status, out := env.Do("PUT", "/api/screens/"+id, map[string]any{"name": "Gate", "match_id": draft})
	if status != http.StatusOK || out["name"] != "Gate" || out["match_id"] != float64(draft) {
		t.Fatalf("update: %d %v", status, out)
	}
	if status, out = env.Do("PUT", "/api/screens/"+id, map[string]any{"match_id": nil}); status != http.StatusOK || out["match_id"] != nil || out["name"] != "Gate" {
		t.Fatalf("clear: %d %v", status, out)
	}
	if status, _ = env.Do("PUT", "/api/screens/"+id, map[string]any{"match_id": 9999}); status != http.StatusNotFound {
		t.Fatalf("missing match: %d", status)
	}
	if status, _ = env.Do("PUT", "/api/screens/"+id, map[string]any{"name": " "}); status != http.StatusUnprocessableEntity {
		t.Fatalf("blank name: %d", status)
	}

	if status, out = env.Do("PUT", "/api/screens/assignment", map[string]any{"match_id": draft}); status != http.StatusOK || out["assigned"] != float64(1) {
		t.Fatalf("assign all: %d %v", status, out)
	}
	_, matches := env.Do("GET", "/api/screens/assignable-matches", nil)
	if len(matches["matches"].([]any)) != 1 {
		t.Fatalf("assignable: %v", matches)
	}

	if status, _ = env.DoToken(token, "DELETE", "/api/spectator/screens/self", nil); status != http.StatusNoContent {
		t.Fatalf("leave: %d", status)
	}
	if status, _ = env.DoToken(token, "DELETE", "/api/spectator/screens/self", nil); status != http.StatusUnauthorized {
		t.Fatalf("leave again: %d", status)
	}
	_, list = env.Do("GET", "/api/screens", nil)
	if len(list["screens"].([]any)) != 0 {
		t.Fatalf("screen not removed: %v", list)
	}
}

func TestAdminRemovesScreen(t *testing.T) {
	env := apitest.NewAdmin(t)
	token, code := registerScreen(t, env)
	_, screen := env.Do("POST", "/api/screens/approve", map[string]any{"code": code, "name": "Hall"})
	id := strconv.Itoa(int(screen["id"].(float64)))
	if status, _ := env.Do("DELETE", "/api/screens/"+id, nil); status != http.StatusNoContent {
		t.Fatalf("remove: %d", status)
	}
	if status, _ := env.DoToken(token, "DELETE", "/api/spectator/screens/self", nil); status != http.StatusUnauthorized {
		t.Fatalf("removed screen token: %d", status)
	}
}

func TestScreenRegistrationIsRateLimited(t *testing.T) {
	env := apitest.New(t)
	for range 5 {
		registerScreen(t, env)
	}
	if status, _ := env.Do("POST", "/api/spectator/screens", nil); status != http.StatusTooManyRequests {
		t.Fatalf("sixth registration: %d", status)
	}
}

func TestScreenAdminEndpointsRequireLogin(t *testing.T) {
	env := apitest.New(t)
	for _, c := range [][2]string{{"GET", "/api/screens"}, {"POST", "/api/screens/approve"}, {"PUT", "/api/screens/assignment"}, {"GET", "/api/screens/assignable-matches"}, {"PUT", "/api/screens/1"}, {"DELETE", "/api/screens/1"}} {
		if status, _ := env.Do(c[0], c[1], nil); status != http.StatusUnauthorized {
			t.Fatalf("%s %s: %d", c[0], c[1], status)
		}
	}
}
