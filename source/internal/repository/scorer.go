package repository

import "context"

type Pairing struct {
	ID            int64
	CompetitionID int64
	Lane          int
	Revoked       bool
}

func (r *Repository) CreatePairing(ctx context.Context, competitionID int64, lane int, tokenHash string) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx,
		`UPDATE device_pairings SET revoked = 1 WHERE competition_id = ? AND lane = ?`, competitionID, lane); err != nil {
		return 0, mapErr(err)
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO device_pairings (token_hash, competition_id, lane) VALUES (?, ?, ?)`, tokenHash, competitionID, lane)
	if err != nil {
		return 0, mapErr(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (r *Repository) ListPairings(ctx context.Context, competitionID int64) ([]Pairing, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, competition_id, lane, revoked FROM device_pairings WHERE competition_id = ? ORDER BY lane, id`, competitionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	pairings := []Pairing{}
	for rows.Next() {
		var p Pairing
		if err := rows.Scan(&p.ID, &p.CompetitionID, &p.Lane, &p.Revoked); err != nil {
			return nil, err
		}
		pairings = append(pairings, p)
	}
	return pairings, rows.Err()
}

func (r *Repository) PairingByTokenHash(ctx context.Context, tokenHash string) (Pairing, error) {
	var p Pairing
	err := r.db.QueryRowContext(ctx,
		`SELECT id, competition_id, lane, revoked FROM device_pairings WHERE token_hash = ?`, tokenHash).
		Scan(&p.ID, &p.CompetitionID, &p.Lane, &p.Revoked)
	return p, mapErr(err)
}

func (r *Repository) RevokePairing(ctx context.Context, competitionID, pairingID int64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE device_pairings SET revoked = 1 WHERE id = ? AND competition_id = ?`, pairingID, competitionID)
	return affected(res, err)
}

func (r *Repository) RevokeCompetitionPairings(ctx context.Context, competitionID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE device_pairings SET revoked = 1 WHERE competition_id = ?`, competitionID)
	return mapErr(err)
}

func (r *Repository) MannedEntryByLane(ctx context.Context, competitionID int64, lane int) (Entry, error) {
	var e Entry
	err := r.db.QueryRowContext(ctx,
		`SELECT id, competition_id, shooter_id, category, lane, withdrawn FROM entries
		 WHERE competition_id = ? AND lane = ? AND withdrawn = 0`, competitionID, lane).
		Scan(&e.ID, &e.CompetitionID, &e.ShooterID, &e.Category, &e.Lane, &e.Withdrawn)
	return e, mapErr(err)
}

func (r *Repository) LockStage(ctx context.Context, stageID int64, scorerLane int, scorerName string, ev StageEvent) error {
	return r.transition(ctx, true, ev,
		`UPDATE stages SET status = 'awaiting_acceptance', scorer_lane = ?, scorer_name = ?, locked_at = ?,
		        accepted_by = NULL, accepted_at = NULL, disputed_at = NULL, started = 0
		 WHERE id = ? AND status = 'open'`,
		scorerLane, scorerName, lockTimestamp(), stageID)
}

func (r *Repository) LockStageAsUser(ctx context.Context, stageID int64, username string, ev StageEvent) error {
	now := lockTimestamp()
	return r.transition(ctx, true, ev,
		`UPDATE stages SET status = 'accepted', scorer_lane = NULL, scorer_name = ?, locked_at = ?,
		        accepted_by = ?, accepted_at = ?, disputed_at = NULL, started = 0
		 WHERE id = ? AND status = 'open'`,
		username, now, username, now, stageID)
}

func (r *Repository) AcceptStage(ctx context.Context, stageID int64, acceptedBy string, ev StageEvent) error {
	return r.transition(ctx, true, ev,
		`UPDATE stages SET status = 'accepted', accepted_by = ?, accepted_at = ?
		 WHERE id = ? AND status = 'awaiting_acceptance'`,
		acceptedBy, lockTimestamp(), stageID)
}

func (r *Repository) DisputeStage(ctx context.Context, stageID int64, ev StageEvent) error {
	return r.transition(ctx, true, ev,
		`UPDATE stages SET status = 'open', scorer_lane = NULL, scorer_name = NULL, locked_at = NULL,
		        accepted_by = NULL, accepted_at = NULL, disputed_at = ?
		 WHERE id = ? AND status = 'awaiting_acceptance'`,
		lockTimestamp(), stageID)
}

func (r *Repository) UnlockStage(ctx context.Context, stageID int64, ev StageEvent) error {
	return r.transition(ctx, true, ev,
		`UPDATE stages SET status = 'open', scorer_lane = NULL, scorer_name = NULL, locked_at = NULL,
		        accepted_by = NULL, accepted_at = NULL, disputed_at = NULL
		 WHERE id = ? AND status <> 'open'`, stageID)
}
