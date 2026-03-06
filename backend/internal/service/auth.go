package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sollie/matchmark/internal/auth"
	"github.com/sollie/matchmark/internal/model"
	"github.com/sollie/matchmark/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrTokenExpired       = errors.New("refresh token expired")
	ErrInvalidToken       = errors.New("invalid refresh token")
)

const accessTokenTTLSeconds = 3600

type AuthService struct {
	db        *sql.DB
	users     *repository.UserRepository
	jwtSecret string
}

func NewAuthService(users *repository.UserRepository, db *sql.DB, jwtSecret string) *AuthService {
	return &AuthService{db: db, users: users, jwtSecret: jwtSecret}
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

	accessToken, err := auth.SignAccessToken(s.jwtSecret, user.ID, user.Username, user.Role, user.AuthVersion)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	rawRefresh, hashedRefresh, expiresAt, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	familyID := uuid.NewString()
	if err := s.users.CreateRefreshToken(ctx, user.ID, hashedRefresh, familyID, expiresAt); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    accessTokenTTLSeconds,
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
		if err := s.users.DeleteRefreshToken(ctx, hashed); err != nil {
			return nil, fmt.Errorf("delete expired token: %w", err)
		}
		return nil, ErrTokenExpired
	}

	user, err := s.users.GetByID(ctx, rt.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	newRaw, newHashed, newExpiry, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	if err := s.rotateRefreshToken(ctx, hashed, rt.UserID, rt.FamilyID, newHashed, newExpiry); err != nil {
		return nil, fmt.Errorf("rotate refresh token: %w", err)
	}

	accessToken, err := auth.SignAccessToken(s.jwtSecret, user.ID, user.Username, user.Role, user.AuthVersion)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	return &RefreshResult{
		AccessToken:  accessToken,
		RefreshToken: newRaw,
		ExpiresIn:    accessTokenTTLSeconds,
	}, nil
}

func (s *AuthService) rotateRefreshToken(ctx context.Context, oldHash string, userID int64, familyID, newHash string, expiresAt time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE token_hash = ?`, oldHash); err != nil {
		return fmt.Errorf("delete old refresh token: %w", err)
	}

	var familyCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM refresh_tokens WHERE family_id = ?`, familyID).Scan(&familyCount); err != nil {
		return fmt.Errorf("check family: %w", err)
	}
	if familyCount > 0 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE family_id = ?`, familyID); err != nil {
			return fmt.Errorf("revoke token family: %w", err)
		}
		return fmt.Errorf("%w: token reuse detected", ErrInvalidToken)
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, family_id, expires_at) VALUES (?, ?, ?, ?)`,
		userID, newHash, familyID, expiresAt.UTC().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("insert new refresh token: %w", err)
	}

	return tx.Commit()
}

func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	hashed := auth.HashRefreshToken(rawRefreshToken)
	if err := s.users.DeleteRefreshToken(ctx, hashed); err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}
	return nil
}
