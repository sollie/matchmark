package service

import (
	"context"
	"errors"
	"strconv"

	"github.com/sollie/matchmark/internal/repository"
)

type StageAction string

const (
	ActionLock    StageAction = "lock"
	ActionAccept  StageAction = "accept"
	ActionDispute StageAction = "dispute"
	ActionUnlock  StageAction = "unlock"
)

type stageRule struct {
	allowDevice  bool
	allowAdmin   bool
	event        repository.StageEventKind
	audit        string
	from         repository.StageStatus
	fromErr      error
	validCounts  bool
	deviceResult repository.StageStatus
	adminResult  repository.StageStatus
	finishDevice bool
	finishAdmin  bool
}

var stageRules = map[StageAction]stageRule{
	ActionLock: {
		allowDevice: true, allowAdmin: true, event: repository.EventLocked, audit: "stage.lock",
		from: repository.StatusOpen, fromErr: ErrStageLocked, validCounts: true,
		deviceResult: repository.StatusAwaitingAcceptance, adminResult: repository.StatusAccepted,
		finishAdmin: true,
	},
	ActionAccept: {
		allowDevice: true, allowAdmin: true, event: repository.EventAccepted, audit: "stage.accept",
		from: repository.StatusAwaitingAcceptance, fromErr: ErrStageNotAwaitingAcceptance,
		deviceResult: repository.StatusAccepted, adminResult: repository.StatusAccepted,
		finishDevice: true, finishAdmin: true,
	},
	ActionDispute: {
		allowDevice: true, event: repository.EventDisputed, audit: "stage.dispute",
		from: repository.StatusAwaitingAcceptance, fromErr: ErrStageNotAwaitingAcceptance,
		deviceResult: repository.StatusOpen,
	},
	ActionUnlock: {
		allowAdmin: true, event: repository.EventUnlocked, audit: "stage.unlock",
		adminResult: repository.StatusOpen,
	},
}

type stageActor struct {
	device *Device
}

func deviceStageActor(d Device) stageActor { return stageActor{device: &d} }

func adminStageActor() stageActor { return stageActor{} }

type stageRef struct {
	competitionID int64
	entryID       int64
	number        int
}

type stageLocation struct {
	ref         stageRef
	stage       repository.Stage
	shooterID   int64
	scorerName  string
	shooterName string
	named       bool
}

func (s *Service) transitionStage(ctx context.Context, actor stageActor, action StageAction, ref stageRef) (repository.StageStatus, error) {
	rule := stageRules[action]
	if (actor.device != nil && !rule.allowDevice) || (actor.device == nil && !rule.allowAdmin) {
		return "", ErrForbidden
	}
	loc, err := s.locateStage(ctx, actor, action, ref)
	if err != nil {
		return "", err
	}
	if err := checkStageLegal(rule, loc.stage); err != nil {
		return "", err
	}
	if actor.device != nil && action != ActionLock && loc.stage.ScorerLane != nil && *loc.stage.ScorerLane == actor.device.Lane {
		return "", ErrForbidden
	}
	ev, err := s.stageTransitionEvent(ctx, actor, action, loc)
	if err != nil {
		return "", err
	}
	if err := s.mutateStage(ctx, actor, action, loc, ev); err != nil {
		return "", err
	}
	s.recordStageTransition(ctx, actor, rule, loc.ref)
	if actor.device != nil && rule.finishDevice {
		s.finishIfComplete(ctx, loc.ref.competitionID, deviceChangeActor(ctx))
	}
	if actor.device == nil && rule.finishAdmin {
		s.finishIfComplete(ctx, loc.ref.competitionID, nil)
	}
	if actor.device != nil {
		return rule.deviceResult, nil
	}
	return rule.adminResult, nil
}

func checkStageLegal(rule stageRule, stage repository.Stage) error {
	if rule.from != "" && stage.Status != rule.from {
		return rule.fromErr
	}
	if rule.validCounts {
		return ValidateRingCounts(stage.Counts, stage.ExpectedShots)
	}
	return nil
}

func (s *Service) locateStage(ctx context.Context, actor stageActor, action StageAction, ref stageRef) (stageLocation, error) {
	if actor.device == nil {
		stage, err := s.repo.GetStage(ctx, ref.competitionID, ref.entryID, ref.number)
		if err != nil {
			return stageLocation{}, mapRepoErr(err)
		}
		return stageLocation{ref: ref, stage: stage}, nil
	}
	d := *actor.device
	if action == ActionLock {
		t, err := s.ScoreTarget(ctx, d)
		if err != nil {
			return stageLocation{}, err
		}
		if ref.number != t.CurrentStage {
			return stageLocation{}, ErrNotCurrentStage
		}
		stage, err := s.repo.GetStage(ctx, d.CompetitionID, t.Entry.ID, ref.number)
		if err != nil {
			return stageLocation{}, mapRepoErr(err)
		}
		return stageLocation{
			ref: stageRef{d.CompetitionID, t.Entry.ID, ref.number}, stage: stage,
			scorerName: t.ScorerName, shooterName: t.ShooterName, named: true,
		}, nil
	}
	comp, err := s.repo.GetCompetition(ctx, d.CompetitionID)
	if err != nil {
		return stageLocation{}, mapRepoErr(err)
	}
	if CompetitionStatus(comp.Status) != StatusActive {
		return stageLocation{}, ErrConflict
	}
	if ref.number != comp.CurrentStage {
		return stageLocation{}, ErrNotCurrentStage
	}
	entry, err := s.repo.MannedEntryByLane(ctx, d.CompetitionID, d.Lane)
	if err != nil {
		return stageLocation{}, mapRepoErr(err)
	}
	stage, err := s.repo.GetStage(ctx, d.CompetitionID, entry.ID, ref.number)
	if err != nil {
		return stageLocation{}, mapRepoErr(err)
	}
	return stageLocation{ref: stageRef{d.CompetitionID, entry.ID, ref.number}, stage: stage, shooterID: entry.ShooterID}, nil
}

func (s *Service) stageTransitionEvent(ctx context.Context, actor stageActor, action StageAction, loc stageLocation) (repository.StageEvent, error) {
	kind := stageRules[action].event
	if actor.device == nil {
		ev, err := s.userStageEvent(ctx, kind, ActorFrom(ctx).Username, loc.ref.competitionID, loc.ref.entryID, loc.stage)
		if err != nil {
			return repository.StageEvent{}, err
		}
		if action == ActionLock {
			ev.ScorerName = ""
		}
		return ev, nil
	}
	ev := repository.StageEvent{
		CompetitionID: loc.ref.competitionID, EntryID: loc.ref.entryID, StageNumber: loc.ref.number, Event: kind,
		ActorType: "device", ActorName: DeviceActor,
	}
	if loc.named {
		ev.ScorerName, ev.ShooterName = loc.scorerName, loc.shooterName
		return ev, nil
	}
	shooter, err := s.repo.GetShooter(ctx, loc.shooterID)
	if err != nil {
		return repository.StageEvent{}, mapRepoErr(err)
	}
	ev.ScorerName, ev.ShooterName = loc.stage.ScorerName, shooterName(shooter)
	return ev, nil
}

func (s *Service) mutateStage(ctx context.Context, actor stageActor, action StageAction, loc stageLocation, ev repository.StageEvent) error {
	id := loc.stage.ID
	username := ActorFrom(ctx).Username
	switch action {
	case ActionLock:
		if actor.device != nil {
			return mapStageWriteErr(s.repo.LockStage(ctx, id, actor.device.Lane, loc.scorerName, ev))
		}
		return mapStageWriteErr(s.repo.LockStageAsUser(ctx, id, username, ev))
	case ActionAccept:
		by := username
		if actor.device != nil {
			by = strconv.Itoa(actor.device.Lane)
		}
		return mapAcceptanceErr(s.repo.AcceptStage(ctx, id, by, ev))
	case ActionDispute:
		return mapAcceptanceErr(s.repo.DisputeStage(ctx, id, ev))
	default:
		err := s.repo.UnlockStage(ctx, id, ev)
		if errors.Is(err, repository.ErrConflict) {
			return ErrStageNotLocked
		}
		return mapRepoErr(err)
	}
}

func deviceChangeActor(ctx context.Context) *Actor {
	return &Actor{Username: DeviceActor, IP: ActorFrom(ctx).IP}
}

func (s *Service) recordStageTransition(ctx context.Context, actor stageActor, rule stageRule, ref stageRef) {
	change := Change{
		Audit:         AuditEvent{Action: rule.audit, Target: stageTarget(ref.competitionID, ref.entryID, ref.number)},
		CompetitionID: ref.competitionID,
		Live:          LiveCompetition,
		Stage:         &StageEvent{CompetitionID: ref.competitionID, EntryID: ref.entryID, StageNumber: ref.number},
	}
	if actor.device != nil {
		change.Audit.Details = "lane " + strconv.Itoa(actor.device.Lane) + " competition " + strconv.FormatInt(ref.competitionID, 10)
		change.Actor = deviceChangeActor(ctx)
	}
	s.changes.Record(ctx, change)
}

func (s *Service) DeviceLockStage(ctx context.Context, d Device, stageNumber int) (repository.StageStatus, error) {
	return s.transitionStage(ctx, deviceStageActor(d), ActionLock, stageRef{competitionID: d.CompetitionID, number: stageNumber})
}

func (s *Service) DeviceAcceptStage(ctx context.Context, d Device, stageNumber int) (repository.StageStatus, error) {
	return s.transitionStage(ctx, deviceStageActor(d), ActionAccept, stageRef{competitionID: d.CompetitionID, number: stageNumber})
}

func (s *Service) DeviceDisputeStage(ctx context.Context, d Device, stageNumber int) (repository.StageStatus, error) {
	return s.transitionStage(ctx, deviceStageActor(d), ActionDispute, stageRef{competitionID: d.CompetitionID, number: stageNumber})
}

func (s *Service) AdminLockStage(ctx context.Context, competitionID, entryID int64, stageNumber int) (repository.StageStatus, error) {
	return s.transitionStage(ctx, adminStageActor(), ActionLock, stageRef{competitionID, entryID, stageNumber})
}

func (s *Service) AdminAcceptStage(ctx context.Context, competitionID, entryID int64, stageNumber int) (repository.StageStatus, error) {
	return s.transitionStage(ctx, adminStageActor(), ActionAccept, stageRef{competitionID, entryID, stageNumber})
}

func (s *Service) UnlockStage(ctx context.Context, competitionID, entryID int64, stageNumber int) (repository.StageStatus, error) {
	return s.transitionStage(ctx, adminStageActor(), ActionUnlock, stageRef{competitionID, entryID, stageNumber})
}
