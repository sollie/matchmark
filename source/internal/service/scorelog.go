package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
)

const (
	scoreLogSchemaVersion = 4
	emittedAtLayout       = "2006-01-02T15:04:05.000Z"
)

const eventSegmentStatus = "segment_status"

type ScoreLog struct {
	repo *repository.Repository
	log  *slog.Logger
	mu   sync.Mutex
	seq  int64
}

func NewScoreLog(repo *repository.Repository, log *slog.Logger) *ScoreLog {
	return &ScoreLog{repo: repo, log: log}
}

func WithScoreLog(l *slog.Logger) Option {
	return func(s *Service) { s.scoreLogger = l }
}

func (l *ScoreLog) EmitSegment(ctx context.Context, matchID, entryID int64, segmentNumber int) {
	if l == nil || l.log == nil {
		return
	}
	ctx = context.WithoutCancel(ctx)
	l.mu.Lock()
	defer l.mu.Unlock()
	segment, err := l.repo.GetSegment(ctx, matchID, entryID, segmentNumber)
	if err != nil {
		l.log.ErrorContext(ctx, "score log context failed", "event", eventSegmentStatus, "error", err)
		return
	}
	attrs, err := l.segmentEventAttrs(ctx, matchID, entryID, segment)
	if err != nil {
		l.log.ErrorContext(ctx, "score log context failed", "event", eventSegmentStatus, "error", err)
		return
	}
	l.log.LogAttrs(ctx, slog.LevelInfo, eventSegmentStatus, attrs...)
}

func (l *ScoreLog) segmentEventAttrs(ctx context.Context, matchID, entryID int64, segment repository.Segment) ([]slog.Attr, error) {
	comp, err := l.repo.GetMatch(ctx, matchID)
	if err != nil {
		return nil, err
	}
	course, err := rulebook.CourseByKey(comp.Course)
	if err != nil {
		return nil, err
	}
	competitionName, err := l.repo.CompetitionName(ctx, comp.CompetitionID)
	if err != nil {
		return nil, err
	}
	entry, err := l.repo.GetEntry(ctx, matchID, entryID)
	if err != nil {
		return nil, err
	}
	shooter, err := l.repo.GetShooter(ctx, entry.ShooterID)
	if err != nil {
		return nil, err
	}
	l.seq++
	c := segment.Counts
	var lockedBy any
	if segment.Status != repository.StatusOpen {
		lockedBy = segment.ScorerName
		if segment.ScorerLane != nil {
			lockedBy = *segment.ScorerLane
		}
	}
	return []slog.Attr{
		slog.Int("schema_version", scoreLogSchemaVersion),
		slog.String("event", eventSegmentStatus),
		slog.String("emitted_at", time.Now().UTC().Format(emittedAtLayout)),
		slog.Int64("seq", l.seq),
		slog.Group("competition",
			slog.Int64("id", comp.CompetitionID),
			slog.String("name", competitionName),
		),
		slog.Group("match",
			slog.Int64("id", comp.ID),
			slog.String("name", comp.Name),
			slog.String("date", comp.Date),
			slog.Group("course_of_fire",
				slog.String("key", comp.Course),
				slog.String("discipline", course.Discipline),
				slog.Int("total_shots", course.TotalShots),
			),
		),
		slog.Group("entry",
			slog.Int64("id", entry.ID),
			slog.Int("lane", entry.Lane),
			slog.String("category", entry.Category),
			slog.Group("shooter",
				slog.String("waid", shooter.WAID),
				slog.String("first_name", shooter.FirstName),
				slog.String("last_name", shooter.LastName),
				slog.String("club", shooter.Club),
			),
		),
		slog.Group("segment",
			slog.Int("number", segment.Number),
			slog.Int("expected_shots", segment.ExpectedShots),
		),
		slog.Any("ring_counts", []int{c.X, c.Ten, c.Nine, c.Eight, c.Seven, c.Zero, c.Miss}),
		slog.String("status", string(segment.Status)),
		slog.Any("locked_by", lockedBy),
		slog.String("scorer_name", segment.ScorerName),
		slog.String("locked_at", segment.LockedAt),
		slog.String("accepted_by", segment.AcceptedBy),
		slog.String("accepted_at", segment.AcceptedAt),
		slog.String("disputed_at", segment.DisputedAt),
	}, nil
}

func (l *ScoreLog) Reemit(ctx context.Context, f repository.LockedSegmentFilter) {
	if l == nil || l.log == nil {
		return
	}
	ctx = context.WithoutCancel(ctx)
	refs, err := l.repo.ListLockedSegments(ctx, f)
	if err != nil {
		l.log.ErrorContext(ctx, "score log context failed", "event", eventSegmentStatus, "error", err)
		return
	}
	for _, ref := range refs {
		l.EmitSegment(ctx, ref.MatchID, ref.EntryID, ref.Number)
	}
}
