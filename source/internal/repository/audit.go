package repository

import (
	"context"
	"strings"
	"time"
)

const auditTimeLayout = "2006-01-02T15:04:05.000Z"

type AuditEntry struct {
	ID        int64
	CreatedAt time.Time
	UserID    *int
	Username  string
	Action    string
	Target    string
	Details   string
	IP        string
}

type AuditFilter struct {
	Username string
	Action   string
	From     time.Time
	To       time.Time
	Limit    int
	Offset   int
}

func (r *Repository) InsertAudit(ctx context.Context, e AuditEntry) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_log (created_at, user_id, username, action, target, details, ip) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.CreatedAt.UTC().Format(auditTimeLayout), e.UserID, e.Username, e.Action, e.Target, e.Details, e.IP,
	)
	return mapErr(err)
}

func auditWhere(f AuditFilter) (string, []any) {
	var conds []string
	var args []any
	if f.Username != "" {
		conds = append(conds, `username = ? COLLATE NOCASE`)
		args = append(args, f.Username)
	}
	if f.Action != "" {
		conds = append(conds, `action = ?`)
		args = append(args, f.Action)
	}
	if !f.From.IsZero() {
		conds = append(conds, `created_at >= ?`)
		args = append(args, f.From.UTC().Format(auditTimeLayout))
	}
	if !f.To.IsZero() {
		conds = append(conds, `created_at < ?`)
		args = append(args, f.To.UTC().Format(auditTimeLayout))
	}
	if len(conds) == 0 {
		return "", nil
	}
	return ` WHERE ` + strings.Join(conds, ` AND `), args
}

func (r *Repository) ListAudit(ctx context.Context, f AuditFilter) ([]AuditEntry, int, error) {
	where, args := auditWhere(f)
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_log`+where, args...).Scan(&total); err != nil {
		return nil, 0, mapErr(err)
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, created_at, user_id, username, action, target, details, ip FROM audit_log`+where+` ORDER BY id DESC LIMIT ? OFFSET ?`,
		append(args, f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	defer func() { _ = rows.Close() }()
	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var created string
		if err := rows.Scan(&e.ID, &created, &e.UserID, &e.Username, &e.Action, &e.Target, &e.Details, &e.IP); err != nil {
			return nil, 0, mapErr(err)
		}
		e.CreatedAt, _ = time.Parse(auditTimeLayout, created)
		out = append(out, e)
	}
	return out, total, mapErr(rows.Err())
}
