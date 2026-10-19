package handler_test

import (
	"bytes"
	"net/http"
	"net/http/cookiejar"
	"testing"

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

func post(t *testing.T, c *http.Client, env *apitest.Env, path, password string) *http.Response {
	t.Helper()
	body := `{"password":"` + password + `"}`
	req, err := http.NewRequest("POST", env.Server.URL+path, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
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
	if got := sessionStatus(t, c, env); got != http.StatusNoContent {
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
