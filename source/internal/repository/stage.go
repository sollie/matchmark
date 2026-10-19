package repository

import "context"

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
}

func (r *Repository) GetStage(ctx context.Context, competitionID, entryID int64, number int) (Stage, error) {
	var s Stage
	var locked int
	err := r.db.QueryRowContext(ctx,
		`SELECT s.id, s.entry_id, s.stage_number, s.expected_shots, s.x, s.ten, s.nine, s.eight, s.seven, s.zero, s.miss, s.locked
		 FROM stages s JOIN entries e ON e.id = s.entry_id
		 WHERE e.competition_id = ? AND e.id = ? AND s.stage_number = ?`,
		competitionID, entryID, number).
		Scan(&s.ID, &s.EntryID, &s.Number, &s.ExpectedShots,
			&s.Counts.X, &s.Counts.Ten, &s.Counts.Nine, &s.Counts.Eight, &s.Counts.Seven, &s.Counts.Zero, &s.Counts.Miss, &locked)
	s.Locked = locked == 1
	return s, mapErr(err)
}

func (r *Repository) SetRingCounts(ctx context.Context, stageID int64, c RingCounts) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE stages SET x = ?, ten = ?, nine = ?, eight = ?, seven = ?, zero = ?, miss = ? WHERE id = ?`,
		c.X, c.Ten, c.Nine, c.Eight, c.Seven, c.Zero, c.Miss, stageID)
	return mapErr(err)
}
