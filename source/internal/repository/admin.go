package repository

import (
	"context"
	"database/sql"
	"strings"
)

type StageInfo struct {
	Number        int
	ExpectedShots int
	Locked        bool
	ScorerLane    *int
	ScorerName    *string
}

type EntryWithStages struct {
	Entry
	Stages []StageInfo
}

func (r *Repository) ListShooters(ctx context.Context) ([]Shooter, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, last_name, first_name, sex, country, club, COALESCE(waid, '') FROM shooters ORDER BY last_name, first_name, id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	shooters := []Shooter{}
	for rows.Next() {
		var s Shooter
		if err := rows.Scan(&s.ID, &s.LastName, &s.FirstName, &s.Sex, &s.Country, &s.Club, &s.WAID); err != nil {
			return nil, err
		}
		shooters = append(shooters, s)
	}
	return shooters, rows.Err()
}

func (r *Repository) GetShooter(ctx context.Context, id int64) (Shooter, error) {
	var s Shooter
	err := r.db.QueryRowContext(ctx,
		`SELECT id, last_name, first_name, sex, country, club, COALESCE(waid, '') FROM shooters WHERE id = ?`, id).
		Scan(&s.ID, &s.LastName, &s.FirstName, &s.Sex, &s.Country, &s.Club, &s.WAID)
	return s, mapErr(err)
}

func (r *Repository) UpdateShooter(ctx context.Context, s Shooter) error {
	var waid any
	if s.WAID != "" {
		waid = s.WAID
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE shooters SET last_name = ?, first_name = ?, sex = ?, country = ?, club = ?, waid = ? WHERE id = ?`,
		s.LastName, s.FirstName, s.Sex, s.Country, s.Club, waid, s.ID)
	return affected(res, err)
}

func (r *Repository) DeleteShooter(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM shooters WHERE id = ?`, id)
	if err != nil && strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
		return ErrConflict
	}
	return affected(res, err)
}

func (r *Repository) ListCompetitions(ctx context.Context) ([]Competition, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, date, course, status FROM competitions ORDER BY date DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	comps := []Competition{}
	for rows.Next() {
		var c Competition
		if err := rows.Scan(&c.ID, &c.Name, &c.Date, &c.Course, &c.Status); err != nil {
			return nil, err
		}
		comps = append(comps, c)
	}
	return comps, rows.Err()
}

func (r *Repository) UpdateCompetition(ctx context.Context, c Competition) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE competitions SET name = ?, date = ?, course = ? WHERE id = ?`, c.Name, c.Date, c.Course, c.ID)
	return affected(res, err)
}

func (r *Repository) SetCompetitionStatus(ctx context.Context, id int64, status string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE competitions SET status = ? WHERE id = ?`, status, id)
	return affected(res, err)
}

func (r *Repository) DeleteCompetition(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM competitions WHERE id = ?`, id)
	return affected(res, err)
}

func (r *Repository) CountEntries(ctx context.Context, competitionID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM entries WHERE competition_id = ?`, competitionID).Scan(&n)
	return n, err
}

func (r *Repository) GetEntry(ctx context.Context, competitionID, entryID int64) (Entry, error) {
	var e Entry
	err := r.db.QueryRowContext(ctx,
		`SELECT id, competition_id, shooter_id, category, lane, withdrawn FROM entries WHERE id = ? AND competition_id = ?`,
		entryID, competitionID).Scan(&e.ID, &e.CompetitionID, &e.ShooterID, &e.Category, &e.Lane, &e.Withdrawn)
	return e, mapErr(err)
}

func (r *Repository) UpdateEntry(ctx context.Context, competitionID, entryID int64, category string, lane int) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE entries SET category = ?, lane = ? WHERE id = ? AND competition_id = ?`,
		category, lane, entryID, competitionID)
	return affected(res, err)
}

func (r *Repository) DeleteEntry(ctx context.Context, competitionID, entryID int64) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM entries WHERE id = ? AND competition_id = ?`, entryID, competitionID)
	return affected(res, err)
}

func (r *Repository) ListEntries(ctx context.Context, competitionID int64) ([]EntryWithStages, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, competition_id, shooter_id, category, lane, withdrawn FROM entries WHERE competition_id = ? ORDER BY lane, id`,
		competitionID)
	if err != nil {
		return nil, err
	}
	entries := []EntryWithStages{}
	index := map[int64]int{}
	for rows.Next() {
		var e EntryWithStages
		if err := rows.Scan(&e.ID, &e.CompetitionID, &e.ShooterID, &e.Category, &e.Lane, &e.Withdrawn); err != nil {
			_ = rows.Close()
			return nil, err
		}
		e.Stages = []StageInfo{}
		index[e.ID] = len(entries)
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()

	srows, err := r.db.QueryContext(ctx,
		`SELECT s.entry_id, s.stage_number, s.expected_shots, s.locked, s.scorer_lane, s.scorer_name FROM stages s
		 JOIN entries e ON e.id = s.entry_id WHERE e.competition_id = ? ORDER BY s.entry_id, s.stage_number`,
		competitionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = srows.Close() }()
	for srows.Next() {
		var entryID int64
		var si StageInfo
		if err := srows.Scan(&entryID, &si.Number, &si.ExpectedShots, &si.Locked, &si.ScorerLane, &si.ScorerName); err != nil {
			return nil, err
		}
		i := index[entryID]
		entries[i].Stages = append(entries[i].Stages, si)
	}
	return entries, srows.Err()
}

func (r *Repository) StageInProgress(ctx context.Context, competitionID int64) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM (
		   SELECT s.stage_number
		   FROM stages s JOIN entries e ON e.id = s.entry_id
		   WHERE e.competition_id = ? AND e.withdrawn = 0
		   GROUP BY s.stage_number
		   HAVING SUM(s.locked = 1 OR s.x + s.ten + s.nine + s.eight + s.seven + s.zero + s.miss > 0) BETWEEN 1 AND COUNT(*) - 1
		 )`, competitionID).Scan(&n)
	return n > 0, err
}

func affected(res sql.Result, err error) error {
	if err != nil {
		return mapErr(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
