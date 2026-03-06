package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/sollie/matchmark/internal/model"
)

var ErrNotFound = errors.New("not found")

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, role, auth_version, created_at FROM users WHERE username = ?`,
		username,
	)
	return scanUser(row)
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, role, auth_version, created_at FROM users WHERE id = ?`,
		id,
	)
	return scanUser(row)
}

func (r *UserRepository) List(ctx context.Context) (_ []model.User, retErr error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, username, password_hash, role, auth_version, created_at FROM users ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && retErr == nil {
			retErr = cerr
		}
	}()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.AuthVersion, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) Create(ctx context.Context, username, passwordHash, role string) (*model.User, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)`,
		username, passwordHash, role,
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *UserRepository) Update(ctx context.Context, id int64, username, role string) (*model.User, error) {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET username = ?, role = ? WHERE id = ?`,
		username, role, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = ?, auth_version = auth_version + 1 WHERE id = ?`,
		passwordHash, id,
	)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepository) BumpAuthVersion(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET auth_version = auth_version + 1 WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("bump auth version: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepository) DeleteRefreshTokensByUser(ctx context.Context, userID int64) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("delete refresh tokens by user: %w", err)
	}
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepository) CreateRefreshToken(ctx context.Context, userID int64, tokenHash, familyID string, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, family_id, expires_at) VALUES (?, ?, ?, ?)`,
		userID, tokenHash, familyID, expiresAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (r *UserRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, token_hash, family_id, expires_at, created_at FROM refresh_tokens WHERE token_hash = ?`,
		tokenHash,
	)
	var rt model.RefreshToken
	if err := row.Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.FamilyID, &rt.ExpiresAt, &rt.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	return &rt, nil
}

func (r *UserRepository) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM refresh_tokens WHERE token_hash = ?`,
		tokenHash,
	)
	if err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}
	return nil
}

func scanUser(row *sql.Row) (*model.User, error) {
	var u model.User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.AuthVersion, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return &u, nil
}
