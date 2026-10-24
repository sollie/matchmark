package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strconv"

	"github.com/sollie/matchmark/internal/repository"
)

type Device struct {
	CompetitionID int64
	Lane          int
	Finished      bool
}

type ScorerSession struct {
	CompetitionID int64
	Lane          int
	CurrentStage  int
	ScoreLane     int
	EntryID       int64
	ShooterName   string
	Stages        []repository.StageInfo
	Phase         Phase
	Acceptance    *Acceptance
}

type Phase string

const (
	PhaseScoring         Phase = "scoring"
	PhaseWaiting         Phase = "waiting"
	PhaseNeedsAcceptance Phase = "needs_acceptance"
	PhaseNextStage       Phase = "next_stage"
	PhaseFinished        Phase = "finished"
	PhaseNotStarted      Phase = "not_started"
)

type Acceptance struct {
	StageNumber int
	Counts      repository.RingCounts
	Total       int
	ScorerName  string
}

func stageOf(e repository.EntryWithStages, number int) (repository.StageInfo, bool) {
	for _, st := range e.Stages {
		if st.Number == number {
			return st, true
		}
	}
	return repository.StageInfo{}, false
}

func derivePhase(t ScoreTarget) (Phase, *Acceptance) {
	if st, ok := stageOf(t.Entry, t.CurrentStage); ok && st.Status == repository.StatusOpen {
		if st.Started {
			return PhaseNextStage, nil
		}
		return PhaseScoring, nil
	}
	if st, ok := stageOf(t.Own, t.CurrentStage); ok && st.Status == repository.StatusAwaitingAcceptance {
		c := st.Counts
		name := ""
		if st.ScorerName != nil {
			name = *st.ScorerName
		}
		return PhaseNeedsAcceptance, &Acceptance{
			StageNumber: st.Number, Counts: c, ScorerName: name,
			Total: c.X + c.Ten + c.Nine + c.Eight + c.Seven + c.Zero + c.Miss,
		}
	}
	return PhaseWaiting, nil
}

func (s *Service) CreatePairing(ctx context.Context, competitionID int64, lane int) (int64, string, error) {
	comp, err := s.repo.GetCompetition(ctx, competitionID)
	if err != nil {
		return 0, "", mapRepoErr(err)
	}
	if CompetitionStatus(comp.Status) == StatusFinished {
		return 0, "", ErrConflict
	}
	if _, err := s.repo.MannedEntryByLane(ctx, competitionID, lane); err != nil {
		if mapRepoErr(err) == ErrNotFound {
			return 0, "", ErrLaneNotManned
		}
		return 0, "", err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return 0, "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	id, err := s.repo.CreatePairing(ctx, competitionID, lane, hashToken(token))
	if err != nil {
		return 0, "", mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "device.pair", Target: idTarget("competition", competitionID), Details: "lane " + strconv.Itoa(lane)}, CompetitionID: competitionID, Live: LiveCompetition})
	return id, token, nil
}

func (s *Service) ListPairings(ctx context.Context, competitionID int64) ([]repository.Pairing, error) {
	if _, err := s.repo.GetCompetition(ctx, competitionID); err != nil {
		return nil, mapRepoErr(err)
	}
	pairings, err := s.repo.ListPairings(ctx, competitionID)
	return pairings, mapRepoErr(err)
}

func (s *Service) RevokePairing(ctx context.Context, competitionID, pairingID int64) error {
	if err := s.repo.RevokePairing(ctx, competitionID, pairingID); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "device.revoke", Target: idTarget("competition", competitionID), Details: "pairing " + strconv.FormatInt(pairingID, 10)}, CompetitionID: competitionID, Live: LiveCompetition})
	return nil
}

func (s *Service) AuthenticateDevice(ctx context.Context, token string) (Device, error) {
	p, err := s.repo.PairingByTokenHash(ctx, hashToken(token))
	if err != nil {
		return Device{}, ErrUnauthenticated
	}
	if p.Revoked {
		comp, err := s.repo.GetCompetition(ctx, p.CompetitionID)
		if err != nil || CompetitionStatus(comp.Status) != StatusFinished {
			return Device{}, ErrUnauthenticated
		}
		return Device{CompetitionID: p.CompetitionID, Lane: p.Lane, Finished: true}, nil
	}
	return Device{CompetitionID: p.CompetitionID, Lane: p.Lane}, nil
}

func shooterName(sh repository.Shooter) string {
	return sh.FirstName + " " + sh.LastName
}

func (s *Service) DeviceSession(ctx context.Context, d Device) (ScorerSession, error) {
	if d.Finished {
		return ScorerSession{CompetitionID: d.CompetitionID, Lane: d.Lane, Phase: PhaseFinished}, nil
	}
	comp, err := s.repo.GetCompetition(ctx, d.CompetitionID)
	if err != nil {
		return ScorerSession{}, mapRepoErr(err)
	}
	if CompetitionStatus(comp.Status) == StatusDraft {
		return ScorerSession{CompetitionID: d.CompetitionID, Lane: d.Lane, Phase: PhaseNotStarted}, nil
	}
	t, err := s.ScoreTarget(ctx, d)
	if err != nil {
		return ScorerSession{}, err
	}
	phase, acceptance := derivePhase(t)
	return ScorerSession{CompetitionID: d.CompetitionID, Lane: d.Lane, CurrentStage: t.CurrentStage, ScoreLane: t.ScoreLane, EntryID: t.Entry.ID, ShooterName: t.ShooterName, Stages: t.Entry.Stages, Phase: phase, Acceptance: acceptance}, nil
}

func (s *Service) DeviceSetRingCounts(ctx context.Context, d Device, stageNumber int, c repository.RingCounts) error {
	t, err := s.ScoreTarget(ctx, d)
	if err != nil {
		return err
	}
	if stageNumber != t.CurrentStage {
		return ErrNotCurrentStage
	}
	if err := s.writeRingCounts(ctx, d.CompetitionID, t.Entry.ID, stageNumber, c); err != nil {
		return err
	}
	s.changes.Record(ctx, Change{
		Audit: AuditEvent{
			Action:  "stage.set_counts",
			Target:  stageTarget(d.CompetitionID, t.Entry.ID, stageNumber),
			Details: "lane " + strconv.Itoa(d.Lane) + " competition " + strconv.FormatInt(d.CompetitionID, 10),
		},
		Actor:         &Actor{Username: DeviceActor, IP: ActorFrom(ctx).IP},
		CompetitionID: d.CompetitionID,
		Live:          LiveCompetition,
	})
	return nil
}

func (s *Service) DeviceLockStage(ctx context.Context, d Device, stageNumber int) (repository.StageStatus, error) {
	t, err := s.ScoreTarget(ctx, d)
	if err != nil {
		return "", err
	}
	if stageNumber != t.CurrentStage {
		return "", ErrNotCurrentStage
	}
	stage, err := s.repo.GetStage(ctx, d.CompetitionID, t.Entry.ID, stageNumber)
	if err != nil {
		return "", mapRepoErr(err)
	}
	if stage.Status != repository.StatusOpen {
		return "", ErrStageLocked
	}
	if err := ValidateRingCounts(stage.Counts, stage.ExpectedShots); err != nil {
		return "", err
	}
	ev := repository.StageEvent{
		CompetitionID: d.CompetitionID, EntryID: t.Entry.ID, StageNumber: stageNumber, Event: repository.EventLocked,
		ActorType: "device", ActorName: DeviceActor, ScorerName: t.ScorerName, ShooterName: t.ShooterName,
	}
	if err := s.repo.LockStage(ctx, stage.ID, d.Lane, t.ScorerName, ev); err != nil {
		return "", mapStageWriteErr(err)
	}
	s.changes.Record(ctx, Change{
		Audit: AuditEvent{
			Action:  "stage.lock",
			Target:  stageTarget(d.CompetitionID, t.Entry.ID, stageNumber),
			Details: "lane " + strconv.Itoa(d.Lane) + " competition " + strconv.FormatInt(d.CompetitionID, 10),
		},
		Actor:         &Actor{Username: DeviceActor, IP: ActorFrom(ctx).IP},
		CompetitionID: d.CompetitionID,
		Live:          LiveCompetition,
		Stage:         &StageEvent{CompetitionID: d.CompetitionID, EntryID: t.Entry.ID, StageNumber: stageNumber},
	})
	return repository.StatusAwaitingAcceptance, nil
}

func (s *Service) DeviceAcceptStage(ctx context.Context, d Device, stageNumber int) (repository.StageStatus, error) {
	stage, ev, err := s.ownAwaitingStage(ctx, d, stageNumber, repository.EventAccepted)
	if err != nil {
		return "", err
	}
	if err := s.repo.AcceptStage(ctx, stage.ID, strconv.Itoa(d.Lane), ev); err != nil {
		return "", mapAcceptanceErr(err)
	}
	s.recordDeviceStageChange(ctx, d, ev, "stage.accept")
	s.finishIfComplete(ctx, d.CompetitionID, &Actor{Username: DeviceActor, IP: ActorFrom(ctx).IP})
	return repository.StatusAccepted, nil
}

func (s *Service) DeviceDisputeStage(ctx context.Context, d Device, stageNumber int) (repository.StageStatus, error) {
	stage, ev, err := s.ownAwaitingStage(ctx, d, stageNumber, repository.EventDisputed)
	if err != nil {
		return "", err
	}
	if err := s.repo.DisputeStage(ctx, stage.ID, ev); err != nil {
		return "", mapAcceptanceErr(err)
	}
	s.recordDeviceStageChange(ctx, d, ev, "stage.dispute")
	return repository.StatusOpen, nil
}

func (s *Service) ownAwaitingStage(ctx context.Context, d Device, stageNumber int, kind repository.StageEventKind) (repository.Stage, repository.StageEvent, error) {
	comp, err := s.repo.GetCompetition(ctx, d.CompetitionID)
	if err != nil {
		return repository.Stage{}, repository.StageEvent{}, mapRepoErr(err)
	}
	if CompetitionStatus(comp.Status) != StatusActive {
		return repository.Stage{}, repository.StageEvent{}, ErrConflict
	}
	if stageNumber != comp.CurrentStage {
		return repository.Stage{}, repository.StageEvent{}, ErrNotCurrentStage
	}
	entry, err := s.repo.MannedEntryByLane(ctx, d.CompetitionID, d.Lane)
	if err != nil {
		return repository.Stage{}, repository.StageEvent{}, mapRepoErr(err)
	}
	stage, err := s.repo.GetStage(ctx, d.CompetitionID, entry.ID, stageNumber)
	if err != nil {
		return repository.Stage{}, repository.StageEvent{}, mapRepoErr(err)
	}
	if stage.Status != repository.StatusAwaitingAcceptance {
		return repository.Stage{}, repository.StageEvent{}, ErrStageNotAwaitingAcceptance
	}
	if stage.ScorerLane != nil && *stage.ScorerLane == d.Lane {
		return repository.Stage{}, repository.StageEvent{}, ErrForbidden
	}
	shooter, err := s.repo.GetShooter(ctx, entry.ShooterID)
	if err != nil {
		return repository.Stage{}, repository.StageEvent{}, mapRepoErr(err)
	}
	ev := repository.StageEvent{
		CompetitionID: d.CompetitionID, EntryID: entry.ID, StageNumber: stageNumber, Event: kind,
		ActorType: "device", ActorName: DeviceActor, ScorerName: stage.ScorerName, ShooterName: shooterName(shooter),
	}
	return stage, ev, nil
}

func mapAcceptanceErr(err error) error {
	if errors.Is(err, repository.ErrConflict) {
		return ErrStageNotAwaitingAcceptance
	}
	return mapRepoErr(err)
}

func (s *Service) recordDeviceStageChange(ctx context.Context, d Device, ev repository.StageEvent, action string) {
	s.changes.Record(ctx, Change{
		Audit: AuditEvent{
			Action:  action,
			Target:  stageTarget(d.CompetitionID, ev.EntryID, ev.StageNumber),
			Details: "lane " + strconv.Itoa(d.Lane) + " competition " + strconv.FormatInt(d.CompetitionID, 10),
		},
		Actor:         &Actor{Username: DeviceActor, IP: ActorFrom(ctx).IP},
		CompetitionID: d.CompetitionID,
		Live:          LiveCompetition,
		Stage:         &StageEvent{CompetitionID: d.CompetitionID, EntryID: ev.EntryID, StageNumber: ev.StageNumber},
	})
}

func (s *Service) AdminLockStage(ctx context.Context, competitionID, entryID int64, stageNumber int) (repository.StageStatus, error) {
	stage, err := s.repo.GetStage(ctx, competitionID, entryID, stageNumber)
	if err != nil {
		return "", mapRepoErr(err)
	}
	if stage.Status != repository.StatusOpen {
		return "", ErrStageLocked
	}
	if err := ValidateRingCounts(stage.Counts, stage.ExpectedShots); err != nil {
		return "", err
	}
	username := ActorFrom(ctx).Username
	ev, err := s.userStageEvent(ctx, repository.EventLocked, username, competitionID, entryID, stage)
	if err != nil {
		return "", err
	}
	ev.ScorerName = ""
	if err := s.repo.LockStageAsUser(ctx, stage.ID, username, ev); err != nil {
		return "", mapStageWriteErr(err)
	}
	s.changes.Record(ctx, Change{
		Audit:         AuditEvent{Action: "stage.lock", Target: stageTarget(competitionID, entryID, stageNumber)},
		CompetitionID: competitionID,
		Live:          LiveCompetition,
		Stage:         &StageEvent{CompetitionID: competitionID, EntryID: entryID, StageNumber: stageNumber},
	})
	s.finishIfComplete(ctx, competitionID, nil)
	return repository.StatusAccepted, nil
}

func (s *Service) AdminAcceptStage(ctx context.Context, competitionID, entryID int64, stageNumber int) (repository.StageStatus, error) {
	stage, err := s.repo.GetStage(ctx, competitionID, entryID, stageNumber)
	if err != nil {
		return "", mapRepoErr(err)
	}
	if stage.Status != repository.StatusAwaitingAcceptance {
		return "", ErrStageNotAwaitingAcceptance
	}
	username := ActorFrom(ctx).Username
	ev, err := s.userStageEvent(ctx, repository.EventAccepted, username, competitionID, entryID, stage)
	if err != nil {
		return "", err
	}
	if err := s.repo.AcceptStage(ctx, stage.ID, username, ev); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return "", ErrStageNotAwaitingAcceptance
		}
		return "", mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{
		Audit:         AuditEvent{Action: "stage.accept", Target: stageTarget(competitionID, entryID, stageNumber)},
		CompetitionID: competitionID,
		Live:          LiveCompetition,
		Stage:         &StageEvent{CompetitionID: competitionID, EntryID: entryID, StageNumber: stageNumber},
	})
	s.finishIfComplete(ctx, competitionID, nil)
	return repository.StatusAccepted, nil
}

func (s *Service) UnlockStage(ctx context.Context, competitionID, entryID int64, stageNumber int) (repository.StageStatus, error) {
	stage, err := s.repo.GetStage(ctx, competitionID, entryID, stageNumber)
	if err != nil {
		return "", mapRepoErr(err)
	}
	ev, err := s.userStageEvent(ctx, repository.EventUnlocked, ActorFrom(ctx).Username, competitionID, entryID, stage)
	if err != nil {
		return "", err
	}
	if err := s.repo.UnlockStage(ctx, stage.ID, ev); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return "", ErrStageNotLocked
		}
		return "", mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{
		Audit:         AuditEvent{Action: "stage.unlock", Target: stageTarget(competitionID, entryID, stageNumber)},
		CompetitionID: competitionID,
		Live:          LiveCompetition,
		Stage:         &StageEvent{CompetitionID: competitionID, EntryID: entryID, StageNumber: stageNumber},
	})
	return repository.StatusOpen, nil
}

func (s *Service) DeviceSnapshot(ctx context.Context, token string) (ScorerSession, error) {
	d, err := s.AuthenticateDevice(ctx, token)
	if err != nil {
		return ScorerSession{}, err
	}
	return s.DeviceSession(ctx, d)
}
