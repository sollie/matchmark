package repository

import (
	"context"
	"time"
)

type RingCounts struct {
	X     int
	Ten   int
	Nine  int
	Eight int
	Seven int
	Zero  int
	Miss  int
}

type Stage struct {
	ID            int64
	EntryID       int64
	Number        int
	ExpectedShots int
	Counts        RingCounts
	Locked        bool
	ScorerLane    *int
	ScorerName    string
	LockedAt      string
}

func (r *Repository) GetStage(ctx context.Context, competitionID, entryID int64, number int) (Stage, error) {
	var s Stage
	var locked int
	err := r.db.QueryRowContext(ctx,
		`SELECT s.id, s.entry_id, s.stage_number, s.expected_shots, s.x, s.ten, s.nine, s.eight, s.seven, s.zero, s.miss, s.locked,
		        s.scorer_lane, COALESCE(s.scorer_name, ''), COALESCE(s.locked_at, '')
		 FROM stages s JOIN entries e ON e.id = s.entry_id
		 WHERE e.competition_id = ? AND e.id = ? AND s.stage_number = ?`,
		competitionID, entryID, number).
		Scan(&s.ID, &s.EntryID, &s.Number, &s.ExpectedShots,
			&s.Counts.X, &s.Counts.Ten, &s.Counts.Nine, &s.Counts.Eight, &s.Counts.Seven, &s.Counts.Zero, &s.Counts.Miss, &locked,
			&s.ScorerLane, &s.ScorerName, &s.LockedAt)
	s.Locked = locked == 1
	return s, mapErr(err)
}

func (r *Repository) SetRingCounts(ctx context.Context, stageID int64, c RingCounts) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE stages SET x = ?, ten = ?, nine = ?, eight = ?, seven = ?, zero = ?, miss = ? WHERE id = ? AND locked = 0`,
		c.X, c.Ten, c.Nine, c.Eight, c.Seven, c.Zero, c.Miss, stageID)
	if err != nil {
		return mapErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrConflict
	}
	return nil
}

func (r *Repository) OverwriteRingCounts(ctx context.Context, stageID int64, c RingCounts, username string) error {
	if username == "" {
		res, err := r.db.ExecContext(ctx,
			`UPDATE stages SET x = ?, ten = ?, nine = ?, eight = ?, seven = ?, zero = ?, miss = ? WHERE id = ?`,
			c.X, c.Ten, c.Nine, c.Eight, c.Seven, c.Zero, c.Miss, stageID)
		return affected(res, err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE stages SET x = ?, ten = ?, nine = ?, eight = ?, seven = ?, zero = ?, miss = ?, scorer_lane = NULL, scorer_name = ? WHERE id = ?`,
		c.X, c.Ten, c.Nine, c.Eight, c.Seven, c.Zero, c.Miss, username, stageID)
	return affected(res, err)
}

type LockedStageRef struct {
	CompetitionID int64
	EntryID       int64
	Number        int
}

type LockedStageFilter struct {
	CompetitionID int64
	EntryID       int64
	ShooterID     int64
}

func (r *Repository) ListLockedStages(ctx context.Context, f LockedStageFilter) ([]LockedStageRef, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT e.competition_id, s.entry_id, s.stage_number FROM stages s
		 JOIN entries e ON e.id = s.entry_id
		 WHERE s.locked = 1
		   AND (? = 0 OR e.competition_id = ?)
		   AND (? = 0 OR s.entry_id = ?)
		   AND (? = 0 OR e.shooter_id = ?)
		 ORDER BY e.competition_id, s.entry_id, s.stage_number`,
		f.CompetitionID, f.CompetitionID, f.EntryID, f.EntryID, f.ShooterID, f.ShooterID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var refs []LockedStageRef
	for rows.Next() {
		var ref LockedStageRef
		if err := rows.Scan(&ref.CompetitionID, &ref.EntryID, &ref.Number); err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}

func lockTimestamp() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}
