package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

const (
	oldPassword = "old-password-123456"
	newPassword = "temporary-password-1"
)

func resetFixture(t *testing.T) (func(string) string, *service.Auth, *repository.Repository) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "reset.db")
	conn, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	repo := repository.New(conn)
	auth := service.NewAuth(repo)
	if err := auth.SeedAdmin("Admin", oldPassword); err != nil {
		t.Fatal(err)
	}
	getenv := func(k string) string {
		if k == "DB_PATH" {
			return dbPath
		}
		return ""
	}
	return getenv, auth, repo
}

func TestResetPasswordCommandSetsTemporaryPasswordAndEndsSessions(t *testing.T) {
	getenv, auth, repo := resetFixture(t)
	token, _, err := auth.Login("ip", "admin", oldPassword)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := runResetPassword([]string{"Admin"}, getenv, strings.NewReader(newPassword+"\n"), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "admin") || strings.Contains(out.String(), newPassword) {
		t.Fatalf("output = %s", out.String())
	}
	if _, err := auth.Authenticate(token); !errors.Is(err, service.ErrUnauthenticated) {
		t.Fatalf("old session err = %v", err)
	}
	if _, _, err := auth.Login("ip", "admin", oldPassword); !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("old password err = %v", err)
	}
	if _, _, err := auth.Login("ip", "admin", newPassword); err != nil {
		t.Fatal(err)
	}
	u, err := repo.UserByUsername("admin")
	if err != nil || !u.MustChangePassword {
		t.Fatalf("user = %+v, %v", u, err)
	}
}

func TestResetPasswordCommandRejectsUnknownUser(t *testing.T) {
	getenv, _, _ := resetFixture(t)
	err := runResetPassword([]string{"nobody"}, getenv, strings.NewReader(newPassword+"\n"), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown user") {
		t.Fatalf("err = %v", err)
	}
}

func TestResetPasswordCommandRejectsWeakPassword(t *testing.T) {
	getenv, auth, _ := resetFixture(t)
	err := runResetPassword([]string{"admin"}, getenv, strings.NewReader("short\n"), &bytes.Buffer{})
	if !errors.Is(err, service.ErrWeakPassword) {
		t.Fatalf("err = %v", err)
	}
	if _, _, err := auth.Login("ip", "admin", oldPassword); err != nil {
		t.Fatalf("password changed: %v", err)
	}
}

func TestResetPasswordCommandUsage(t *testing.T) {
	getenv, _, _ := resetFixture(t)
	for _, args := range [][]string{nil, {"a", "b"}} {
		if err := runResetPassword(args, getenv, strings.NewReader(newPassword), &bytes.Buffer{}); err == nil {
			t.Fatalf("args %v: expected error", args)
		}
	}
	if err := runResetPassword([]string{"admin"}, getenv, strings.NewReader(""), &bytes.Buffer{}); err == nil {
		t.Fatal("expected error for empty stdin")
	}
}

func TestResetPasswordCommandRefusesMissingDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "never.db")
	getenv := func(k string) string {
		if k == "DB_PATH" {
			return dbPath
		}
		return ""
	}
	if err := runResetPassword([]string{"admin"}, getenv, strings.NewReader(newPassword), &bytes.Buffer{}); err == nil {
		t.Fatal("expected error")
	}
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("stat err = %v", err)
	}
}
