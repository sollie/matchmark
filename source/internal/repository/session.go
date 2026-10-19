package repository

import "time"

func (r *Repository) CreateAdminSession(tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(
		`INSERT INTO admin_sessions (token_hash, expires_at) VALUES (?, ?)`,
		tokenHash, expiresAt.UTC().Format(time.RFC3339),
	)
	return mapErr(err)
}

func (r *Repository) AdminSessionValid(tokenHash string, now time.Time) (bool, error) {
	var n int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM admin_sessions WHERE token_hash = ? AND expires_at > ?`,
		tokenHash, now.UTC().Format(time.RFC3339),
	).Scan(&n)
	return n > 0, mapErr(err)
}

func (r *Repository) DeleteAdminSession(tokenHash string) error {
	_, err := r.db.Exec(`DELETE FROM admin_sessions WHERE token_hash = ?`, tokenHash)
	return mapErr(err)
}

func (r *Repository) DeleteExpiredAdminSessions(now time.Time) error {
	_, err := r.db.Exec(`DELETE FROM admin_sessions WHERE expires_at <= ?`, now.UTC().Format(time.RFC3339))
	return mapErr(err)
}
