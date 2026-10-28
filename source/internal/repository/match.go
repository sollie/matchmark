package repository

import (
	"context"
	"database/sql"
)

type Shooter struct {
	ID        int64
	LastName  string
	FirstName string
	Country   string
	Club      string
	WAID      string
	Archived  bool
}

type Match struct {
	ID            int64
	CompetitionID int64
	Name          string
	Date          string
	Course        string
	Status        string
	CurrentStage  int
}

type DateRange struct {
	Start   string
	End     string
	Matches int
}

type Entry struct {
	ID        int64
	MatchID   int64
	ShooterID int64
	Category  string
	Lane      int
	Withdrawn bool
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
		`INSERT INTO shooters (last_name, first_name, country, club, waid) VALUES (?, ?, ?, ?, ?)`,
		s.LastName, s.FirstName, s.Country, s.Club, waid)
	if err != nil {
		return 0, mapErr(err)
	}
	return res.LastInsertId()
}

func (r *Repository) CreateMatch(ctx context.Context, c Match) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	containerID, err := ensureContainer(ctx, tx, c.CompetitionID, c.Name, c.Date)
	if err != nil {
		return 0, err
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO matches (competition_id, name, date, course) VALUES (?, ?, ?, ?)`, containerID, c.Name, c.Date, c.Course)
	if err != nil {
		return 0, mapErr(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (r *Repository) CreateMatchInNewCompetition(ctx context.Context, comp Competition, c Match) (int64, int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx,
		`INSERT INTO competitions (name, start_date, end_date) VALUES (?, ?, ?)`, comp.Name, comp.StartDate, comp.EndDate)
	if err != nil {
		return 0, 0, mapErr(err)
	}
	compID, err := res.LastInsertId()
	if err != nil {
		return 0, 0, err
	}
	res, err = tx.ExecContext(ctx,
		`INSERT INTO matches (competition_id, name, date, course) VALUES (?, ?, ?, ?)`, compID, c.Name, c.Date, c.Course)
	if err != nil {
		return 0, 0, mapErr(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, 0, err
	}
	return compID, id, tx.Commit()
}

func ensureContainer(ctx context.Context, tx *sql.Tx, id int64, name, date string) (int64, error) {
	if id != 0 {
		return id, nil
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO competitions (name, start_date, end_date) VALUES (?, ?, ?)`, name, date, date)
	if err != nil {
		return 0, mapErr(err)
	}
	return res.LastInsertId()
}

func (r *Repository) CompetitionName(ctx context.Context, id int64) (string, error) {
	var name string
	err := r.db.QueryRowContext(ctx, `SELECT name FROM competitions WHERE id = ?`, id).Scan(&name)
	return name, mapErr(err)
}

func (r *Repository) CompetitionDateRange(ctx context.Context, id int64) (DateRange, error) {
	var d DateRange
	err := r.db.QueryRowContext(ctx,
		`SELECT start_date, end_date, (SELECT COUNT(*) FROM matches WHERE competition_id = competitions.id)
		 FROM competitions WHERE id = ?`, id).Scan(&d.Start, &d.End, &d.Matches)
	return d, mapErr(err)
}

func (r *Repository) GetMatch(ctx context.Context, id int64) (Match, error) {
	var c Match
	err := r.db.QueryRowContext(ctx,
		`SELECT id, competition_id, name, date, course, status, current_stage FROM matches WHERE id = ?`, id).
		Scan(&c.ID, &c.CompetitionID, &c.Name, &c.Date, &c.Course, &c.Status, &c.CurrentStage)
	return c, mapErr(err)
}

func (r *Repository) CreateEntry(ctx context.Context, e Entry, stages []StageSeed) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx,
		`INSERT INTO entries (match_id, shooter_id, category, lane) VALUES (?, ?, ?, ?)`,
		e.MatchID, e.ShooterID, e.Category, e.Lane)
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

func (r *Repository) WithdrawEntry(ctx context.Context, matchID, entryID int64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE entries SET withdrawn = 1 WHERE id = ? AND match_id = ?`, entryID, matchID)
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
