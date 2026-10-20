package repository

import "time"

type User struct {
	ID                 int
	Username           string
	Role               string
	Active             bool
	CanDownloadBackup  bool
	MustChangePassword bool
	PasswordHash       string
}

const userColumns = `id, username, role, active, can_download_backup, must_change_password, password_hash`

type scanner interface {
	Scan(dest ...any) error
}

func scanUser(s scanner) (User, error) {
	var u User
	err := s.Scan(&u.ID, &u.Username, &u.Role, &u.Active, &u.CanDownloadBackup, &u.MustChangePassword, &u.PasswordHash)
	return u, mapErr(err)
}

func (r *Repository) CountUsers() (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, mapErr(err)
}

func (r *Repository) CreateUser(username, role, passwordHash string, canDownloadBackup, mustChangePassword bool) (int, error) {
	res, err := r.db.Exec(
		`INSERT INTO users (username, role, can_download_backup, must_change_password, password_hash) VALUES (?, ?, ?, ?, ?)`,
		username, role, canDownloadBackup, mustChangePassword, passwordHash,
	)
	if err != nil {
		return 0, mapErr(err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *Repository) ListUsers() ([]User, error) {
	rows, err := r.db.Query(`SELECT ` + userColumns + ` FROM users ORDER BY id`)
	if err != nil {
		return nil, mapErr(err)
	}
	defer func() { _ = rows.Close() }()
	var users []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, mapErr(rows.Err())
}

func (r *Repository) UserByUsername(username string) (User, error) {
	return scanUser(r.db.QueryRow(`SELECT `+userColumns+` FROM users WHERE username = ?`, username))
}

func (r *Repository) PasswordHash(username string) (string, error) {
	u, err := r.UserByUsername(username)
	return u.PasswordHash, err
}

func (r *Repository) SetPasswordHash(username, hash string) error {
	_, err := r.db.Exec(`UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE username = ?`, hash, username)
	return mapErr(err)
}

func (r *Repository) CreateSession(tokenHash string, userID int, expiresAt time.Time) error {
	_, err := r.db.Exec(
		`INSERT INTO user_sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?)`,
		tokenHash, userID, expiresAt.UTC().Format(time.RFC3339),
	)
	return mapErr(err)
}

func (r *Repository) SessionUser(tokenHash string, now time.Time) (User, error) {
	return scanUser(r.db.QueryRow(
		`SELECT u.id, u.username, u.role, u.active, u.can_download_backup, u.must_change_password, u.password_hash
		 FROM user_sessions s JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = ? AND s.expires_at > ? AND u.active = 1`,
		tokenHash, now.UTC().Format(time.RFC3339),
	))
}

func (r *Repository) DeleteSession(tokenHash string) error {
	_, err := r.db.Exec(`DELETE FROM user_sessions WHERE token_hash = ?`, tokenHash)
	return mapErr(err)
}

func (r *Repository) DeleteExpiredSessions(now time.Time) error {
	_, err := r.db.Exec(`DELETE FROM user_sessions WHERE expires_at <= ?`, now.UTC().Format(time.RFC3339))
	return mapErr(err)
}

func (r *Repository) ResetPassword(userID int, hash string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return mapErr(err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(
		`UPDATE users SET password_hash = ?, must_change_password = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		hash, userID,
	); err != nil {
		return mapErr(err)
	}
	if _, err := tx.Exec(`DELETE FROM user_sessions WHERE user_id = ?`, userID); err != nil {
		return mapErr(err)
	}
	return mapErr(tx.Commit())
}

func (r *Repository) ChangePassword(userID int, hash, keepTokenHash string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return mapErr(err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(
		`UPDATE users SET password_hash = ?, must_change_password = 0, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		hash, userID,
	); err != nil {
		return mapErr(err)
	}
	if _, err := tx.Exec(`DELETE FROM user_sessions WHERE user_id = ? AND token_hash <> ?`, userID, keepTokenHash); err != nil {
		return mapErr(err)
	}
	return mapErr(tx.Commit())
}

func (r *Repository) UserByID(id int) (User, error) {
	return scanUser(r.db.QueryRow(`SELECT `+userColumns+` FROM users WHERE id = ?`, id))
}

func (r *Repository) CountActiveAdmins() (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'admin' AND active = 1`).Scan(&n)
	return n, mapErr(err)
}

func (r *Repository) SetActive(userID int, active bool) error {
	tx, err := r.db.Begin()
	if err != nil {
		return mapErr(err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`UPDATE users SET active = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, active, userID); err != nil {
		return mapErr(err)
	}
	if !active {
		if _, err := tx.Exec(`DELETE FROM user_sessions WHERE user_id = ?`, userID); err != nil {
			return mapErr(err)
		}
	}
	return mapErr(tx.Commit())
}

func (r *Repository) SetRole(userID int, role string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return mapErr(err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(
		`UPDATE users SET role = ?, can_download_backup = CASE WHEN ? = 'admin' THEN can_download_backup ELSE 0 END, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		role, role, userID,
	); err != nil {
		return mapErr(err)
	}
	if role != "admin" {
		if _, err := tx.Exec(`DELETE FROM user_sessions WHERE user_id = ?`, userID); err != nil {
			return mapErr(err)
		}
	}
	return mapErr(tx.Commit())
}
