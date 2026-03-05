package service_test

import (
	"context"
	"errors"
	"testing"

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
