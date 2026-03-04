package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/sollie/matchmark/internal/model"
)

type CompetitionRepository struct {
	db *sql.DB
}

func NewCompetitionRepository(db *sql.DB) *CompetitionRepository {
	return &CompetitionRepository{db: db}
}

const competitionCols = `id, name, date, lanes_total, status, spectator_enabled, created_at`

func scanCompetition(row *sql.Row) (*model.Competition, error) {
	var c model.Competition
	var spectator int
	if err := row.Scan(&c.ID, &c.Name, &c.Date, &c.LanesTotal, &c.Status, &spectator, &c.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan competition: %w", err)
	}
	c.SpectatorEnabled = spectator == 1
	return &c, nil
}

func scanCompetitionRow(rows *sql.Rows) (model.Competition, error) {
	var c model.Competition
	var spectator int
	if err := rows.Scan(&c.ID, &c.Name, &c.Date, &c.LanesTotal, &c.Status, &spectator, &c.CreatedAt); err != nil {
		return model.Competition{}, fmt.Errorf("scan competition: %w", err)
	}
	c.SpectatorEnabled = spectator == 1
	return c, nil
}

func (r *CompetitionRepository) GetByID(ctx context.Context, id int64) (*model.Competition, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+competitionCols+` FROM competitions WHERE id = ?`, id,
	)
	return scanCompetition(row)
}

func (r *CompetitionRepository) List(ctx context.Context, status string) (_ []model.Competition, retErr error) {
	query := `SELECT ` + competitionCols + ` FROM competitions`
	args := []any{}
	if status != "" {
		query += ` WHERE status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY id`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list competitions: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && retErr == nil {
			retErr = cerr
		}
	}()

	var out []model.Competition
	for rows.Next() {
		c, err := scanCompetitionRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CompetitionRepository) Create(ctx context.Context, name, date string, lanesTotal int, spectatorEnabled bool) (*model.Competition, error) {
	se := 0
	if spectatorEnabled {
		se = 1
	}
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO competitions (name, date, lanes_total, spectator_enabled) VALUES (?, ?, ?, ?)`,
		name, date, lanesTotal, se,
	)
	if err != nil {
		return nil, fmt.Errorf("create competition: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *CompetitionRepository) Update(ctx context.Context, id int64, name, date, status string, lanesTotal int, spectatorEnabled bool) (*model.Competition, error) {
	se := 0
	if spectatorEnabled {
		se = 1
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE competitions SET name = ?, date = ?, lanes_total = ?, status = ?, spectator_enabled = ? WHERE id = ?`,
		name, date, lanesTotal, status, se, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update competition: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, id)
}
