package repository

import (
	"context"
	"database/sql"
	"time"
)

type Screen struct {
	ID         int64
	TokenHash  string
	Code       string
	Name       string
	Approved   bool
	MatchID    *int64
	CreatedAt  time.Time
	LastSeenAt time.Time
}

const screenColumns = `id, token_hash, code, name, approved_at IS NOT NULL, match_id, created_at, last_seen_at`

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func scanScreen(s scanner) (Screen, error) {
	var sc Screen
	var match sql.NullInt64
	var created, seen string
	if err := s.Scan(&sc.ID, &sc.TokenHash, &sc.Code, &sc.Name, &sc.Approved, &match, &created, &seen); err != nil {
		return Screen{}, mapErr(err)
	}
	if match.Valid {
		sc.MatchID = &match.Int64
	}
	sc.CreatedAt, _ = time.Parse(time.RFC3339, created)
	sc.LastSeenAt, _ = time.Parse(time.RFC3339, seen)
	return sc, nil
}

func (r *Repository) CreateScreen(ctx context.Context, tokenHash, code string, now time.Time, limit int) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO screens (token_hash, code, created_at, last_seen_at)
		 SELECT ?, ?, ?, ? WHERE (SELECT COUNT(*) FROM screens) < ?`,
		tokenHash, code, formatTime(now), formatTime(now), limit)
	if err != nil {
		return mapErr(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrLimit
	}
	return nil
}

func (r *Repository) ScreenByTokenHash(ctx context.Context, tokenHash string) (Screen, error) {
	return scanScreen(r.db.QueryRowContext(ctx, `SELECT `+screenColumns+` FROM screens WHERE token_hash = ?`, tokenHash))
}

func (r *Repository) GetScreen(ctx context.Context, id int64) (Screen, error) {
	return scanScreen(r.db.QueryRowContext(ctx, `SELECT `+screenColumns+` FROM screens WHERE id = ?`, id))
}

func (r *Repository) ListApprovedScreens(ctx context.Context) ([]Screen, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+screenColumns+` FROM screens WHERE approved_at IS NOT NULL ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Screen
	for rows.Next() {
		s, err := scanScreen(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) TouchScreen(ctx context.Context, id int64, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE screens SET last_seen_at = ? WHERE id = ?`, formatTime(now), id)
	return err
}

func (r *Repository) ApproveScreenByCode(ctx context.Context, code, name string, now time.Time) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`UPDATE screens SET approved_at = ?,
		   name = CASE WHEN ? != '' THEN ? ELSE 'Screen ' || ((SELECT COUNT(*) FROM screens WHERE approved_at IS NOT NULL) + 1) END
		 WHERE code = ? AND approved_at IS NULL RETURNING id`,
		formatTime(now), name, name, code).Scan(&id)
	return id, mapErr(err)
}

func (r *Repository) RenameScreen(ctx context.Context, id int64, name string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE screens SET name = ? WHERE id = ? AND approved_at IS NOT NULL`, name, id)
	return affected(res, err)
}

func (r *Repository) AssignScreen(ctx context.Context, id int64, matchID *int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE screens SET match_id = ? WHERE id = ? AND approved_at IS NOT NULL`, matchID, id)
	return affected(res, err)
}

func (r *Repository) AssignAllScreens(ctx context.Context, matchID int64) (int64, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE screens SET match_id = ? WHERE approved_at IS NOT NULL`, matchID)
	if err != nil {
		return 0, mapErr(err)
	}
	return res.RowsAffected()
}

func (r *Repository) DeleteScreen(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM screens WHERE id = ?`, id)
	return affected(res, err)
}

func (r *Repository) PurgeScreensSeenBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM screens WHERE last_seen_at < ?`, formatTime(cutoff))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *Repository) ListAssignableMatches(ctx context.Context) ([]Match, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, competition_id, name, date, course, status, current_segment, COALESCE(activated_at, '')
		 FROM matches WHERE status != 'finished' ORDER BY date DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []Match{}
	for rows.Next() {
		var c Match
		if err := rows.Scan(&c.ID, &c.CompetitionID, &c.Name, &c.Date, &c.Course, &c.Status, &c.CurrentSegment, &c.ActivatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
