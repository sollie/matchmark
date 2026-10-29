package handler_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

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

func openScreenStream(env *apitest.Env, token string) *apitest.Stream {
	return env.OpenStream("/api/spectator/screens/stream", http.Header{"Authorization": {"Bearer " + token}})
}

func nextScreenState(t *testing.T, s *apitest.Stream) (string, any, string) {
	t.Helper()
	for {
		ev, ok := s.Next(3 * time.Second)
		if !ok {
			t.Fatal("no screen event received")
		}
		if ev.Name != "snapshot" {
			continue
		}
		var body map[string]any
		if err := json.Unmarshal([]byte(ev.Data), &body); err != nil {
			t.Fatal(err)
		}
		return body["state"].(string), body["match_id"], ev.Data
	}
}

func TestScreenStreamRejectsUnknownToken(t *testing.T) {
	env := apitest.New(t)
	if s := openScreenStream(env, "nope"); s.Status != http.StatusUnauthorized {
		t.Fatalf("status %d", s.Status)
	}
	if s := env.OpenStream("/api/spectator/screens/stream", nil); s.Status != http.StatusUnauthorized {
		t.Fatalf("no token: %d", s.Status)
	}
}

func TestScreenStreamResolvesAssignmentAndNeverLeaksDraft(t *testing.T) {
	env := apitest.NewAdmin(t)
	token, code := registerScreen(t, env)
	_, screen := env.Do("POST", "/api/screens/approve", map[string]any{"code": code})
	id := strconv.Itoa(int(screen["id"].(float64)))

	s := openScreenStream(env, token)
	if s.Status != http.StatusOK {
		t.Fatalf("status %d", s.Status)
	}
	if state, mid, _ := nextScreenState(t, s); state != "waiting" || mid != nil {
		t.Fatalf("empty: %s %v", state, mid)
	}

	draft := env.ID("POST", "/api/matches", map[string]any{"name": "Secret Draft Cup", "date": env.Today(), "course": "40"})
	env.Do("PUT", "/api/screens/"+id, map[string]any{"match_id": draft})
	if ev, ok := s.Next(time.Second); ok {
		t.Fatalf("draft assignment changed the screen: %+v", ev)
	}

	if status, _ := env.Do("POST", "/api/matches/"+strconv.Itoa(draft)+"/status", map[string]any{"status": "active"}); status != http.StatusOK {
		t.Fatalf("activate: %d", status)
	}
	if state, mid, _ := nextScreenState(t, s); state != "match" || mid != float64(draft) {
		t.Fatalf("started: %s %v", state, mid)
	}

	env.Do("POST", "/api/matches/"+strconv.Itoa(draft)+"/status", map[string]any{"status": "finished"})
	if ev, ok := s.Next(time.Second); ok {
		t.Fatalf("finish changed the screen: %+v", ev)
	}
	if state, mid, _ := openAndRead(t, env, token); state != "match" || mid != float64(draft) {
		t.Fatalf("finished: %s %v", state, mid)
	}

	if status, _ := env.Do("DELETE", "/api/matches/"+strconv.Itoa(draft), nil); status != http.StatusOK {
		t.Fatalf("delete: %d", status)
	}
	if state, mid, _ := nextScreenState(t, s); state != "waiting" || mid != nil {
		t.Fatalf("deleted: %s %v", state, mid)
	}
}

func TestUnmanagedScreenFollowsNewestActiveMatch(t *testing.T) {
	env := apitest.NewAdmin(t)
	token, _ := registerScreen(t, env)
	draft := env.ID("POST", "/api/matches", map[string]any{"name": "Draft", "date": env.Today(), "course": "40"})
	s := openScreenStream(env, token)
	if state, _, _ := nextScreenState(t, s); state != "waiting" {
		t.Fatalf("draft default: %s", state)
	}
	env.Do("POST", "/api/matches/"+strconv.Itoa(draft)+"/status", map[string]any{"status": "active"})
	if state, mid, _ := nextScreenState(t, s); state != "match" || mid != float64(draft) {
		t.Fatalf("active: %s %v", state, mid)
	}
}

func TestScreenStreamMarksScreenOnline(t *testing.T) {
	env := apitest.NewAdmin(t)
	token, code := registerScreen(t, env)
	env.Do("POST", "/api/screens/approve", map[string]any{"code": code})
	s := openScreenStream(env, token)
	nextScreenState(t, s)
	_, list := env.Do("GET", "/api/screens", nil)
	if list["screens"].([]any)[0].(map[string]any)["online"] != true {
		t.Fatalf("offline: %v", list)
	}
}

func openAndRead(t *testing.T, env *apitest.Env, token string) (string, any, string) {
	t.Helper()
	return nextScreenState(t, openScreenStream(env, token))
}
