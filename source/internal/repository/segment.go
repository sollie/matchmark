package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/sollie/matchmark/internal/rulebook"
)

type SegmentStatus string

const (
	StatusOpen               SegmentStatus = "open"
	StatusAwaitingAcceptance SegmentStatus = "awaiting_acceptance"
	StatusAccepted           SegmentStatus = "accepted"
)

type SegmentEventKind string

const (
	EventLocked      SegmentEventKind = "locked"
	EventAccepted    SegmentEventKind = "accepted"
	EventDisputed    SegmentEventKind = "disputed"
	EventAcceptedAll SegmentEventKind = "accepted_all"
	EventUnlocked    SegmentEventKind = "unlocked"
)

type SegmentEvent struct {
	ID            int64
	MatchID       int64
	EntryID       int64
	SegmentNumber int
	Event         SegmentEventKind
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
	return rulebook.TenRingValue*(c.X+c.Ten) +
		rulebook.NineRingValue*c.Nine +
		rulebook.EightRingValue*c.Eight +
		rulebook.SevenRingValue*c.Seven
}

type Segment struct {
	ID            int64
	EntryID       int64
	Number        int
	ExpectedShots int
	Counts        RingCounts
	Status        SegmentStatus
	ScorerLane    *int
	ScorerName    string
	LockedAt      string
	AcceptedBy    string
	AcceptedAt    string
	DisputedAt    string
}

func (r *Repository) GetSegment(ctx context.Context, matchID, entryID int64, number int) (Segment, error) {
	var s Segment
	err := r.db.QueryRowContext(ctx,
		`SELECT s.id, s.entry_id, s.segment_number, s.expected_shots, s.x, s.ten, s.nine, s.eight, s.seven, s.zero, s.miss, s.status,
		        s.scorer_lane, COALESCE(s.scorer_name, ''), COALESCE(s.locked_at, ''),
		        COALESCE(s.accepted_by, ''), COALESCE(s.accepted_at, ''), COALESCE(s.disputed_at, '')
		 FROM segments s JOIN entries e ON e.id = s.entry_id
		 WHERE e.match_id = ? AND e.id = ? AND s.segment_number = ?`,
		matchID, entryID, number).
		Scan(&s.ID, &s.EntryID, &s.Number, &s.ExpectedShots,
			&s.Counts.X, &s.Counts.Ten, &s.Counts.Nine, &s.Counts.Eight, &s.Counts.Seven, &s.Counts.Zero, &s.Counts.Miss, &s.Status,
			&s.ScorerLane, &s.ScorerName, &s.LockedAt, &s.AcceptedBy, &s.AcceptedAt, &s.DisputedAt)
	return s, mapErr(err)
}

func (r *Repository) SetRingCounts(ctx context.Context, segmentID int64, c RingCounts, activeOnly bool) error {
	query := `UPDATE segments SET x = ?, ten = ?, nine = ?, eight = ?, seven = ?, zero = ?, miss = ?, started = 0 WHERE id = ? AND status = 'open'`
	if activeOnly {
		query += ` AND EXISTS (SELECT 1 FROM entries e JOIN matches m ON m.id = e.match_id WHERE e.id = segments.entry_id AND m.status = 'active')`
	}
	res, err := r.db.ExecContext(ctx, query, c.X, c.Ten, c.Nine, c.Eight, c.Seven, c.Zero, c.Miss, segmentID)
	if err != nil {
		return mapErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrConflict
	}
	return nil
}

func (r *Repository) OverwriteRingCounts(ctx context.Context, segment Segment, c RingCounts, username string, ev SegmentEvent) error {
	if segment.Status == StatusOpen {
		res, err := r.db.ExecContext(ctx,
			`UPDATE segments SET x = ?, ten = ?, nine = ?, eight = ?, seven = ?, zero = ?, miss = ?, started = 0 WHERE id = ? AND status = 'open'`,
			c.X, c.Ten, c.Nine, c.Eight, c.Seven, c.Zero, c.Miss, segment.ID)
		return conflictIfNone(res, err)
	}
	return r.transition(ctx, segment.Status == StatusAwaitingAcceptance, ev,
		`UPDATE segments SET x = ?, ten = ?, nine = ?, eight = ?, seven = ?, zero = ?, miss = ?,
		        status = 'accepted', scorer_lane = NULL, scorer_name = ?, accepted_by = ?, accepted_at = ?, disputed_at = NULL, started = 0
		 WHERE id = ? AND status = ?`,
		c.X, c.Ten, c.Nine, c.Eight, c.Seven, c.Zero, c.Miss, username, username, lockTimestamp(), segment.ID, segment.Status)
}

func (r *Repository) transition(ctx context.Context, record bool, ev SegmentEvent, query string, args ...any) error {
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
		if err := insertSegmentEvent(ctx, tx, ev); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func insertSegmentEvent(ctx context.Context, tx *sql.Tx, ev SegmentEvent) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO segment_events (match_id, entry_id, segment_number, event, actor_type, actor_name, scorer_name, shooter_name, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ev.MatchID, ev.EntryID, ev.SegmentNumber, ev.Event, ev.ActorType, ev.ActorName, ev.ScorerName, ev.ShooterName, lockTimestamp())
	return mapErr(err)
}

type SegmentAcceptance struct {
	SegmentID int64
	Event     SegmentEvent
}

func (r *Repository) AcceptSegments(ctx context.Context, acceptedBy string, items []SegmentAcceptance) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	now := lockTimestamp()
	for _, it := range items {
		res, err := tx.ExecContext(ctx,
			`UPDATE segments SET status = 'accepted', accepted_by = ?, accepted_at = ?
			 WHERE id = ? AND status = 'awaiting_acceptance'`,
			acceptedBy, now, it.SegmentID)
		if err := conflictIfNone(res, err); err != nil {
			return err
		}
		if err := insertSegmentEvent(ctx, tx, it.Event); err != nil {
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

func (r *Repository) ListSegmentEvents(ctx context.Context, matchID int64) ([]SegmentEvent, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, match_id, entry_id, segment_number, event, actor_type, actor_name, scorer_name, shooter_name, created_at
		 FROM segment_events WHERE match_id = ? ORDER BY id`, matchID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var events []SegmentEvent
	for rows.Next() {
		var e SegmentEvent
		if err := rows.Scan(&e.ID, &e.MatchID, &e.EntryID, &e.SegmentNumber, &e.Event, &e.ActorType, &e.ActorName, &e.ScorerName, &e.ShooterName, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

type LockedSegmentRef struct {
	MatchID int64
	EntryID int64
	Number  int
}

type LockedSegmentFilter struct {
	MatchID   int64
	EntryID   int64
	ShooterID int64
}

func (r *Repository) ListLockedSegments(ctx context.Context, f LockedSegmentFilter) ([]LockedSegmentRef, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT e.match_id, s.entry_id, s.segment_number FROM segments s
		 JOIN entries e ON e.id = s.entry_id
		 WHERE s.status <> 'open'
		   AND (? = 0 OR e.match_id = ?)
		   AND (? = 0 OR s.entry_id = ?)
		   AND (? = 0 OR e.shooter_id = ?)
		 ORDER BY e.match_id, s.entry_id, s.segment_number`,
		f.MatchID, f.MatchID, f.EntryID, f.EntryID, f.ShooterID, f.ShooterID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var refs []LockedSegmentRef
	for rows.Next() {
		var ref LockedSegmentRef
		if err := rows.Scan(&ref.MatchID, &ref.EntryID, &ref.Number); err != nil {
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
