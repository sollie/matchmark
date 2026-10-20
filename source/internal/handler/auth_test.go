package handler_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
	"time"

	"github.com/sollie/matchmark/internal/apitest"
)

func client(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar}
}

func postBody(t *testing.T, c *http.Client, env *apitest.Env, path, body string) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequest("POST", env.Server.URL+path, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, string(out)
}

func post(t *testing.T, c *http.Client, env *apitest.Env, path, password string) *http.Response {
	t.Helper()
	return login(t, c, env, path, apitest.AdminUsername, password)
}

func login(t *testing.T, c *http.Client, env *apitest.Env, path, username, password string) *http.Response {
	t.Helper()
	resp, _ := postBody(t, c, env, path, `{"username":"`+username+`","password":"`+password+`"}`)
	return resp
}

func sessionStatus(t *testing.T, c *http.Client, env *apitest.Env) int {
	t.Helper()
	resp, err := c.Get(env.Server.URL + "/api/admin/session")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

func TestAdminLoginLogout(t *testing.T) {
	env := apitest.New(t)
	c := client(t)

	if got := sessionStatus(t, c, env); got != http.StatusUnauthorized {
		t.Fatalf("before login: status %d", got)
	}
	if got := post(t, c, env, "/api/admin/login", "wrong-password-wrong").StatusCode; got != http.StatusUnauthorized {
		t.Fatalf("wrong password: status %d", got)
	}
	if got := sessionStatus(t, c, env); got != http.StatusUnauthorized {
		t.Fatalf("after failed login: status %d", got)
	}

	resp := post(t, c, env, "/api/admin/login", apitest.AdminPassword)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("login: status %d", resp.StatusCode)
	}
	if got := sessionStatus(t, c, env); got != http.StatusOK {
		t.Fatalf("after login: status %d", got)
	}

	if got := post(t, c, env, "/api/admin/logout", "").StatusCode; got != http.StatusNoContent {
		t.Fatalf("logout: status %d", got)
	}
	if got := sessionStatus(t, c, env); got != http.StatusUnauthorized {
		t.Fatalf("after logout: status %d", got)
	}
}

func TestLogoutInvalidatesStolenCookie(t *testing.T) {
	env := apitest.New(t)
	c := client(t)
	resp := post(t, c, env, "/api/admin/login", apitest.AdminPassword)
	cookie := resp.Cookies()[0]
	post(t, c, env, "/api/admin/logout", "")

	req, _ := http.NewRequest("GET", env.Server.URL+"/api/admin/session", nil)
	req.AddCookie(cookie)
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = r.Body.Close()
	if r.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", r.StatusCode)
	}
}

func TestSessionCookieAttributes(t *testing.T) {
	tests := []struct {
		baseURL string
		secure  bool
	}{
		{"http://localhost:8080", false},
		{"https://scores.example.org", true},
	}
	for _, tc := range tests {
		t.Run(tc.baseURL, func(t *testing.T) {
			env := apitest.NewWithBaseURL(t, tc.baseURL)
			resp := post(t, client(t), env, "/api/admin/login", apitest.AdminPassword)
			cookies := resp.Cookies()
			if len(cookies) != 1 {
				t.Fatalf("cookies: %v", cookies)
			}
			c := cookies[0]
			if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Secure != tc.secure || c.Path != "/" {
				t.Fatalf("cookie attributes: %+v", c)
			}
		})
	}
}

func TestLoginRateLimited(t *testing.T) {
	env := apitest.New(t)
	c := client(t)
	for i := 0; i < 5; i++ {
		if got := post(t, c, env, "/api/admin/login", "wrong-password-wrong").StatusCode; got != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status %d", i, got)
		}
	}
	if got := post(t, c, env, "/api/admin/login", "wrong-password-wrong").StatusCode; got != http.StatusTooManyRequests {
		t.Fatalf("over limit: status %d", got)
	}
	if got := post(t, c, env, "/api/admin/login", apitest.AdminPassword).StatusCode; got != http.StatusTooManyRequests {
		t.Fatalf("correct password while limited: status %d", got)
	}
}

func TestUnknownUsernameAndWrongPasswordLookTheSame(t *testing.T) {
	env := apitest.New(t)
	wrongResp, wrongBody := postBody(t, client(t), env, "/api/admin/login", `{"username":"admin","password":"wrong-password-wrong"}`)
	unknownResp, unknownBody := postBody(t, client(t), env, "/api/admin/login", `{"username":"nobody","password":"wrong-password-wrong"}`)
	if wrongResp.StatusCode != http.StatusUnauthorized || unknownResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("statuses %d, %d", wrongResp.StatusCode, unknownResp.StatusCode)
	}
	if wrongBody != unknownBody || len(unknownResp.Cookies()) != 0 || len(wrongResp.Cookies()) != 0 {
		t.Fatalf("responses differ: %q vs %q", wrongBody, unknownBody)
	}
}

func TestLoginUsernameIsCaseInsensitive(t *testing.T) {
	env := apitest.New(t)
	if got := login(t, client(t), env, "/api/admin/login", "ADMIN", apitest.AdminPassword).StatusCode; got != http.StatusNoContent {
		t.Fatalf("status %d", got)
	}
}

func TestSessionExpiresAfterTwelveHours(t *testing.T) {
	env := apitest.New(t)
	c := client(t)
	resp := post(t, c, env, "/api/admin/login", apitest.AdminPassword)
	if got := resp.Cookies()[0].MaxAge; got != int((12 * time.Hour).Seconds()) {
		t.Fatalf("max age %d", got)
	}
	env.Advance(11*time.Hour + 59*time.Minute)
	if got := sessionStatus(t, c, env); got != http.StatusOK {
		t.Fatalf("before expiry: status %d", got)
	}
	env.Advance(2 * time.Minute)
	if got := sessionStatus(t, c, env); got != http.StatusUnauthorized {
		t.Fatalf("after expiry: status %d", got)
	}
}

func TestStoredHashIsArgon2idWithPerUserSalt(t *testing.T) {
	env := apitest.New(t)
	hash := env.PasswordHash(apitest.AdminUsername)
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$") || strings.Contains(hash, apitest.AdminPassword) {
		t.Fatalf("hash %q", hash)
	}
	other := apitest.New(t)
	if other.PasswordHash(apitest.AdminUsername) == hash {
		t.Fatal("same password produced the same hash")
	}
}

func TestWeakerStoredHashIsUpgradedAtLogin(t *testing.T) {
	env := apitest.New(t)
	weak := apitest.LegacyHash(apitest.AdminPassword)
	env.SetPasswordHash(apitest.AdminUsername, weak)

	if got := post(t, client(t), env, "/api/admin/login", "wrong-password-wrong").StatusCode; got != http.StatusUnauthorized {
		t.Fatalf("wrong password against weak hash: status %d", got)
	}
	if env.PasswordHash(apitest.AdminUsername) != weak {
		t.Fatal("hash changed after failed login")
	}
	if got := post(t, client(t), env, "/api/admin/login", apitest.AdminPassword).StatusCode; got != http.StatusNoContent {
		t.Fatalf("login with weak hash: status %d", got)
	}
	upgraded := env.PasswordHash(apitest.AdminUsername)
	if upgraded == weak || !strings.HasPrefix(upgraded, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("hash not upgraded: %q", upgraded)
	}
	if got := post(t, client(t), env, "/api/admin/login", apitest.AdminPassword).StatusCode; got != http.StatusNoContent {
		t.Fatalf("login after upgrade: status %d", got)
	}
}

const newPassword = "another-long-password-1"

func changePassword(t *testing.T, c *http.Client, env *apitest.Env, current, next string) (*http.Response, string) {
	t.Helper()
	return postBody(t, c, env, "/api/admin/password", `{"current_password":"`+current+`","new_password":"`+next+`"}`)
}

func TestChangePassword(t *testing.T) {
	env := apitest.New(t)
	c := client(t)
	other := client(t)
	post(t, c, env, "/api/admin/login", apitest.AdminPassword)
	post(t, other, env, "/api/admin/login", apitest.AdminPassword)

	if resp, _ := changePassword(t, c, env, "wrong-current-password", newPassword); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong current: status %d", resp.StatusCode)
	}
	if resp, _ := changePassword(t, c, env, apitest.AdminPassword, "short"); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("weak new: status %d", resp.StatusCode)
	}
	if got := sessionStatus(t, other, env); got != http.StatusOK {
		t.Fatalf("other session after rejected changes: %d", got)
	}

	if resp, _ := changePassword(t, c, env, apitest.AdminPassword, newPassword); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("change: status %d", resp.StatusCode)
	}
	if got := sessionStatus(t, c, env); got != http.StatusOK {
		t.Fatalf("own session after change: %d", got)
	}
	if got := sessionStatus(t, other, env); got != http.StatusUnauthorized {
		t.Fatalf("other session after change: %d", got)
	}
	fresh := client(t)
	if got := post(t, fresh, env, "/api/admin/login", apitest.AdminPassword).StatusCode; got != http.StatusUnauthorized {
		t.Fatalf("old password: status %d", got)
	}
	if got := post(t, fresh, env, "/api/admin/login", newPassword).StatusCode; got != http.StatusNoContent {
		t.Fatalf("new password: status %d", got)
	}
}

func TestChangePasswordRequiresLogin(t *testing.T) {
	env := apitest.New(t)
	resp, _ := changePassword(t, client(t), env, apitest.AdminPassword, newPassword)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestTemporaryPasswordRestrictsUserUntilChanged(t *testing.T) {
	env := apitest.New(t)
	env.SetTemporaryPassword(apitest.AdminUsername, "temporary-password-12")
	c := client(t)
	if got := post(t, c, env, "/api/admin/login", "temporary-password-12").StatusCode; got != http.StatusNoContent {
		t.Fatalf("login: status %d", got)
	}
	for _, path := range []string{"/api/admin/session", "/api/shooters", "/api/competitions", "/api/admin/backup"} {
		resp, err := c.Get(env.Server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden || !strings.Contains(string(body), "password_change_required") {
			t.Fatalf("%s: status %d body %s", path, resp.StatusCode, body)
		}
	}

	if resp, _ := changePassword(t, c, env, "temporary-password-12", newPassword); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("change: status %d", resp.StatusCode)
	}
	if got := sessionStatus(t, c, env); got != http.StatusOK {
		t.Fatalf("session after change: %d", got)
	}
	resp, err := c.Get(env.Server.URL + "/api/shooters")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("shooters after change: %d", resp.StatusCode)
	}
}

func TestFlaggedUserCanLogOut(t *testing.T) {
	env := apitest.New(t)
	env.SetTemporaryPassword(apitest.AdminUsername, "temporary-password-12")
	c := client(t)
	post(t, c, env, "/api/admin/login", "temporary-password-12")
	if got := post(t, c, env, "/api/admin/logout", "").StatusCode; got != http.StatusNoContent {
		t.Fatalf("logout: status %d", got)
	}
}
