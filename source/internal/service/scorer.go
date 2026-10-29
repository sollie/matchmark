package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strconv"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
)

type Device struct {
	MatchID  int64
	Lane     int
	Finished bool
}

type ScorerSession struct {
	MatchID        int64
	Lane           int
	CurrentSegment int
	ScoreLane      int
	EntryID        int64
	ShooterName    string
	Segments       []repository.SegmentInfo
	Names          map[int]string
	Phase          Phase
	Acceptance     *Acceptance
}

type Phase string

const (
	PhaseScoring         Phase = "scoring"
	PhaseWaiting         Phase = "waiting"
	PhaseNeedsAcceptance Phase = "needs_acceptance"
	PhaseNextSegment     Phase = "next_segment"
	PhaseFinished        Phase = "finished"
	PhaseNotStarted      Phase = "not_started"
)

type Acceptance struct {
	SegmentNumber int
	Counts        repository.RingCounts
	Total         int
	ScorerName    string
}

func segmentOf(e repository.EntryWithSegments, number int) (repository.SegmentInfo, bool) {
	for _, st := range e.Segments {
		if st.Number == number {
			return st, true
		}
	}
	return repository.SegmentInfo{}, false
}

func derivePhase(t ScoreTarget) (Phase, *Acceptance) {
	if st, ok := segmentOf(t.Entry, t.CurrentSegment); ok && st.Status == repository.StatusOpen {
		if st.Started {
			return PhaseNextSegment, nil
		}
		return PhaseScoring, nil
	}
	if st, ok := segmentOf(t.Own, t.CurrentSegment); ok && st.Status == repository.StatusAwaitingAcceptance {
		c := st.Counts
		name := ""
		if st.ScorerName != nil {
			name = *st.ScorerName
		}
		return PhaseNeedsAcceptance, &Acceptance{
			SegmentNumber: st.Number, Counts: c, ScorerName: name,
			Total: c.X + c.Ten + c.Nine + c.Eight + c.Seven + c.Zero + c.Miss,
		}
	}
	return PhaseWaiting, nil
}

func (s *Service) CreatePairing(ctx context.Context, matchID int64, lane int) (int64, string, error) {
	comp, err := s.repo.GetMatch(ctx, matchID)
	if err != nil {
		return 0, "", mapRepoErr(err)
	}
	if MatchStatus(comp.Status) == StatusFinished {
		return 0, "", ErrConflict
	}
	if _, err := s.repo.MannedEntryByLane(ctx, matchID, lane); err != nil {
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
	id, err := s.repo.CreatePairing(ctx, matchID, lane, hashToken(token))
	if err != nil {
		return 0, "", mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "device.pair", Target: idTarget("match", matchID), Details: "lane " + strconv.Itoa(lane)}, MatchID: matchID, Live: LiveMatch})
	return id, token, nil
}

func (s *Service) ListPairings(ctx context.Context, matchID int64) ([]repository.Pairing, error) {
	if _, err := s.repo.GetMatch(ctx, matchID); err != nil {
		return nil, mapRepoErr(err)
	}
	pairings, err := s.repo.ListPairings(ctx, matchID)
	return pairings, mapRepoErr(err)
}

func (s *Service) RevokePairing(ctx context.Context, matchID, pairingID int64) error {
	if err := s.repo.RevokePairing(ctx, matchID, pairingID); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "device.revoke", Target: idTarget("match", matchID), Details: "pairing " + strconv.FormatInt(pairingID, 10)}, MatchID: matchID, Live: LiveMatch})
	return nil
}

func (s *Service) AuthenticateDevice(ctx context.Context, token string) (Device, error) {
	p, err := s.repo.PairingByTokenHash(ctx, hashToken(token))
	if err != nil {
		return Device{}, ErrUnauthenticated
	}
	if p.Revoked {
		comp, err := s.repo.GetMatch(ctx, p.MatchID)
		if err != nil || MatchStatus(comp.Status) != StatusFinished {
			return Device{}, ErrUnauthenticated
		}
		return Device{MatchID: p.MatchID, Lane: p.Lane, Finished: true}, nil
	}
	return Device{MatchID: p.MatchID, Lane: p.Lane}, nil
}

func shooterName(sh repository.Shooter) string {
	return sh.FirstName + " " + sh.LastName
}

func (s *Service) DeviceSession(ctx context.Context, d Device) (ScorerSession, error) {
	if d.Finished {
		return ScorerSession{MatchID: d.MatchID, Lane: d.Lane, Phase: PhaseFinished}, nil
	}
	comp, err := s.repo.GetMatch(ctx, d.MatchID)
	if err != nil {
		return ScorerSession{}, mapRepoErr(err)
	}
	if MatchStatus(comp.Status) == StatusDraft {
		return ScorerSession{MatchID: d.MatchID, Lane: d.Lane, Phase: PhaseNotStarted}, nil
	}
	t, err := s.ScoreTarget(ctx, d)
	if err != nil {
		return ScorerSession{}, err
	}
	phase, acceptance := derivePhase(t)
	return ScorerSession{MatchID: d.MatchID, Lane: d.Lane, CurrentSegment: t.CurrentSegment, ScoreLane: t.ScoreLane, EntryID: t.Entry.ID, ShooterName: t.ShooterName, Segments: t.Entry.Segments, Names: segmentNames(comp.Course), Phase: phase, Acceptance: acceptance}, nil
}

func segmentNames(courseKey string) map[int]string {
	course, err := rulebook.CourseByKey(courseKey)
	if err != nil {
		return nil
	}
	names := map[int]string{}
	for _, st := range course.Segments {
		names[st.Number] = st.Name
	}
	return names
}

func (s *Service) DeviceSetRingCounts(ctx context.Context, d Device, segmentNumber int, c repository.RingCounts) error {
	t, err := s.ScoreTarget(ctx, d)
	if err != nil {
		return err
	}
	if segmentNumber != t.CurrentSegment {
		return ErrNotCurrentSegment
	}
	if err := s.writeRingCounts(ctx, d.MatchID, t.Entry.ID, segmentNumber, c, true); err != nil {
		return err
	}
	s.changes.Record(ctx, Change{
		Audit: AuditEvent{
			Action:  "segment.set_counts",
			Target:  segmentTarget(d.MatchID, t.Entry.ID, segmentNumber),
			Details: "lane " + strconv.Itoa(d.Lane) + " match " + strconv.FormatInt(d.MatchID, 10),
		},
		Actor:   &Actor{Username: DeviceActor, IP: ActorFrom(ctx).IP},
		MatchID: d.MatchID,
		Live:    LiveMatch,
	})
	return nil
}

func mapAcceptanceErr(err error) error {
	if errors.Is(err, repository.ErrConflict) {
		return ErrSegmentNotAwaitingAcceptance
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
