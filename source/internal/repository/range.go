package repository

import "context"

type Range struct {
	ID       int64
	Name     string
	Archived bool
}

func (r *Repository) CreateRange(ctx context.Context, name string) (int64, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO ranges (name) VALUES (?)`, name)
	if err != nil {
		return 0, mapErr(err)
	}
	return res.LastInsertId()
}

func (r *Repository) GetRange(ctx context.Context, id int64) (Range, error) {
	var g Range
	err := r.db.QueryRowContext(ctx, `SELECT id, name, archived FROM ranges WHERE id = ?`, id).Scan(&g.ID, &g.Name, &g.Archived)
	return g, mapErr(err)
}

func (r *Repository) ListRanges(ctx context.Context) ([]Range, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, archived FROM ranges ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []Range{}
	for rows.Next() {
		var g Range
		if err := rows.Scan(&g.ID, &g.Name, &g.Archived); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *Repository) RenameRange(ctx context.Context, id int64, name string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE ranges SET name = ? WHERE id = ?`, name, id)
	return affected(res, err)
}

func (r *Repository) SetRangeArchived(ctx context.Context, id int64, archived bool) error {
	res, err := r.db.ExecContext(ctx, `UPDATE ranges SET archived = ? WHERE id = ?`, archived, id)
	return affected(res, err)
}

func (r *Repository) DeleteRange(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM ranges WHERE id = ?`, id)
	return affected(res, err)
}

func (r *Repository) RangeUsage(ctx context.Context, id int64) (competitions int, isDefault bool, err error) {
	err = r.db.QueryRowContext(ctx,
		`SELECT (SELECT COUNT(*) FROM competitions WHERE range_id = ?),
		        EXISTS (SELECT 1 FROM club_settings WHERE default_range_id = ?)`, id, id).Scan(&competitions, &isDefault)
	return competitions, isDefault, mapErr(err)
}

func (r *Repository) SetClubDefaultRange(ctx context.Context, id *int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE club_settings SET default_range_id = ?, version = version + 1 WHERE id = 1`, id)
	return mapErr(err)
}
