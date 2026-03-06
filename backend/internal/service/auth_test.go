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

func TestLogout_RevokesRefreshToken(t *testing.T) {
	database := newTestDB(t)
	userRepo := repository.NewUserRepository(database)
	passwordHash, err := auth.HashPassword("supersecretpassword1234")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user, err := userRepo.Create(context.Background(), "tester", passwordHash, "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	svc := service.NewAuthService(userRepo, database, testJWTSecret)

	loginResult, err := svc.Login(context.Background(), user.Username, "supersecretpassword1234")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if err := svc.Logout(context.Background(), loginResult.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := svc.Refresh(context.Background(), loginResult.RefreshToken); !errors.Is(err, service.ErrInvalidToken) {
		t.Fatalf("expected invalid token after logout, got %v", err)
	}
}

func TestRefresh_RejectsAfterPasswordReset(t *testing.T) {
	database := newTestDB(t)
	userRepo := repository.NewUserRepository(database)
	passwordHash, err := auth.HashPassword("supersecretpassword1234")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user, err := userRepo.Create(context.Background(), "tester", passwordHash, "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	authSvc := service.NewAuthService(userRepo, database, testJWTSecret)
	userSvc := service.NewUserService(userRepo)

	loginResult, err := authSvc.Login(context.Background(), user.Username, "supersecretpassword1234")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if err := userSvc.ResetPassword(context.Background(), user.ID, "newpassword12345678"); err != nil {
		t.Fatalf("reset password: %v", err)
	}
	if _, err := authSvc.Refresh(context.Background(), loginResult.RefreshToken); !errors.Is(err, service.ErrInvalidToken) {
		t.Fatalf("expected invalid token after password reset, got %v", err)
	}
}

func TestRefresh_RejectsAfterRoleChange(t *testing.T) {
	database := newTestDB(t)
	userRepo := repository.NewUserRepository(database)
	passwordHash, err := auth.HashPassword("supersecretpassword1234")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user, err := userRepo.Create(context.Background(), "tester", passwordHash, "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	authSvc := service.NewAuthService(userRepo, database, testJWTSecret)
	userSvc := service.NewUserService(userRepo)

	loginResult, err := authSvc.Login(context.Background(), user.Username, "supersecretpassword1234")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	role := "user"
	if _, err := userSvc.Update(context.Background(), user.ID, service.UpdateUserParams{Role: &role}); err != nil {
		t.Fatalf("update role: %v", err)
	}
	if _, err := authSvc.Refresh(context.Background(), loginResult.RefreshToken); !errors.Is(err, service.ErrInvalidToken) {
		t.Fatalf("expected invalid token after role change, got %v", err)
	}
	updated, err := userRepo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get updated user: %v", err)
	}
	claims, err := auth.VerifyAccessToken(testJWTSecret, loginResult.AccessToken)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if claims.AuthVer == updated.AuthVersion {
		t.Fatalf("expected token auth version to differ after role change")
	}
	if claims.Role == updated.Role {
		t.Fatalf("expected token role to differ after role change")
	}
}

func TestAccessTokenClaimsIncludeAuthVersion(t *testing.T) {
	svc, username := newAuthServiceWithUser(t)
	result, err := svc.Login(context.Background(), username, "supersecretpassword1234")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	claims, err := auth.VerifyAccessToken(testJWTSecret, result.AccessToken)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if claims.AuthVer != 0 {
		t.Fatalf("expected initial auth version 0, got %d", claims.AuthVer)
	}
}
