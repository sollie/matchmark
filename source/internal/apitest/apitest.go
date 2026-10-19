package apitest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/handler"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

type Env struct {
	t      *testing.T
	Server *httptest.Server
	client *http.Client
}

const AdminPassword = "correct-horse-battery"

func New(t *testing.T) *Env {
	t.Helper()
	return NewWithBaseURL(t, "http://localhost")
}

func NewWithBaseURL(t *testing.T, baseURL string) *Env {
	t.Helper()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	repo := repository.New(conn)
	auth, err := service.NewAdminAuth(repo, AdminPassword)
	if err != nil {
		t.Fatalf("admin auth: %v", err)
	}
	srv := httptest.NewServer(handler.NewRouter(service.New(repo), handler.WithAdminAuth(auth, baseURL)))
	t.Cleanup(srv.Close)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	return &Env{t: t, Server: srv, client: &http.Client{Jar: jar}}
}

func NewAdmin(t *testing.T) *Env {
	t.Helper()
	e := New(t)
	if status, out := e.Do("POST", "/api/admin/login", map[string]string{"password": AdminPassword}); status != http.StatusNoContent {
		t.Fatalf("admin login: %d %v", status, out)
	}
	return e
}

func (e *Env) Do(method, path string, body any) (int, map[string]any) {
	e.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			e.t.Fatalf("encode body: %v", err)
		}
	}
	req, err := http.NewRequest(method, e.Server.URL+path, &buf)
	if err != nil {
		e.t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		e.t.Fatalf("do request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func (e *Env) ID(method, path string, body any) int {
	e.t.Helper()
	status, out := e.Do(method, path, body)
	if status != http.StatusCreated {
		e.t.Fatalf("%s %s: status %d body %v", method, path, status, out)
	}
	return int(out["id"].(float64))
}
