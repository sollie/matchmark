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

func mapAcceptanceErr(err error) error {
	if errors.Is(err, repository.ErrConflict) {
		return ErrStageNotAwaitingAcceptance
	}
	return mapRepoErr(err)
}

func (s *Service) DeviceSnapshot(ctx context.Context, token string) (ScorerSession, error) {
	d, err := s.AuthenticateDevice(ctx, token)
	if err != nil {
		return ScorerSession{}, err
	}
	return s.DeviceSession(ctx, d)
}
