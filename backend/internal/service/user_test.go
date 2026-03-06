package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sollie/matchmark/internal/auth"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

func newUserSvc(t *testing.T) *service.UserService {
	t.Helper()
	database := newTestDB(t)
	return service.NewUserService(repository.NewUserRepository(database))
}

func TestUserService_Create_InvalidRole(t *testing.T) {
	svc := newUserSvc(t)

	_, err := svc.Create(context.Background(), "alice", "supersecretpassword1234", "superuser")
	if !errors.Is(err, service.ErrInvalidRole) {
		t.Errorf("expected ErrInvalidRole, got %v", err)
	}
}

func TestUserService_Create_Success(t *testing.T) {
	svc := newUserSvc(t)

	u, err := svc.Create(context.Background(), "alice", "supersecretpassword1234", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Username != "alice" {
		t.Errorf("expected username alice, got %s", u.Username)
	}
	if u.Role != "user" {
		t.Errorf("expected role user, got %s", u.Role)
	}
}

func TestUserService_Update_NotFound(t *testing.T) {
	svc := newUserSvc(t)

	_, err := svc.Update(context.Background(), 999, service.UpdateUserParams{})
	if !errors.Is(err, service.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserService_Update_InvalidRole(t *testing.T) {
	database := newTestDB(t)
	svc := service.NewUserService(repository.NewUserRepository(database))
	userID := seedUser(t, database)

	bad := "superuser"
	_, err := svc.Update(context.Background(), userID, service.UpdateUserParams{Role: &bad})
	if !errors.Is(err, service.ErrInvalidRole) {
		t.Errorf("expected ErrInvalidRole, got %v", err)
	}
}

func TestUserService_Update_Success(t *testing.T) {
	database := newTestDB(t)
	svc := service.NewUserService(repository.NewUserRepository(database))
	userID := seedUser(t, database)

	newName := "scorer_renamed"
	u, err := svc.Update(context.Background(), userID, service.UpdateUserParams{Username: &newName})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Username != newName {
		t.Errorf("expected username %s, got %s", newName, u.Username)
	}
}

func TestUserService_Delete_CannotDeleteSelf(t *testing.T) {
	database := newTestDB(t)
	svc := service.NewUserService(repository.NewUserRepository(database))
	userID := seedUser(t, database)

	err := svc.Delete(context.Background(), userID, userID)
	if !errors.Is(err, service.ErrCannotDeleteSelf) {
		t.Errorf("expected ErrCannotDeleteSelf, got %v", err)
	}
}

func TestUserService_Delete_NotFound(t *testing.T) {
	svc := newUserSvc(t)

	err := svc.Delete(context.Background(), 999, 1)
	if !errors.Is(err, service.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserService_Delete_Success(t *testing.T) {
	database := newTestDB(t)
	svc := service.NewUserService(repository.NewUserRepository(database))
	userID := seedUser(t, database)

	callerID := userID + 1
	if err := svc.Delete(context.Background(), userID, callerID); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUserService_ResetPassword_NotFound(t *testing.T) {
	svc := newUserSvc(t)

	err := svc.ResetPassword(context.Background(), 999, "supersecretpassword1234")
	if !errors.Is(err, service.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserService_ResetPassword_Success(t *testing.T) {
	database := newTestDB(t)
	svc := service.NewUserService(repository.NewUserRepository(database))
	userID := seedUser(t, database)

	if err := svc.ResetPassword(context.Background(), userID, "newpassword12345678"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUserService_Update_RoleChangeBumpsAuthVersionAndRevokesRefreshTokens(t *testing.T) {
	database := newTestDB(t)
	userRepo := repository.NewUserRepository(database)
	userSvc := service.NewUserService(userRepo)
	authSvc := service.NewAuthService(userRepo, database, testJWTSecret)

	passwordHash, err := auth.HashPassword("supersecretpassword1234")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user, err := userRepo.Create(context.Background(), "tester", passwordHash, "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	loginResult, err := authSvc.Login(context.Background(), user.Username, "supersecretpassword1234")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	role := "user"
	updated, err := userSvc.Update(context.Background(), user.ID, service.UpdateUserParams{Role: &role})
	if err != nil {
		t.Fatalf("update role: %v", err)
	}
	if updated.AuthVersion != 1 {
		t.Fatalf("expected auth version 1, got %d", updated.AuthVersion)
	}
	if _, err := authSvc.Refresh(context.Background(), loginResult.RefreshToken); !errors.Is(err, service.ErrInvalidToken) {
		t.Fatalf("expected invalid refresh token after role change, got %v", err)
	}
}

func TestUserService_ResetPassword_BumpsAuthVersionAndRevokesRefreshTokens(t *testing.T) {
	database := newTestDB(t)
	userRepo := repository.NewUserRepository(database)
	userSvc := service.NewUserService(userRepo)
	authSvc := service.NewAuthService(userRepo, database, testJWTSecret)

	passwordHash, err := auth.HashPassword("supersecretpassword1234")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user, err := userRepo.Create(context.Background(), "tester", passwordHash, "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	loginResult, err := authSvc.Login(context.Background(), user.Username, "supersecretpassword1234")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if err := userSvc.ResetPassword(context.Background(), user.ID, "newpassword12345678"); err != nil {
		t.Fatalf("reset password: %v", err)
	}
	updated, err := userRepo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if updated.AuthVersion != 1 {
		t.Fatalf("expected auth version 1, got %d", updated.AuthVersion)
	}
	if _, err := authSvc.Refresh(context.Background(), loginResult.RefreshToken); !errors.Is(err, service.ErrInvalidToken) {
		t.Fatalf("expected invalid refresh token after password reset, got %v", err)
	}
}
