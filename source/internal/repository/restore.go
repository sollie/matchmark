package repository

import (
	"context"
	"database/sql"
)

type RestoreStage struct {
	Number     int
	Expected   int
	Scored     bool
	Status     StageStatus
	Counts     RingCounts
	ScorerLane *int
	ScorerName string
	LockedAt   string
	AcceptedBy string
	AcceptedAt string
	DisputedAt string
}

type RestoreEntry struct {
	ID       int64
	Category string
	Lane     int
	Shooter  Shooter
	Stages   []RestoreStage
}

type RestoreMatch struct {
	CompetitionID   int64
	CompetitionName string
	Match           Match
	Entries         []RestoreEntry
}

type RestoreCounts struct {
	Matches int
	Entries int
	Stages  int
}

func (r *Repository) CountMatches(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM matches`).Scan(&n)
	return n, err
}

func (r *Repository) CountShooters(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM shooters`).Scan(&n)
	return n, err
}

func (r *Repository) EntryExists(ctx context.Context, id int64) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM entries WHERE id = ?`, id).Scan(&n)
	return n > 0, err
}

func (r *Repository) RestoreMatches(ctx context.Context, rcs []RestoreMatch) (RestoreCounts, error) {
	var counts RestoreCounts
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return RestoreCounts{}, err
	}
	defer func() { _ = tx.Rollback() }()
	for _, rc := range rcs {
		if err := restoreMatch(ctx, tx, rc, &counts); err != nil {
			return RestoreCounts{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return RestoreCounts{}, err
	}
	return counts, nil
}

func restoreMatch(ctx context.Context, tx *sql.Tx, rc RestoreMatch, counts *RestoreCounts) error {
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM matches WHERE id = ?`, rc.Match.ID).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		containerID, err := restoreCompetition(ctx, tx, rc)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO matches (id, competition_id, name, date, course, status, current_stage, activated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, CASE WHEN ? = 'draft' THEN NULL ELSE ? END)`,
			rc.Match.ID, containerID, rc.Match.Name, rc.Match.Date, rc.Match.Course, rc.Match.Status, rc.Match.CurrentStage, rc.Match.Status, rc.Match.Date); err != nil {
			return mapErr(err)
		}
		counts.Matches++
	}
	for _, e := range rc.Entries {
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM entries WHERE id = ?`, e.ID).Scan(&exists); err != nil {
			return err
		}
		if exists == 0 {
			shooterID, err := restoreShooter(ctx, tx, e.Shooter)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO entries (id, match_id, shooter_id, category, lane) VALUES (?, ?, ?, ?, ?)`,
				e.ID, rc.Match.ID, shooterID, e.Category, e.Lane); err != nil {
				return mapErr(err)
			}
			counts.Entries++
		}
		for _, s := range e.Stages {
			if _, err := tx.ExecContext(ctx,
				`INSERT OR IGNORE INTO stages (entry_id, stage_number, expected_shots) VALUES (?, ?, ?)`,
				e.ID, s.Number, s.Expected); err != nil {
				return mapErr(err)
			}
			if !s.Scored {
				continue
			}
			res, err := tx.ExecContext(ctx,
				`UPDATE stages SET x = ?, ten = ?, nine = ?, eight = ?, seven = ?, zero = ?, miss = ?,
				        status = ?, scorer_lane = ?, scorer_name = ?, locked_at = ?, accepted_by = ?, accepted_at = ?, disputed_at = ?
				 WHERE entry_id = ? AND stage_number = ? AND status = 'open'`,
				s.Counts.X, s.Counts.Ten, s.Counts.Nine, s.Counts.Eight, s.Counts.Seven, s.Counts.Zero, s.Counts.Miss,
				s.Status, s.ScorerLane, nullIfEmpty(s.ScorerName), nullIfEmpty(s.LockedAt), nullIfEmpty(s.AcceptedBy),
				nullIfEmpty(s.AcceptedAt), nullIfEmpty(s.DisputedAt), e.ID, s.Number)
			if err != nil {
				return mapErr(err)
			}
			if n, _ := res.RowsAffected(); n > 0 {
				counts.Stages++
			}
		}
	}
	return nil
}

func restoreCompetition(ctx context.Context, tx *sql.Tx, rc RestoreMatch) (int64, error) {
	if rc.CompetitionID == 0 {
		return ensureContainer(ctx, tx, 0, rc.Match.Name, rc.Match.Date)
	}
	res, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO competitions (id, name, start_date, end_date) VALUES (?, ?, ?, ?)`,
		rc.CompetitionID, rc.CompetitionName, rc.Match.Date, rc.Match.Date)
	if err != nil {
		return 0, mapErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := tx.ExecContext(ctx,
			`UPDATE competitions SET start_date = MIN(start_date, ?), end_date = MAX(end_date, ?) WHERE id = ?`,
			rc.Match.Date, rc.Match.Date, rc.CompetitionID); err != nil {
			return 0, mapErr(err)
		}
	}
	return rc.CompetitionID, nil
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
		`INSERT INTO shooters (last_name, first_name, country, club, waid) VALUES (?, ?, ?, ?, ?)`,
		s.LastName, s.FirstName, s.Country, s.Club, waid)
	if err != nil {
		return 0, mapErr(err)
	}
	return res.LastInsertId()
}
