package handler_test

import (
	"net/http"
	"strconv"
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
	resp, body := postBody(t, c, admin, "/api/shooters", `{"first_name":"Ola","last_name":"HANSEN","sex":"M","country":"NOR","club":"Club"}`)
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

func userPath(id int, action string) string {
	return "/api/users/" + strconv.Itoa(id) + "/" + action
}

func newUserClient(t *testing.T, admin *apitest.Env, username, role string) (int, *http.Client) {
	t.Helper()
	status, out := createUser(admin, username, role, tempPassword)
	if status != http.StatusCreated {
		t.Fatalf("create %s: %d %v", username, status, out)
	}
	c := client(t)
	if resp := login(t, c, admin, "/api/admin/login", username, tempPassword); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("login %s: %d", username, resp.StatusCode)
	}
	if resp, _ := changePassword(t, c, admin, tempPassword, newPassword); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("change %s: %d", username, resp.StatusCode)
	}
	return int(out["id"].(float64)), c
}

func userRow(t *testing.T, admin *apitest.Env, username string) map[string]any {
	t.Helper()
	_, list := admin.Do("GET", "/api/users", nil)
	for _, u := range list["users"].([]any) {
		if m := u.(map[string]any); m["username"] == username {
			return m
		}
	}
	t.Fatalf("user %s not listed", username)
	return nil
}

func TestDeactivationEndsSessionsAndBlocksLoginUntilReactivated(t *testing.T) {
	admin := apitest.NewAdmin(t)
	id, c := newUserClient(t, admin, "anna", "user")

	status, out := admin.Do("POST", userPath(id, "deactivate"), nil)
	if status != http.StatusOK || out["active"] != false {
		t.Fatalf("deactivate: %d %v", status, out)
	}
	if got := sessionStatus(t, c, admin); got != http.StatusUnauthorized {
		t.Fatalf("session after deactivate: %d", got)
	}
	if resp := login(t, client(t), admin, "/api/admin/login", "anna", newPassword); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("login while deactivated: %d", resp.StatusCode)
	}
	if row := userRow(t, admin, "anna"); row["active"] != false {
		t.Fatalf("row: %v", row)
	}

	status, out = admin.Do("POST", userPath(id, "reactivate"), nil)
	if status != http.StatusOK || out["active"] != true {
		t.Fatalf("reactivate: %d %v", status, out)
	}
	if resp := login(t, client(t), admin, "/api/admin/login", "anna", newPassword); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("login after reactivate: %d", resp.StatusCode)
	}
}

func TestAdminResetsPasswordWhichForcesChangeAndRevokesSessions(t *testing.T) {
	admin := apitest.NewAdmin(t)
	id, c := newUserClient(t, admin, "anna", "user")

	if status, _ := admin.Do("POST", userPath(id, "reset-password"), map[string]string{"password": "short"}); status != http.StatusUnprocessableEntity {
		t.Fatalf("weak: %d", status)
	}
	if status, _ := admin.Do("POST", userPath(id, "reset-password"), map[string]string{"password": tempPassword}); status != http.StatusNoContent {
		t.Fatalf("reset: %d", status)
	}
	if got := sessionStatus(t, c, admin); got != http.StatusUnauthorized {
		t.Fatalf("old session: %d", got)
	}
	if resp := login(t, client(t), admin, "/api/admin/login", "anna", newPassword); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("old password: %d", resp.StatusCode)
	}
	fresh := client(t)
	if resp := login(t, fresh, admin, "/api/admin/login", "anna", tempPassword); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("temporary login: %d", resp.StatusCode)
	}
	if row := userRow(t, admin, "anna"); row["must_change_password"] != true {
		t.Fatalf("row: %v", row)
	}
}

func TestRoleChangesPromoteAndDemoteAndDemotionRevokesSessions(t *testing.T) {
	admin := apitest.NewAdmin(t)
	id, c := newUserClient(t, admin, "anna", "user")

	status, out := admin.Do("PUT", "/api/users/"+strconv.Itoa(id)+"/role", map[string]string{"role": "admin"})
	if status != http.StatusOK || out["role"] != "admin" {
		t.Fatalf("promote: %d %v", status, out)
	}
	if got := sessionStatus(t, c, admin); got != http.StatusOK {
		t.Fatalf("session after promote: %d", got)
	}
	if resp, err := c.Get(admin.Server.URL + "/api/users"); err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("promoted list: %v %v", resp, err)
	}

	status, out = admin.Do("PUT", "/api/users/"+strconv.Itoa(id)+"/role", map[string]string{"role": "user"})
	if status != http.StatusOK || out["role"] != "user" || out["can_download_backup"] != false {
		t.Fatalf("demote: %d %v", status, out)
	}
	if got := sessionStatus(t, c, admin); got != http.StatusUnauthorized {
		t.Fatalf("session after demote: %d", got)
	}

	if status, _ := admin.Do("PUT", "/api/users/"+strconv.Itoa(id)+"/role", map[string]string{"role": "root"}); status != http.StatusUnprocessableEntity {
		t.Fatalf("bad role: %d", status)
	}
}

func TestLastActiveAdminCannotBeDemotedOrDeactivated(t *testing.T) {
	admin := apitest.NewAdmin(t)
	_, me := admin.Do("GET", "/api/users", nil)
	adminID := int(me["users"].([]any)[0].(map[string]any)["id"].(float64))
	rolePath := "/api/users/" + strconv.Itoa(adminID) + "/role"

	if status, _ := admin.Do("POST", userPath(adminID, "deactivate"), nil); status != http.StatusConflict {
		t.Fatalf("deactivate last: %d", status)
	}
	if status, _ := admin.Do("PUT", rolePath, map[string]string{"role": "user"}); status != http.StatusConflict {
		t.Fatalf("demote last: %d", status)
	}

	otherID, _ := newUserClient(t, admin, "boss", "admin")
	if status, _ := admin.Do("POST", userPath(otherID, "deactivate"), nil); status != http.StatusOK {
		t.Fatalf("deactivate other admin: %d", status)
	}
	if status, _ := admin.Do("POST", userPath(adminID, "deactivate"), nil); status != http.StatusConflict {
		t.Fatalf("deactivated admins do not count: %d", status)
	}
	if status, _ := admin.Do("POST", userPath(otherID, "reactivate"), nil); status != http.StatusOK {
		t.Fatalf("reactivate: %d", status)
	}
	if status, _ := admin.Do("PUT", rolePath, map[string]string{"role": "user"}); status != http.StatusOK {
		t.Fatalf("demote with another active admin: %d", status)
	}
}

func TestUserLifecycleIsAdminOnlyAndUnknownUserIsNotFound(t *testing.T) {
	admin := apitest.NewAdmin(t)
	id, c := newUserClient(t, admin, "anna", "user")
	for _, action := range []string{"deactivate", "reactivate", "reset-password"} {
		resp, _ := postBody(t, c, admin, userPath(id, action), `{"password":"`+tempPassword+`"}`)
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("%s as user: %d", action, resp.StatusCode)
		}
	}
	if status, _ := admin.Do("POST", userPath(9999, "deactivate"), nil); status != http.StatusNotFound {
		t.Fatalf("unknown: %d", status)
	}
}
