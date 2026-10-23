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
	scoreLogSchemaVersion = 2
	emittedAtLayout       = "2006-01-02T15:04:05.000Z"
)

const eventStageStatus = "stage_status"

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

func (l *ScoreLog) EmitStage(ctx context.Context, competitionID, entryID int64, stageNumber int) {
	if l == nil || l.log == nil {
		return
	}
	ctx = context.WithoutCancel(ctx)
	l.mu.Lock()
	defer l.mu.Unlock()
	stage, err := l.repo.GetStage(ctx, competitionID, entryID, stageNumber)
	if err != nil {
		l.log.ErrorContext(ctx, "score log context failed", "event", eventStageStatus, "error", err)
		return
	}
	attrs, err := l.stageEventAttrs(ctx, competitionID, entryID, stage)
	if err != nil {
		l.log.ErrorContext(ctx, "score log context failed", "event", eventStageStatus, "error", err)
		return
	}
	l.log.LogAttrs(ctx, slog.LevelInfo, eventStageStatus, attrs...)
}

func (l *ScoreLog) stageEventAttrs(ctx context.Context, competitionID, entryID int64, stage repository.Stage) ([]slog.Attr, error) {
	comp, err := l.repo.GetCompetition(ctx, competitionID)
	if err != nil {
		return nil, err
	}
	course, err := rulebook.CourseByKey(comp.Course)
	if err != nil {
		return nil, err
	}
	entry, err := l.repo.GetEntry(ctx, competitionID, entryID)
	if err != nil {
		return nil, err
	}
	shooter, err := l.repo.GetShooter(ctx, entry.ShooterID)
	if err != nil {
		return nil, err
	}
	l.seq++
	c := stage.Counts
	var lockedBy any
	if stage.Status != repository.StatusOpen {
		lockedBy = stage.ScorerName
		if stage.ScorerLane != nil {
			lockedBy = *stage.ScorerLane
		}
	}
	return []slog.Attr{
		slog.Int("schema_version", scoreLogSchemaVersion),
		slog.String("event", eventStageStatus),
		slog.String("emitted_at", time.Now().UTC().Format(emittedAtLayout)),
		slog.Int64("seq", l.seq),
		slog.Group("competition",
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
		slog.Group("stage",
			slog.Int("number", stage.Number),
			slog.Int("expected_shots", stage.ExpectedShots),
		),
		slog.Any("ring_counts", []int{c.X, c.Ten, c.Nine, c.Eight, c.Seven, c.Zero, c.Miss}),
		slog.String("status", string(stage.Status)),
		slog.Any("locked_by", lockedBy),
		slog.String("scorer_name", stage.ScorerName),
		slog.String("locked_at", stage.LockedAt),
		slog.String("accepted_by", stage.AcceptedBy),
		slog.String("accepted_at", stage.AcceptedAt),
		slog.String("disputed_at", stage.DisputedAt),
	}, nil
}

func (l *ScoreLog) Reemit(ctx context.Context, f repository.LockedStageFilter) {
	if l == nil || l.log == nil {
		return
	}
	ctx = context.WithoutCancel(ctx)
	refs, err := l.repo.ListLockedStages(ctx, f)
	if err != nil {
		l.log.ErrorContext(ctx, "score log context failed", "event", eventStageStatus, "error", err)
		return
	}
	for _, ref := range refs {
		l.EmitStage(ctx, ref.CompetitionID, ref.EntryID, ref.Number)
	}
}
