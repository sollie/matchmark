package repository

import (
	"context"
	"database/sql"
)

type RestoreStage struct {
	Number     int
	Expected   int
	Scored     bool
	Counts     RingCounts
	ScorerLane *int
	ScorerName string
	LockedAt   string
}

type RestoreEntry struct {
	ID       int64
	Category string
	Lane     int
	Shooter  Shooter
	Stages   []RestoreStage
}

type RestoreCompetition struct {
	Competition Competition
	Entries     []RestoreEntry
}

type RestoreCounts struct {
	Competitions int
	Entries      int
	Stages       int
}

func (r *Repository) RestoreCompetition(ctx context.Context, rc RestoreCompetition) (RestoreCounts, error) {
	var counts RestoreCounts
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return counts, err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO competitions (id, name, date, course, status) VALUES (?, ?, ?, ?, ?)`,
		rc.Competition.ID, rc.Competition.Name, rc.Competition.Date, rc.Competition.Course, rc.Competition.Status)
	if err != nil {
		return counts, mapErr(err)
	}
	if n, _ := res.RowsAffected(); n > 0 {
		counts.Competitions++
	}
	for _, e := range rc.Entries {
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM entries WHERE id = ?`, e.ID).Scan(&exists); err != nil {
			return counts, err
		}
		if exists == 0 {
			shooterID, err := restoreShooter(ctx, tx, e.Shooter)
			if err != nil {
				return counts, err
			}
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO entries (id, competition_id, shooter_id, category, lane) VALUES (?, ?, ?, ?, ?)`,
				e.ID, rc.Competition.ID, shooterID, e.Category, e.Lane); err != nil {
				return counts, mapErr(err)
			}
			counts.Entries++
		}
		for _, s := range e.Stages {
			if _, err := tx.ExecContext(ctx,
				`INSERT OR IGNORE INTO stages (entry_id, stage_number, expected_shots) VALUES (?, ?, ?)`,
				e.ID, s.Number, s.Expected); err != nil {
				return counts, mapErr(err)
			}
			if !s.Scored {
				continue
			}
			res, err := tx.ExecContext(ctx,
				`UPDATE stages SET x = ?, ten = ?, nine = ?, eight = ?, seven = ?, zero = ?, miss = ?,
				        locked = 1, scorer_lane = ?, scorer_name = ?, locked_at = ?
				 WHERE entry_id = ? AND stage_number = ? AND locked = 0`,
				s.Counts.X, s.Counts.Ten, s.Counts.Nine, s.Counts.Eight, s.Counts.Seven, s.Counts.Zero, s.Counts.Miss,
				s.ScorerLane, s.ScorerName, s.LockedAt, e.ID, s.Number)
			if err != nil {
				return counts, mapErr(err)
			}
			if n, _ := res.RowsAffected(); n > 0 {
				counts.Stages++
			}
		}
	}
	return counts, tx.Commit()
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func restoreShooter(ctx context.Context, tx execer, s Shooter) (int64, error) {
	var id int64
	var err error
	if s.WAID != "" {
		err = tx.QueryRowContext(ctx, `SELECT id FROM shooters WHERE waid = ?`, s.WAID).Scan(&id)
	} else {
		err = tx.QueryRowContext(ctx,
			`SELECT id FROM shooters WHERE waid IS NULL AND first_name = ? AND last_name = ? AND club = ?`,
			s.FirstName, s.LastName, s.Club).Scan(&id)
	}
	if err == nil {
		return id, nil
	}
	if mapErr(err) != ErrNotFound {
		return 0, err
	}
	var waid any
	if s.WAID != "" {
		waid = s.WAID
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO shooters (last_name, first_name, sex, country, club, waid) VALUES (?, ?, ?, ?, ?, ?)`,
		s.LastName, s.FirstName, s.Sex, s.Country, s.Club, waid)
	if err != nil {
		return 0, mapErr(err)
	}
	return res.LastInsertId()
}
