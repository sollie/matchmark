package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"strconv"

	"github.com/sollie/matchmark/internal/repository"
)

type Device struct {
	CompetitionID int64
	Lane          int
}

type ScorerSession struct {
	Lane        int
	ScoreLane   int
	EntryID     int64
	ShooterName string
	Stages      []repository.StageInfo
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
	if err != nil || p.Revoked {
		return Device{}, ErrUnauthenticated
	}
	return Device{CompetitionID: p.CompetitionID, Lane: p.Lane}, nil
}

func shooterName(sh repository.Shooter) string {
	return sh.FirstName + " " + sh.LastName
}

func (s *Service) DeviceSession(ctx context.Context, d Device) (ScorerSession, error) {
	t, err := s.ScoreTarget(ctx, d)
	if err != nil {
		return ScorerSession{}, err
	}
	return ScorerSession{Lane: d.Lane, ScoreLane: t.ScoreLane, EntryID: t.Entry.ID, ShooterName: t.ShooterName, Stages: t.Entry.Stages}, nil
}

func (s *Service) DeviceSetRingCounts(ctx context.Context, d Device, stageNumber int, c repository.RingCounts) error {
	t, err := s.ScoreTarget(ctx, d)
	if err != nil {
		return err
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

func (s *Service) DeviceLockStage(ctx context.Context, d Device, stageNumber int) error {
	t, err := s.ScoreTarget(ctx, d)
	if err != nil {
		return err
	}
	stage, err := s.repo.GetStage(ctx, d.CompetitionID, t.Entry.ID, stageNumber)
	if err != nil {
		return mapRepoErr(err)
	}
	if stage.Locked {
		return ErrStageLocked
	}
	if err := ValidateRingCounts(stage.Counts, stage.ExpectedShots); err != nil {
		return err
	}
	if err := s.repo.LockStage(ctx, stage.ID, d.Lane, t.ScorerName); err != nil {
		if err == repository.ErrConflict {
			return ErrStageLocked
		}
		return mapRepoErr(err)
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
		Stage:         &StageEvent{Event: eventStageLocked, CompetitionID: d.CompetitionID, EntryID: t.Entry.ID, StageNumber: stageNumber},
	})
	return nil
}

func (s *Service) AdminLockStage(ctx context.Context, competitionID, entryID int64, stageNumber int) error {
	stage, err := s.repo.GetStage(ctx, competitionID, entryID, stageNumber)
	if err != nil {
		return mapRepoErr(err)
	}
	if stage.Locked {
		return ErrStageLocked
	}
	if err := ValidateRingCounts(stage.Counts, stage.ExpectedShots); err != nil {
		return err
	}
	if err := s.repo.LockStageAsUser(ctx, stage.ID, ActorFrom(ctx).Username); err != nil {
		if err == repository.ErrConflict {
			return ErrStageLocked
		}
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{
		Audit:         AuditEvent{Action: "stage.lock", Target: stageTarget(competitionID, entryID, stageNumber)},
		CompetitionID: competitionID,
		Live:          LiveCompetition,
		Stage:         &StageEvent{Event: eventStageLocked, CompetitionID: competitionID, EntryID: entryID, StageNumber: stageNumber},
	})
	return nil
}

func (s *Service) UnlockStage(ctx context.Context, competitionID, entryID int64, stageNumber int) error {
	stage, err := s.repo.GetStage(ctx, competitionID, entryID, stageNumber)
	if err != nil {
		return mapRepoErr(err)
	}
	if err := s.repo.UnlockStage(ctx, stage.ID); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{
		Audit:         AuditEvent{Action: "stage.unlock", Target: stageTarget(competitionID, entryID, stageNumber)},
		CompetitionID: competitionID,
		Live:          LiveCompetition,
		Stage:         &StageEvent{Event: eventStageUnlocked, CompetitionID: competitionID, EntryID: entryID, StageNumber: stageNumber, Snapshot: &stage},
	})
	return nil
}

func (s *Service) DeviceSnapshot(ctx context.Context, token string) (ScorerSession, error) {
	d, err := s.AuthenticateDevice(ctx, token)
	if err != nil {
		return ScorerSession{}, err
	}
	return s.DeviceSession(ctx, d)
}
