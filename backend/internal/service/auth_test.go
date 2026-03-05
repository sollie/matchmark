package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sollie/matchmark/internal/auth"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

const testJWTSecret = "testsecretthatis32byteslong12345"

func newAuthServiceWithUser(t *testing.T) (*service.AuthService, string) {
	t.Helper()
	database := newTestDB(t)
	userRepo := repository.NewUserRepository(database)

	passwordHash, err := auth.HashPassword("supersecretpassword1234")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	_, err = userRepo.Create(context.Background(), "tester", passwordHash, "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	return service.NewAuthService(userRepo, database, testJWTSecret), "tester"
}

func TestRotateRefreshToken_Success(t *testing.T) {
	svc, username := newAuthServiceWithUser(t)
	ctx := context.Background()

	loginResult, err := svc.Login(ctx, username, "supersecretpassword1234")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	refreshResult, err := svc.Refresh(ctx, loginResult.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}

	if refreshResult.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if refreshResult.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if refreshResult.RefreshToken == loginResult.RefreshToken {
		t.Error("expected a new refresh token, got the same one")
	}
}

func TestRotateRefreshToken_ReuseDetected(t *testing.T) {
	svc, username := newAuthServiceWithUser(t)
	ctx := context.Background()

	loginResult, err := svc.Login(ctx, username, "supersecretpassword1234")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	_, err = svc.Refresh(ctx, loginResult.RefreshToken)
	if err != nil {
		t.Fatalf("first refresh: %v", err)
	}

	_, err = svc.Refresh(ctx, loginResult.RefreshToken)
	if !errors.Is(err, service.ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken on reuse, got %v", err)
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	svc, _ := newAuthServiceWithUser(t)
	ctx := context.Background()

	_, err := svc.Refresh(ctx, "notavalidtoken")
	if !errors.Is(err, service.ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}
