package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/sollie/matchmark/internal/apitest"
)

const tempPassword = "temporary-password-12"

func createUser(env *apitest.Env, username, role, password string) (int, map[string]any) {
	return env.Do("POST", "/api/users", map[string]any{"username": username, "role": role, "password": password})
}

func TestAdminCreatesAndListsUsers(t *testing.T) {
	env := apitest.NewAdmin(t)
	status, out := createUser(env, "anna", "user", tempPassword)
	if status != http.StatusCreated || out["username"] != "anna" || out["role"] != "user" || out["active"] != true || out["must_change_password"] != true {
		t.Fatalf("create: %d %v", status, out)
	}
	if _, leaked := out["password_hash"]; leaked {
		t.Fatalf("hash leaked: %v", out)
	}

	status, list := env.Do("GET", "/api/users", nil)
	users, _ := list["users"].([]any)
	if status != http.StatusOK || len(users) != 2 {
		t.Fatalf("list: %d %v", status, list)
	}
	first := users[0].(map[string]any)
	if first["username"] != apitest.AdminUsername || first["role"] != "admin" || first["active"] != true || first["must_change_password"] != false {
		t.Fatalf("admin row: %v", first)
	}
}

func TestUsernameRules(t *testing.T) {
	env := apitest.NewAdmin(t)
	for _, name := range []string{"ab", strings.Repeat("a", 33), "has space", "bad/char", "åke", ""} {
		if status, out := createUser(env, name, "user", tempPassword); status != http.StatusUnprocessableEntity {
			t.Fatalf("%q: %d %v", name, status, out)
		}
	}
	for _, name := range []string{"abc", strings.Repeat("a", 32), "a.b_c-1"} {
		if status, out := createUser(env, name, "user", tempPassword); status != http.StatusCreated {
			t.Fatalf("%q: %d %v", name, status, out)
		}
	}
}

func TestUsernamesAreUniqueIgnoringCase(t *testing.T) {
	env := apitest.NewAdmin(t)
	createUser(env, "anna", "user", tempPassword)
	for _, name := range []string{"anna", "ANNA", strings.ToUpper(apitest.AdminUsername)} {
		if status, out := createUser(env, name, "user", tempPassword); status != http.StatusConflict {
			t.Fatalf("%q: %d %v", name, status, out)
		}
	}
}

func TestCreateUserValidatesRoleAndPassword(t *testing.T) {
	env := apitest.NewAdmin(t)
	if status, _ := createUser(env, "anna", "root", tempPassword); status != http.StatusUnprocessableEntity {
		t.Fatalf("role: %d", status)
	}
	if status, _ := createUser(env, "anna", "user", "short"); status != http.StatusUnprocessableEntity {
		t.Fatalf("short password: %d", status)
	}
	if status, _ := createUser(env, "anna", "user", strings.Repeat("x", 129)); status != http.StatusUnprocessableEntity {
		t.Fatalf("long password: %d", status)
	}
}

func TestUserManagementRequiresLogin(t *testing.T) {
	env := apitest.New(t)
	if status, _ := env.Do("GET", "/api/users", nil); status != http.StatusUnauthorized {
		t.Fatalf("list: %d", status)
	}
	if status, _ := createUser(env, "anna", "user", tempPassword); status != http.StatusUnauthorized {
		t.Fatalf("create: %d", status)
	}
}

func TestRegularUserIsRefusedUserManagementButRunsCompetitions(t *testing.T) {
	admin := apitest.NewAdmin(t)
	createUser(admin, "anna", "user", tempPassword)

	c := client(t)
	if resp := login(t, c, admin, "/api/admin/login", "anna", tempPassword); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("login: %d", resp.StatusCode)
	}
	if resp, _ := changePassword(t, c, admin, tempPassword, newPassword); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("change: %d", resp.StatusCode)
	}

	get := func(path string) int {
		resp, err := c.Get(admin.Server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		return resp.StatusCode
	}
	if got := get("/api/users"); got != http.StatusForbidden {
		t.Fatalf("list users: %d", got)
	}
	if resp, _ := postBody(t, c, admin, "/api/users", `{"username":"bob","role":"user","password":"`+tempPassword+`"}`); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("create user: %d", resp.StatusCode)
	}
	if got := get("/api/admin/backup"); got != http.StatusForbidden {
		t.Fatalf("backup: %d", got)
	}

	for _, path := range []string{"/api/admin/session", "/api/shooters", "/api/competitions"} {
		if got := get(path); got != http.StatusOK {
			t.Fatalf("%s: %d", path, got)
		}
	}
	resp, body := postBody(t, c, admin, "/api/shooters", `{"first_name":"Ola","last_name":"Hansen","sex":"M","country":"NOR","club":"Club"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create shooter: %d %s", resp.StatusCode, body)
	}
}

func TestSessionReportsUsernameAndRole(t *testing.T) {
	env := apitest.NewAdmin(t)
	status, out := env.Do("GET", "/api/admin/session", nil)
	if status != http.StatusOK || out["username"] != apitest.AdminUsername || out["role"] != "admin" {
		t.Fatalf("session: %d %v", status, out)
	}
}
