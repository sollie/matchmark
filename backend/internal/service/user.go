package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/sollie/matchmark/internal/auth"
	"github.com/sollie/matchmark/internal/model"
	"github.com/sollie/matchmark/internal/repository"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrUsernameTaken    = errors.New("username already taken")
	ErrInvalidRole      = errors.New("role must be 'user' or 'admin'")
	ErrCannotDeleteSelf = errors.New("cannot delete your own account")
)

type UserService struct {
	users *repository.UserRepository
}

func NewUserService(users *repository.UserRepository) *UserService {
	return &UserService{users: users}
}

func (s *UserService) GetByID(ctx context.Context, id int64) (*model.User, error) {
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

func (s *UserService) List(ctx context.Context) ([]model.User, error) {
	return s.users.List(ctx)
}

func (s *UserService) Create(ctx context.Context, username, password, role string) (*model.User, error) {
	if role != "user" && role != "admin" {
		return nil, ErrInvalidRole
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	u, err := s.users.Create(ctx, username, hash, role)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

type UpdateUserParams struct {
	Username *string
	Role     *string
}

func (s *UserService) Update(ctx context.Context, id int64, params UpdateUserParams) (*model.User, error) {
	current, err := s.users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	username := current.Username
	role := current.Role

	if params.Username != nil {
		username = *params.Username
	}
	if params.Role != nil {
		if *params.Role != "user" && *params.Role != "admin" {
			return nil, ErrInvalidRole
		}
		role = *params.Role
	}

	u, err := s.users.Update(ctx, id, username, role)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("update user: %w", err)
	}
	if params.Role != nil && *params.Role != current.Role {
		if err := s.users.BumpAuthVersion(ctx, id); err != nil {
			return nil, fmt.Errorf("bump auth version: %w", err)
		}
		if err := s.users.DeleteRefreshTokensByUser(ctx, id); err != nil {
			return nil, fmt.Errorf("delete refresh tokens: %w", err)
		}
		u, err = s.users.GetByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("reload user: %w", err)
		}
	}
	return u, nil
}

func (s *UserService) Delete(ctx context.Context, id, callerID int64) error {
	if id == callerID {
		return ErrCannotDeleteSelf
	}
	if err := s.users.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (s *UserService) ResetPassword(ctx context.Context, id int64, newPassword string) error {
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := s.users.UpdatePassword(ctx, id, hash); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("update password: %w", err)
	}
	if err := s.users.DeleteRefreshTokensByUser(ctx, id); err != nil {
		return fmt.Errorf("delete refresh tokens: %w", err)
	}
	return nil
}
