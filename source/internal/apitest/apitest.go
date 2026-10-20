package apitest

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/argon2"

	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/handler"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

type Env struct {
	t      *testing.T
	Server *httptest.Server
	client *http.Client
	repo   *repository.Repository
	auth   *service.Auth

	mu     sync.Mutex
	offset time.Duration
	delays []time.Duration
	audit  syncBuffer
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func (e *Env) AuditLines() []map[string]any {
	e.t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(e.audit.String(), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			e.t.Fatalf("audit line %q: %v", line, err)
		}
		if m["msg"] == "audit" {
			out = append(out, m)
		}
	}
	return out
}

const (
	AdminUsername = "admin"
	AdminPassword = "correct-horse-battery"
)

func (e *Env) Advance(d time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.offset += d
}

func (e *Env) sleep(d time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.delays = append(e.delays, d)
	e.offset += d
}

func (e *Env) Delays() []time.Duration {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]time.Duration(nil), e.delays...)
}

func (e *Env) Today() string {
	return e.now().UTC().Format(time.DateOnly)
}

func (e *Env) now() time.Time {
	e.mu.Lock()
	defer e.mu.Unlock()
	return time.Now().Add(e.offset)
}

func LegacyHash(password string) string {
	salt := []byte("0123456789abcdef")
	key := argon2.IDKey([]byte(password), salt, 1, 8*1024, 1, 32)
	enc := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=19$m=8192,t=1,p=1$%s$%s", enc.EncodeToString(salt), enc.EncodeToString(key))
}

func (e *Env) PasswordHash(username string) string {
	e.t.Helper()
	hash, err := e.repo.PasswordHash(username)
	if err != nil {
		e.t.Fatalf("password hash: %v", err)
	}
	return hash
}

func (e *Env) SetPasswordHash(username, hash string) {
	e.t.Helper()
	if err := e.repo.SetPasswordHash(username, hash); err != nil {
		e.t.Fatalf("set password hash: %v", err)
	}
}

func (e *Env) SetTemporaryPassword(username, password string) {
	e.t.Helper()
	if err := e.auth.ResetPassword(username, password); err != nil {
		e.t.Fatalf("set temporary password: %v", err)
	}
}

func New(t *testing.T) *Env {
	t.Helper()
	return NewWithBaseURL(t, "http://localhost")
}

func NewWithBaseURL(t *testing.T, baseURL string, opts ...handler.Option) *Env {
	t.Helper()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	repo := repository.New(conn)
	e := &Env{t: t, repo: repo}
	audit := service.NewAudit(repo, slog.New(slog.NewJSONHandler(&e.audit, nil)))
	auth := service.NewAuth(repo, service.WithAuthClock(e.now), service.WithAuthSleep(e.sleep), service.WithAuthAudit(audit))
	e.auth = auth
	if err := auth.SeedAdmin(AdminUsername, AdminPassword); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	e.Server = httptest.NewServer(handler.NewRouter(service.New(repo, service.WithAudit(audit)), append([]handler.Option{handler.WithAuth(auth, baseURL), handler.WithAudit(audit)}, opts...)...))
	t.Cleanup(e.Server.Close)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	e.client = &http.Client{Jar: jar}
	return e
}

func NewAdmin(t *testing.T, opts ...handler.Option) *Env {
	t.Helper()
	e := NewWithBaseURL(t, "http://localhost", opts...)
	if status, out := e.Do("POST", "/api/admin/login", map[string]string{"username": AdminUsername, "password": AdminPassword}); status != http.StatusNoContent {
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

func (e *Env) DoToken(token, method, path string, body any) (int, map[string]any) {
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
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
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
