package repository

import "context"

type Competition struct {
	ID         int64
	Name       string
	StartDate  string
	EndDate    string
	MatchCount int
	RangeID    *int64
	RangeName  string
}

func (r *Repository) CreateCompetition(ctx context.Context, c Competition) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO competitions (name, start_date, end_date, range_id) VALUES (?, ?, ?, ?)`, c.Name, c.StartDate, c.EndDate, c.RangeID)
	if err != nil {
		return 0, mapErr(err)
	}
	return res.LastInsertId()
}

func (r *Repository) GetCompetition(ctx context.Context, id int64) (Competition, error) {
	var c Competition
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, start_date, end_date, (SELECT COUNT(*) FROM matches WHERE competition_id = competitions.id),
		        range_id, COALESCE((SELECT name FROM ranges WHERE id = competitions.range_id), '')
		 FROM competitions WHERE id = ?`, id).Scan(&c.ID, &c.Name, &c.StartDate, &c.EndDate, &c.MatchCount, &c.RangeID, &c.RangeName)
	return c, mapErr(err)
}

func (r *Repository) ListCompetitions(ctx context.Context) ([]Competition, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, start_date, end_date, (SELECT COUNT(*) FROM matches WHERE competition_id = competitions.id),
		        range_id, COALESCE((SELECT name FROM ranges WHERE id = competitions.range_id), '')
		 FROM competitions ORDER BY start_date DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []Competition{}
	for rows.Next() {
		var c Competition
		if err := rows.Scan(&c.ID, &c.Name, &c.StartDate, &c.EndDate, &c.MatchCount, &c.RangeID, &c.RangeName); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateCompetition(ctx context.Context, c Competition) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE competitions SET name = ?, start_date = ?, end_date = ?, range_id = ? WHERE id = ?`, c.Name, c.StartDate, c.EndDate, c.RangeID, c.ID)
	return affected(res, err)
}

func (r *Repository) DeleteCompetition(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM competitions WHERE id = ?`, id)
	return affected(res, err)
}

func (r *Repository) MatchDateBounds(ctx context.Context, competitionID int64) (first, last string, err error) {
	var lo, hi *string
	err = r.db.QueryRowContext(ctx,
		`SELECT MIN(date), MAX(date) FROM matches WHERE competition_id = ?`, competitionID).Scan(&lo, &hi)
	if lo != nil {
		first = *lo
	}
	if hi != nil {
		last = *hi
	}
	return first, last, mapErr(err)
}

func (r *Repository) ListCompetitionMatches(ctx context.Context, competitionID int64) ([]Match, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+matchColumns+` FROM matches WHERE competition_id = ? ORDER BY date, id`, competitionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []Match{}
	for rows.Next() {
		m, err := scanMatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	_ = rows.Close()
	return out, r.loadMatchOfficials(ctx, out)
}
