package service

import (
	"context"
	"errors"
	"strconv"

	"github.com/sollie/matchmark/internal/repository"
)

type SegmentAction string

const (
	ActionLock    SegmentAction = "lock"
	ActionAccept  SegmentAction = "accept"
	ActionDispute SegmentAction = "dispute"
	ActionUnlock  SegmentAction = "unlock"
)

type segmentRule struct {
	allowDevice  bool
	allowAdmin   bool
	event        repository.SegmentEventKind
	audit        string
	from         repository.SegmentStatus
	fromErr      error
	validCounts  bool
	deviceResult repository.SegmentStatus
	adminResult  repository.SegmentStatus
	finishDevice bool
	finishAdmin  bool
}

var segmentRules = map[SegmentAction]segmentRule{
	ActionLock: {
		allowDevice: true, allowAdmin: true, event: repository.EventLocked, audit: "segment.lock",
		from: repository.StatusOpen, fromErr: ErrSegmentLocked, validCounts: true,
		deviceResult: repository.StatusAwaitingAcceptance, adminResult: repository.StatusAccepted,
		finishAdmin: true,
	},
	ActionAccept: {
		allowDevice: true, allowAdmin: true, event: repository.EventAccepted, audit: "segment.accept",
		from: repository.StatusAwaitingAcceptance, fromErr: ErrSegmentNotAwaitingAcceptance,
		deviceResult: repository.StatusAccepted, adminResult: repository.StatusAccepted,
		finishDevice: true, finishAdmin: true,
	},
	ActionDispute: {
		allowDevice: true, event: repository.EventDisputed, audit: "segment.dispute",
		from: repository.StatusAwaitingAcceptance, fromErr: ErrSegmentNotAwaitingAcceptance,
		deviceResult: repository.StatusOpen,
	},
	ActionUnlock: {
		allowAdmin: true, event: repository.EventUnlocked, audit: "segment.unlock",
		adminResult: repository.StatusOpen,
	},
}

type segmentActor struct {
	device *Device
}

func deviceSegmentActor(d Device) segmentActor { return segmentActor{device: &d} }

func adminSegmentActor() segmentActor { return segmentActor{} }

type segmentRef struct {
	matchID int64
	entryID int64
	number  int
}

type segmentLocation struct {
	ref         segmentRef
	segment     repository.Segment
	shooterID   int64
	scorerName  string
	shooterName string
	named       bool
}

func (s *Service) transitionSegment(ctx context.Context, actor segmentActor, action SegmentAction, ref segmentRef) (repository.SegmentStatus, error) {
	rule := segmentRules[action]
	if (actor.device != nil && !rule.allowDevice) || (actor.device == nil && !rule.allowAdmin) {
		return "", ErrForbidden
	}
	loc, err := s.locateSegment(ctx, actor, action, ref)
	if err != nil {
		return "", err
	}
	if err := checkSegmentLegal(rule, loc.segment); err != nil {
		return "", err
	}
	if actor.device != nil && action != ActionLock && loc.segment.ScorerLane != nil && *loc.segment.ScorerLane == actor.device.Lane {
		return "", ErrForbidden
	}
	ev, err := s.segmentTransitionEvent(ctx, actor, action, loc)
	if err != nil {
		return "", err
	}
	if err := s.mutateSegment(ctx, actor, action, loc, ev); err != nil {
		return "", err
	}
	s.recordSegmentTransition(ctx, actor, rule, loc.ref)
	if actor.device != nil && rule.finishDevice {
		s.finishIfComplete(ctx, loc.ref.matchID, deviceChangeActor(ctx))
	}
	if actor.device == nil && rule.finishAdmin {
		s.finishIfComplete(ctx, loc.ref.matchID, nil)
	}
	if actor.device != nil {
		return rule.deviceResult, nil
	}
	return rule.adminResult, nil
}

func checkSegmentLegal(rule segmentRule, segment repository.Segment) error {
	if rule.from != "" && segment.Status != rule.from {
		return rule.fromErr
	}
	if rule.validCounts {
		return ValidateRingCounts(segment.Counts, segment.ExpectedShots)
	}
	return nil
}

func (s *Service) locateSegment(ctx context.Context, actor segmentActor, action SegmentAction, ref segmentRef) (segmentLocation, error) {
	if actor.device == nil {
		segment, err := s.repo.GetSegment(ctx, ref.matchID, ref.entryID, ref.number)
		if err != nil {
			return segmentLocation{}, mapRepoErr(err)
		}
		return segmentLocation{ref: ref, segment: segment}, nil
	}
	d := *actor.device
	if action == ActionLock {
		t, err := s.ScoreTarget(ctx, d)
		if err != nil {
			return segmentLocation{}, err
		}
		if ref.number != t.CurrentSegment {
			return segmentLocation{}, ErrNotCurrentSegment
		}
		segment, err := s.repo.GetSegment(ctx, d.MatchID, t.Entry.ID, ref.number)
		if err != nil {
			return segmentLocation{}, mapRepoErr(err)
		}
		return segmentLocation{
			ref: segmentRef{d.MatchID, t.Entry.ID, ref.number}, segment: segment,
			scorerName: t.ScorerName, shooterName: t.ShooterName, named: true,
		}, nil
	}
	comp, err := s.repo.GetMatch(ctx, d.MatchID)
	if err != nil {
		return segmentLocation{}, mapRepoErr(err)
	}
	if MatchStatus(comp.Status) != StatusActive {
		return segmentLocation{}, ErrConflict
	}
	if ref.number != comp.CurrentSegment {
		return segmentLocation{}, ErrNotCurrentSegment
	}
	entry, err := s.repo.MannedEntryByLane(ctx, d.MatchID, d.Lane)
	if err != nil {
		return segmentLocation{}, mapRepoErr(err)
	}
	segment, err := s.repo.GetSegment(ctx, d.MatchID, entry.ID, ref.number)
	if err != nil {
		return segmentLocation{}, mapRepoErr(err)
	}
	return segmentLocation{ref: segmentRef{d.MatchID, entry.ID, ref.number}, segment: segment, shooterID: entry.ShooterID}, nil
}

func (s *Service) segmentTransitionEvent(ctx context.Context, actor segmentActor, action SegmentAction, loc segmentLocation) (repository.SegmentEvent, error) {
	kind := segmentRules[action].event
	if actor.device == nil {
		ev, err := s.userSegmentEvent(ctx, kind, ActorFrom(ctx).Username, loc.ref.matchID, loc.ref.entryID, loc.segment)
		if err != nil {
			return repository.SegmentEvent{}, err
		}
		if action == ActionLock {
			ev.ScorerName = ""
		}
		return ev, nil
	}
	ev := repository.SegmentEvent{
		MatchID: loc.ref.matchID, EntryID: loc.ref.entryID, SegmentNumber: loc.ref.number, Event: kind,
		ActorType: "device", ActorName: DeviceActor,
	}
	if loc.named {
		ev.ScorerName, ev.ShooterName = loc.scorerName, loc.shooterName
		return ev, nil
	}
	shooter, err := s.repo.GetShooter(ctx, loc.shooterID)
	if err != nil {
		return repository.SegmentEvent{}, mapRepoErr(err)
	}
	ev.ScorerName, ev.ShooterName = loc.segment.ScorerName, shooterName(shooter)
	return ev, nil
}

func (s *Service) mutateSegment(ctx context.Context, actor segmentActor, action SegmentAction, loc segmentLocation, ev repository.SegmentEvent) error {
	id := loc.segment.ID
	username := ActorFrom(ctx).Username
	switch action {
	case ActionLock:
		if actor.device != nil {
			return mapSegmentWriteErr(s.repo.LockSegment(ctx, id, actor.device.Lane, loc.scorerName, ev))
		}
		return mapSegmentWriteErr(s.repo.LockSegmentAsUser(ctx, id, username, ev))
	case ActionAccept:
		by := username
		if actor.device != nil {
			by = strconv.Itoa(actor.device.Lane)
		}
		return mapAcceptanceErr(s.repo.AcceptSegment(ctx, id, by, ev))
	case ActionDispute:
		return mapAcceptanceErr(s.repo.DisputeSegment(ctx, id, ev))
	default:
		err := s.repo.UnlockSegment(ctx, id, ev)
		if errors.Is(err, repository.ErrConflict) {
			return ErrSegmentNotLocked
		}
		return mapRepoErr(err)
	}
}

func deviceChangeActor(ctx context.Context) *Actor {
	return &Actor{Username: DeviceActor, IP: ActorFrom(ctx).IP}
}

func (s *Service) recordSegmentTransition(ctx context.Context, actor segmentActor, rule segmentRule, ref segmentRef) {
	change := Change{
		Audit:   AuditEvent{Action: rule.audit, Target: segmentTarget(ref.matchID, ref.entryID, ref.number)},
		MatchID: ref.matchID,
		Live:    LiveMatch,
		Segment: &SegmentEvent{MatchID: ref.matchID, EntryID: ref.entryID, SegmentNumber: ref.number},
	}
	if actor.device != nil {
		change.Audit.Details = "lane " + strconv.Itoa(actor.device.Lane) + " match " + strconv.FormatInt(ref.matchID, 10)
		change.Actor = deviceChangeActor(ctx)
	}
	s.changes.Record(ctx, change)
}

func (s *Service) DeviceLockSegment(ctx context.Context, d Device, segmentNumber int) (repository.SegmentStatus, error) {
	return s.transitionSegment(ctx, deviceSegmentActor(d), ActionLock, segmentRef{matchID: d.MatchID, number: segmentNumber})
}

func (s *Service) DeviceAcceptSegment(ctx context.Context, d Device, segmentNumber int) (repository.SegmentStatus, error) {
	return s.transitionSegment(ctx, deviceSegmentActor(d), ActionAccept, segmentRef{matchID: d.MatchID, number: segmentNumber})
}

func (s *Service) DeviceDisputeSegment(ctx context.Context, d Device, segmentNumber int) (repository.SegmentStatus, error) {
	return s.transitionSegment(ctx, deviceSegmentActor(d), ActionDispute, segmentRef{matchID: d.MatchID, number: segmentNumber})
}

func (s *Service) AdminLockSegment(ctx context.Context, matchID, entryID int64, segmentNumber int) (repository.SegmentStatus, error) {
	return s.transitionSegment(ctx, adminSegmentActor(), ActionLock, segmentRef{matchID, entryID, segmentNumber})
}

func (s *Service) AdminAcceptSegment(ctx context.Context, matchID, entryID int64, segmentNumber int) (repository.SegmentStatus, error) {
	return s.transitionSegment(ctx, adminSegmentActor(), ActionAccept, segmentRef{matchID, entryID, segmentNumber})
}

func (s *Service) UnlockSegment(ctx context.Context, matchID, entryID int64, segmentNumber int) (repository.SegmentStatus, error) {
	return s.transitionSegment(ctx, adminSegmentActor(), ActionUnlock, segmentRef{matchID, entryID, segmentNumber})
}
