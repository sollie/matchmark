package auth_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/sollie/matchmark/internal/auth"
	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/repository"
)

const middlewareTestJWTSecret = "testsecretthatis32byteslong12345"

func newMiddlewareTestRepo(t *testing.T) (*sql.DB, *repository.UserRepository) {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.Migrate(database); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return database, repository.NewUserRepository(database)
}

func seedMiddlewareTestUser(t *testing.T, users *repository.UserRepository) int64 {
	t.Helper()
	passwordHash, err := auth.HashPassword("supersecretpassword1234")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user, err := users.Create(t.Context(), "tester", passwordHash, "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user.ID
}

func TestRequireAuth_AllowsCurrentToken(t *testing.T) {
	_, users := newMiddlewareTestRepo(t)
	userID := seedMiddlewareTestUser(t, users)

	user, err := users.GetByID(t.Context(), userID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	token, err := auth.SignAccessToken(middlewareTestJWTSecret, user.ID, user.Username, user.Role, user.AuthVersion)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	h := auth.RequireAuth(middlewareTestJWTSecret, users)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

func TestRequireAuth_RejectsDeletedUserToken(t *testing.T) {
	_, users := newMiddlewareTestRepo(t)
	userID := seedMiddlewareTestUser(t, users)

	user, err := users.GetByID(t.Context(), userID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	token, err := auth.SignAccessToken(middlewareTestJWTSecret, user.ID, user.Username, user.Role, user.AuthVersion)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	if err := users.Delete(t.Context(), userID); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	h := auth.RequireAuth(middlewareTestJWTSecret, users)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestRequireAuth_RejectsAuthVersionMismatch(t *testing.T) {
	_, users := newMiddlewareTestRepo(t)
	userID := seedMiddlewareTestUser(t, users)

	user, err := users.GetByID(t.Context(), userID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	token, err := auth.SignAccessToken(middlewareTestJWTSecret, user.ID, user.Username, user.Role, user.AuthVersion)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	if err := users.BumpAuthVersion(t.Context(), userID); err != nil {
		t.Fatalf("bump auth version: %v", err)
	}

	h := auth.RequireAuth(middlewareTestJWTSecret, users)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}
