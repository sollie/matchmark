package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/sollie/matchmark/internal/rulebook"
)

const scoreLogSchemaVersion = 1

const (
	eventStageLocked   = "stage_locked"
	eventStageUnlocked = "stage_unlocked"
)

func WithScoreLog(l *slog.Logger) Option {
	return func(s *Service) { s.scoreLog = l }
}

func (s *Service) emitStageEvent(ctx context.Context, event string, competitionID, entryID int64, stageNumber int) {
	if s.scoreLog == nil {
		return
	}
	attrs, err := s.stageEventAttrs(ctx, event, competitionID, entryID, stageNumber)
	if err != nil {
		s.scoreLog.ErrorContext(ctx, "score log context failed", "event", event, "error", err)
		return
	}
	s.scoreLog.LogAttrs(ctx, slog.LevelInfo, event, attrs...)
}

func (s *Service) stageEventAttrs(ctx context.Context, event string, competitionID, entryID int64, stageNumber int) ([]slog.Attr, error) {
	comp, err := s.repo.GetCompetition(ctx, competitionID)
	if err != nil {
		return nil, err
	}
	course, err := rulebook.CourseByKey(comp.Course)
	if err != nil {
		return nil, err
	}
	entry, err := s.repo.GetEntry(ctx, competitionID, entryID)
	if err != nil {
		return nil, err
	}
	shooter, err := s.repo.GetShooter(ctx, entry.ShooterID)
	if err != nil {
		return nil, err
	}
	stage, err := s.repo.GetStage(ctx, competitionID, entryID, stageNumber)
	if err != nil {
		return nil, err
	}
	c := stage.Counts
	var lockedBy any = "Admin"
	if stage.ScorerLane != nil {
		lockedBy = *stage.ScorerLane
	}
	return []slog.Attr{
		slog.Int("schema_version", scoreLogSchemaVersion),
		slog.String("event", event),
		slog.String("emitted_at", time.Now().UTC().Format(time.RFC3339)),
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
		slog.Any("locked_by", lockedBy),
		slog.String("scorer_name", stage.ScorerName),
		slog.String("locked_at", stage.LockedAt),
	}, nil
}

func (s *Service) reemitLockedStages(ctx context.Context, competitionID, entryID, shooterID int64) {
	if s.scoreLog == nil {
		return
	}
	refs, err := s.repo.ListLockedStages(ctx, competitionID, entryID, shooterID)
	if err != nil {
		s.scoreLog.ErrorContext(ctx, "score log context failed", "event", eventStageLocked, "error", err)
		return
	}
	for _, ref := range refs {
		s.emitStageEvent(ctx, eventStageLocked, ref.CompetitionID, ref.EntryID, ref.Number)
	}
}
