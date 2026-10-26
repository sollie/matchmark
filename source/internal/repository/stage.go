package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/sollie/matchmark/internal/rulebook"
)

type StageStatus string

const (
	StatusOpen               StageStatus = "open"
	StatusAwaitingAcceptance StageStatus = "awaiting_acceptance"
	StatusAccepted           StageStatus = "accepted"
)

type StageEventKind string

const (
	EventLocked      StageEventKind = "locked"
	EventAccepted    StageEventKind = "accepted"
	EventDisputed    StageEventKind = "disputed"
	EventAcceptedAll StageEventKind = "accepted_all"
	EventUnlocked    StageEventKind = "unlocked"
)

type StageEvent struct {
	ID            int64
	CompetitionID int64
	EntryID       int64
	StageNumber   int
	Event         StageEventKind
	ActorType     string
	ActorName     string
	ScorerName    string
	ShooterName   string
	CreatedAt     string
}

type RingCounts struct {
	X     int
	Ten   int
	Nine  int
	Eight int
	Seven int
	Zero  int
	Miss  int
}

func (c RingCounts) Score() int {
	return rulebook.TenRingPoints*(c.X+c.Ten) +
		rulebook.NineRingPoints*c.Nine +
		rulebook.EightRingPoints*c.Eight +
		rulebook.SevenRingPoints*c.Seven
}

type Stage struct {
	ID            int64
	EntryID       int64
	Number        int
	ExpectedShots int
	Counts        RingCounts
	Status        StageStatus
	ScorerLane    *int
	ScorerName    string
	LockedAt      string
	AcceptedBy    string
	AcceptedAt    string
	DisputedAt    string
}

func (r *Repository) GetStage(ctx context.Context, competitionID, entryID int64, number int) (Stage, error) {
	var s Stage
	err := r.db.QueryRowContext(ctx,
		`SELECT s.id, s.entry_id, s.stage_number, s.expected_shots, s.x, s.ten, s.nine, s.eight, s.seven, s.zero, s.miss, s.status,
		        s.scorer_lane, COALESCE(s.scorer_name, ''), COALESCE(s.locked_at, ''),
		        COALESCE(s.accepted_by, ''), COALESCE(s.accepted_at, ''), COALESCE(s.disputed_at, '')
		 FROM stages s JOIN entries e ON e.id = s.entry_id
		 WHERE e.competition_id = ? AND e.id = ? AND s.stage_number = ?`,
		competitionID, entryID, number).
		Scan(&s.ID, &s.EntryID, &s.Number, &s.ExpectedShots,
			&s.Counts.X, &s.Counts.Ten, &s.Counts.Nine, &s.Counts.Eight, &s.Counts.Seven, &s.Counts.Zero, &s.Counts.Miss, &s.Status,
			&s.ScorerLane, &s.ScorerName, &s.LockedAt, &s.AcceptedBy, &s.AcceptedAt, &s.DisputedAt)
	return s, mapErr(err)
}

func (r *Repository) SetRingCounts(ctx context.Context, stageID int64, c RingCounts) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE stages SET x = ?, ten = ?, nine = ?, eight = ?, seven = ?, zero = ?, miss = ?, started = 0 WHERE id = ? AND status = 'open'`,
		c.X, c.Ten, c.Nine, c.Eight, c.Seven, c.Zero, c.Miss, stageID)
	if err != nil {
		return mapErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrConflict
	}
	return nil
}

func (r *Repository) OverwriteRingCounts(ctx context.Context, stage Stage, c RingCounts, username string, ev StageEvent) error {
	if stage.Status == StatusOpen {
		res, err := r.db.ExecContext(ctx,
			`UPDATE stages SET x = ?, ten = ?, nine = ?, eight = ?, seven = ?, zero = ?, miss = ?, started = 0 WHERE id = ? AND status = 'open'`,
			c.X, c.Ten, c.Nine, c.Eight, c.Seven, c.Zero, c.Miss, stage.ID)
		return conflictIfNone(res, err)
	}
	return r.transition(ctx, stage.Status == StatusAwaitingAcceptance, ev,
		`UPDATE stages SET x = ?, ten = ?, nine = ?, eight = ?, seven = ?, zero = ?, miss = ?,
		        status = 'accepted', scorer_lane = NULL, scorer_name = ?, accepted_by = ?, accepted_at = ?, disputed_at = NULL, started = 0
		 WHERE id = ? AND status = ?`,
		c.X, c.Ten, c.Nine, c.Eight, c.Seven, c.Zero, c.Miss, username, username, lockTimestamp(), stage.ID, stage.Status)
}

func (r *Repository) transition(ctx context.Context, record bool, ev StageEvent, query string, args ...any) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, query, args...)
	if err := conflictIfNone(res, err); err != nil {
		return err
	}
	if record {
		if err := insertStageEvent(ctx, tx, ev); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func insertStageEvent(ctx context.Context, tx *sql.Tx, ev StageEvent) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO stage_events (competition_id, entry_id, stage_number, event, actor_type, actor_name, scorer_name, shooter_name, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ev.CompetitionID, ev.EntryID, ev.StageNumber, ev.Event, ev.ActorType, ev.ActorName, ev.ScorerName, ev.ShooterName, lockTimestamp())
	return mapErr(err)
}

type StageAcceptance struct {
	StageID int64
	Event   StageEvent
}

func (r *Repository) AcceptStages(ctx context.Context, acceptedBy string, items []StageAcceptance) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	now := lockTimestamp()
	for _, it := range items {
		res, err := tx.ExecContext(ctx,
			`UPDATE stages SET status = 'accepted', accepted_by = ?, accepted_at = ?
			 WHERE id = ? AND status = 'awaiting_acceptance'`,
			acceptedBy, now, it.StageID)
		if err := conflictIfNone(res, err); err != nil {
			return err
		}
		if err := insertStageEvent(ctx, tx, it.Event); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func conflictIfNone(res sql.Result, err error) error {
	if err != nil {
		return mapErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrConflict
	}
	return nil
}

func (r *Repository) ListStageEvents(ctx context.Context, competitionID int64) ([]StageEvent, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, competition_id, entry_id, stage_number, event, actor_type, actor_name, scorer_name, shooter_name, created_at
		 FROM stage_events WHERE competition_id = ? ORDER BY id`, competitionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var events []StageEvent
	for rows.Next() {
		var e StageEvent
		if err := rows.Scan(&e.ID, &e.CompetitionID, &e.EntryID, &e.StageNumber, &e.Event, &e.ActorType, &e.ActorName, &e.ScorerName, &e.ShooterName, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
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
		 WHERE s.status <> 'open'
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

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
