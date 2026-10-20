package service_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

func newAuth(t *testing.T) *service.Auth {
	t.Helper()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return service.NewAuth(repository.New(conn))
}

func TestSeedAdminRefusesMissingOrInvalidInput(t *testing.T) {
	tests := []struct {
		name               string
		username, password string
		want               error
	}{
		{"missing username", "", "1234567890123456", service.ErrSeedMissing},
		{"missing password", "admin", "", service.ErrSeedMissing},
		{"invalid username", "Ad min", "1234567890123456", service.ErrInvalidUsername},
		{"short username", "ab", "1234567890123456", service.ErrInvalidUsername},
		{"fifteen char password", "admin", "123456789012345", service.ErrWeakPassword},
		{"129 char password", "admin", strings.Repeat("x", 129), service.ErrWeakPassword},
		{"sixteen char password", "admin", "1234567890123456", nil},
		{"128 char password", "admin", strings.Repeat("x", 128), nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := newAuth(t).SeedAdmin(tc.username, tc.password)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestSeedVariablesAreIgnoredOnceUsersExist(t *testing.T) {
	auth := newAuth(t)
	if err := auth.SeedAdmin("first", "first-password-123456"); err != nil {
		t.Fatal(err)
	}
	if err := auth.SeedAdmin("", ""); err != nil {
		t.Fatalf("empty seed with existing users: %v", err)
	}
	if err := auth.SeedAdmin("second", "second-password-12345"); err != nil {
		t.Fatalf("seed with existing users: %v", err)
	}
	if _, _, err := auth.Login("c", "second", "second-password-12345"); !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("second user logged in: %v", err)
	}
	if _, _, err := auth.Login("c", "first", "first-password-123456"); err != nil {
		t.Fatalf("first user: %v", err)
	}
}
