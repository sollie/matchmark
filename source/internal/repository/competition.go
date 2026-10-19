package repository

import (
	"context"
	"sort"
)

type Shooter struct {
	ID        int64
	LastName  string
	FirstName string
	Sex       string
	Country   string
	Club      string
	WAID      string
}

type Competition struct {
	ID     int64
	Name   string
	Date   string
	Course string
	Status string
}

type Entry struct {
	ID            int64
	CompetitionID int64
	ShooterID     int64
	Category      string
	Lane          int
	Withdrawn     bool
}

type StageSeed struct {
	Number        int
	ExpectedShots int
}

func (r *Repository) CreateShooter(ctx context.Context, s Shooter) (int64, error) {
	var waid any
	if s.WAID != "" {
		waid = s.WAID
	}
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO shooters (last_name, first_name, sex, country, club, waid) VALUES (?, ?, ?, ?, ?, ?)`,
		s.LastName, s.FirstName, s.Sex, s.Country, s.Club, waid)
	if err != nil {
		return 0, mapErr(err)
	}
	return res.LastInsertId()
}

func (r *Repository) CreateCompetition(ctx context.Context, c Competition) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO competitions (name, date, course) VALUES (?, ?, ?)`, c.Name, c.Date, c.Course)
	if err != nil {
		return 0, mapErr(err)
	}
	return res.LastInsertId()
}

func (r *Repository) GetCompetition(ctx context.Context, id int64) (Competition, error) {
	var c Competition
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, date, course, status FROM competitions WHERE id = ?`, id).
		Scan(&c.ID, &c.Name, &c.Date, &c.Course, &c.Status)
	return c, mapErr(err)
}

func (r *Repository) CreateEntry(ctx context.Context, e Entry, stages []StageSeed) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx,
		`INSERT INTO entries (competition_id, shooter_id, category, lane) VALUES (?, ?, ?, ?)`,
		e.CompetitionID, e.ShooterID, e.Category, e.Lane)
	if err != nil {
		return 0, mapErr(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	for _, s := range stages {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO stages (entry_id, stage_number, expected_shots) VALUES (?, ?, ?)`,
			id, s.Number, s.ExpectedShots); err != nil {
			return 0, mapErr(err)
		}
	}
	return id, tx.Commit()
}

func (r *Repository) WithdrawEntry(ctx context.Context, competitionID, entryID int64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE entries SET withdrawn = 1 WHERE id = ? AND competition_id = ?`, entryID, competitionID)
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

func (r *Repository) MannedLanes(ctx context.Context, competitionID int64) ([]int, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT lane FROM entries WHERE competition_id = ? AND withdrawn = 0 ORDER BY lane`, competitionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var lanes []int
	for rows.Next() {
		var l int
		if err := rows.Scan(&l); err != nil {
			return nil, err
		}
		lanes = append(lanes, l)
	}
	sort.Ints(lanes)
	return lanes, rows.Err()
}
