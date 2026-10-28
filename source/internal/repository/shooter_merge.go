package repository

import (
	"context"
	"database/sql"
	"errors"
)

var (
	ErrMergeBlocked  = errors.New("merge blocked")
	ErrMergeArchived = errors.New("merge target archived")
)

type MergeOutcome struct {
	WAID  string
	Moved int64
}

func (r *Repository) MergeShooters(ctx context.Context, sourceID, targetID int64) (MergeOutcome, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return MergeOutcome{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var sourceWAID, targetWAID sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT waid FROM shooters WHERE id = ?`, sourceID).Scan(&sourceWAID); err != nil {
		return MergeOutcome{}, mapErr(err)
	}
	var targetArchived bool
	var targetFirst, targetLast string
	if err := tx.QueryRowContext(ctx, `SELECT waid, archived, first_name, last_name FROM shooters WHERE id = ?`, targetID).
		Scan(&targetWAID, &targetArchived, &targetFirst, &targetLast); err != nil {
		return MergeOutcome{}, mapErr(err)
	}
	if targetArchived {
		return MergeOutcome{}, ErrMergeArchived
	}
	if sourceWAID.String != "" && targetWAID.String != "" && sourceWAID.String != targetWAID.String {
		return MergeOutcome{}, ErrMergeBlocked
	}
	var shared int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM entries a JOIN entries b ON a.match_id = b.match_id WHERE a.shooter_id = ? AND b.shooter_id = ?`,
		sourceID, targetID).Scan(&shared); err != nil {
		return MergeOutcome{}, err
	}
	if shared > 0 {
		return MergeOutcome{}, ErrMergeBlocked
	}
	outcome := MergeOutcome{WAID: targetWAID.String}
	if outcome.WAID == "" {
		outcome.WAID = sourceWAID.String
	}
	var waid any
	if outcome.WAID != "" {
		waid = outcome.WAID
	}
	if _, err := tx.ExecContext(ctx, `UPDATE shooters SET waid = NULL WHERE id = ?`, sourceID); err != nil {
		return MergeOutcome{}, mapErr(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE shooters SET waid = ? WHERE id = ?`, waid, targetID); err != nil {
		return MergeOutcome{}, mapErr(err)
	}
	moved, err := tx.ExecContext(ctx, `UPDATE entries SET shooter_id = ? WHERE shooter_id = ?`, targetID, sourceID)
	if err != nil {
		return MergeOutcome{}, mapErr(err)
	}
	if outcome.Moved, err = moved.RowsAffected(); err != nil {
		return MergeOutcome{}, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE stage_events SET shooter_name = ? WHERE entry_id IN (SELECT id FROM entries WHERE shooter_id = ?)`,
		targetFirst+" "+targetLast, targetID); err != nil {
		return MergeOutcome{}, err
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM shooters WHERE id = ?`, sourceID)
	if err := affected(res, err); err != nil {
		return MergeOutcome{}, err
	}
	return outcome, tx.Commit()
}
