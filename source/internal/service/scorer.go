package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"

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
	return mapRepoErr(s.repo.RevokePairing(ctx, competitionID, pairingID))
}

func (s *Service) AuthenticateDevice(ctx context.Context, token string) (Device, error) {
	p, err := s.repo.PairingByTokenHash(ctx, hashToken(token))
	if err != nil || p.Revoked {
		return Device{}, ErrUnauthenticated
	}
	return Device{CompetitionID: p.CompetitionID, Lane: p.Lane}, nil
}

type scoringTarget struct {
	entry repository.Entry
	name  string
	score int
}

func (s *Service) deviceTarget(ctx context.Context, d Device) (scoringTarget, string, error) {
	comp, err := s.repo.GetCompetition(ctx, d.CompetitionID)
	if err != nil {
		return scoringTarget{}, "", mapRepoErr(err)
	}
	if CompetitionStatus(comp.Status) != StatusActive {
		return scoringTarget{}, "", ErrConflict
	}
	scoreLane, err := s.ScoreLane(ctx, d.CompetitionID, d.Lane)
	if err != nil {
		return scoringTarget{}, "", err
	}
	own, err := s.repo.MannedEntryByLane(ctx, d.CompetitionID, d.Lane)
	if err != nil {
		return scoringTarget{}, "", mapRepoErr(err)
	}
	ownShooter, err := s.repo.GetShooter(ctx, own.ShooterID)
	if err != nil {
		return scoringTarget{}, "", mapRepoErr(err)
	}
	target, err := s.repo.MannedEntryByLane(ctx, d.CompetitionID, scoreLane)
	if err != nil {
		return scoringTarget{}, "", mapRepoErr(err)
	}
	targetShooter, err := s.repo.GetShooter(ctx, target.ShooterID)
	if err != nil {
		return scoringTarget{}, "", mapRepoErr(err)
	}
	return scoringTarget{entry: target, name: shooterName(targetShooter), score: scoreLane}, shooterName(ownShooter), nil
}

func shooterName(sh repository.Shooter) string {
	return sh.FirstName + " " + sh.LastName
}

func (s *Service) DeviceSession(ctx context.Context, d Device) (ScorerSession, error) {
	t, _, err := s.deviceTarget(ctx, d)
	if err != nil {
		return ScorerSession{}, err
	}
	entries, err := s.repo.ListEntries(ctx, d.CompetitionID)
	if err != nil {
		return ScorerSession{}, err
	}
	out := ScorerSession{Lane: d.Lane, ScoreLane: t.score, EntryID: t.entry.ID, ShooterName: t.name}
	for _, e := range entries {
		if e.ID == t.entry.ID {
			out.Stages = e.Stages
		}
	}
	return out, nil
}

func (s *Service) DeviceSetRingCounts(ctx context.Context, d Device, stageNumber int, c repository.RingCounts) error {
	t, _, err := s.deviceTarget(ctx, d)
	if err != nil {
		return err
	}
	return s.SetRingCounts(ctx, d.CompetitionID, t.entry.ID, stageNumber, c)
}

func (s *Service) DeviceLockStage(ctx context.Context, d Device, stageNumber int) error {
	t, scorerName, err := s.deviceTarget(ctx, d)
	if err != nil {
		return err
	}
	stage, err := s.repo.GetStage(ctx, d.CompetitionID, t.entry.ID, stageNumber)
	if err != nil {
		return mapRepoErr(err)
	}
	if stage.Locked {
		return ErrStageLocked
	}
	if err := ValidateRingCounts(stage.Counts, stage.ExpectedShots); err != nil {
		return err
	}
	if err := s.repo.LockStage(ctx, stage.ID, d.Lane, scorerName); err != nil {
		if err == repository.ErrConflict {
			return ErrStageLocked
		}
		return mapRepoErr(err)
	}
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
	if err := s.repo.LockStageAsAdmin(ctx, stage.ID); err != nil {
		if err == repository.ErrConflict {
			return ErrStageLocked
		}
		return mapRepoErr(err)
	}
	return nil
}

func (s *Service) UnlockStage(ctx context.Context, competitionID, entryID int64, stageNumber int) error {
	stage, err := s.repo.GetStage(ctx, competitionID, entryID, stageNumber)
	if err != nil {
		return mapRepoErr(err)
	}
	return mapRepoErr(s.repo.UnlockStage(ctx, stage.ID))
}
