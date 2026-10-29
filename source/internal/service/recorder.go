package service

import (
	"context"

	"github.com/sollie/matchmark/internal/repository"
)

type LiveScope int

const (
	LiveNone LiveScope = iota
	LiveMatch
	LiveAll
)

type StageEvent struct {
	MatchID     int64
	EntryID     int64
	StageNumber int
}

type Change struct {
	Audit   AuditEvent
	Actor   *Actor
	MatchID int64
	Live    LiveScope
	Screens bool
	Reemit  *repository.LockedStageFilter
	Stage   *StageEvent
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
		r.scoreLog.EmitStage(ctx, e.MatchID, e.EntryID, e.StageNumber)
	}
	if c.Screens {
		r.bus.Publish(screenTopic)
	}
	switch c.Live {
	case LiveMatch:
		r.bus.Publish(c.MatchID)
	case LiveAll:
		r.bus.PublishAll()
	}
}
