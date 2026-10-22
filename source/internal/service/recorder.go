package service

import (
	"context"

	"github.com/sollie/matchmark/internal/repository"
)

type LiveScope int

const (
	LiveNone LiveScope = iota
	LiveCompetition
	LiveAll
)

type StageEvent struct {
	Event         string
	CompetitionID int64
	EntryID       int64
	StageNumber   int
	Snapshot      *repository.Stage
}

type Change struct {
	Audit         AuditEvent
	Actor         *Actor
	CompetitionID int64
	Live          LiveScope
	Reemit        *repository.LockedStageFilter
	Stage         *StageEvent
}

type ChangeRecorder struct {
	audit    *Audit
	scoreLog *ScoreLog
	bus      *Broadcaster
}

func NewChangeRecorder(audit *Audit, scoreLog *ScoreLog, bus *Broadcaster) *ChangeRecorder {
	return &ChangeRecorder{audit: audit, scoreLog: scoreLog, bus: bus}
}

func (r *ChangeRecorder) Record(ctx context.Context, c Change) {
	if c.Audit.Action != "" {
		if c.Actor != nil {
			r.audit.RecordAs(ctx, *c.Actor, c.Audit)
		} else {
			r.audit.Record(ctx, c.Audit)
		}
	}
	if c.Reemit != nil {
		r.scoreLog.Reemit(ctx, *c.Reemit)
	}
	if e := c.Stage; e != nil {
		if e.Snapshot != nil {
			r.scoreLog.EmitStageSnapshot(ctx, e.Event, e.CompetitionID, e.EntryID, *e.Snapshot)
		} else {
			r.scoreLog.EmitStage(ctx, e.Event, e.CompetitionID, e.EntryID, e.StageNumber)
		}
	}
	switch c.Live {
	case LiveCompetition:
		r.bus.Publish(c.CompetitionID)
	case LiveAll:
		r.bus.PublishAll()
	}
}
