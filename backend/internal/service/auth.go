package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sollie/matchmark/internal/auth"
	"github.com/sollie/matchmark/internal/model"
	"github.com/sollie/matchmark/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrTokenExpired       = errors.New("refresh token expired")
	ErrInvalidToken       = errors.New("invalid refresh token")
)

type AuthService struct {
	users     *repository.UserRepository
	jwtSecret string
}

func NewAuthService(users *repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{users: users, jwtSecret: jwtSecret}
}

type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	User         *model.User
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	if err := auth.VerifyPassword(password, user.PasswordHash); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := auth.SignAccessToken(s.jwtSecret, user.ID, user.Username, user.Role)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	rawRefresh, hashedRefresh, expiresAt, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	if err := s.users.CreateRefreshToken(ctx, user.ID, hashedRefresh, expiresAt); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    3600,
		User:         user,
	}, nil
}

type RefreshResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken string) (*RefreshResult, error) {
	hashed := auth.HashRefreshToken(rawRefreshToken)

	rt, err := s.users.GetRefreshToken(ctx, hashed)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}

	expiry, err := time.Parse(time.RFC3339, rt.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("parse expiry: %w", err)
	}
	if time.Now().After(expiry) {
		_ = s.users.DeleteRefreshToken(ctx, hashed)
		return nil, ErrTokenExpired
	}

	user, err := s.users.GetByID(ctx, rt.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	if err := s.users.DeleteRefreshToken(ctx, hashed); err != nil {
		return nil, fmt.Errorf("delete old refresh token: %w", err)
	}

	newRaw, newHashed, newExpiry, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	if err := s.users.CreateRefreshToken(ctx, user.ID, newHashed, newExpiry); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	accessToken, err := auth.SignAccessToken(s.jwtSecret, user.ID, user.Username, user.Role)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	return &RefreshResult{
		AccessToken:  accessToken,
		RefreshToken: newRaw,
		ExpiresIn:    3600,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	hashed := auth.HashRefreshToken(rawRefreshToken)
	if err := s.users.DeleteRefreshToken(ctx, hashed); err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}
	return nil
}
