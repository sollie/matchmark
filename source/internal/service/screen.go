package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/sollie/matchmark/internal/repository"
)

const (
	MaxScreens        = 50
	ScreenOnlineAfter = 2 * time.Minute
	ScreenStaleAfter  = 24 * time.Hour
	screenSweepEvery  = time.Hour
	screenCodeLength  = 6
	screenCodeAlpha   = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	maxScreenNameLen  = 60
)

var (
	ErrTooManyRegistrations = errors.New("too many screen registrations")
	ErrScreenLimit          = errors.New("screen limit reached")
)

type ScreenRegistration struct {
	Token string
	Code  string
}

type Screen struct {
	ID         int64
	Name       string
	MatchID    *int64
	Online     bool
	LastSeenAt time.Time
}

type ScreenUpdate struct {
	Name     *string
	SetMatch bool
	MatchID  *int64
}

func newScreenCode() (string, error) {
	raw := make([]byte, screenCodeLength)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	code := make([]byte, screenCodeLength)
	for i, b := range raw {
		code[i] = screenCodeAlpha[int(b)%len(screenCodeAlpha)]
	}
	return string(code), nil
}

func (s *Service) toScreen(r repository.Screen, now time.Time) Screen {
	return Screen{ID: r.ID, Name: r.Name, MatchID: r.MatchID, Online: s.screenConnected(r.ID) || now.Sub(r.LastSeenAt) < ScreenOnlineAfter, LastSeenAt: r.LastSeenAt}
}

func (s *Service) RegisterScreen(ctx context.Context, client string) (ScreenRegistration, error) {
	now := time.Now()
	if !s.screenThrottle.Allowed(client, now) {
		return ScreenRegistration{}, ErrTooManyRegistrations
	}
	s.screenThrottle.FailClient(client, now)
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return ScreenRegistration{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	for range 5 {
		code, err := newScreenCode()
		if err != nil {
			return ScreenRegistration{}, err
		}
		switch err := s.repo.CreateScreen(ctx, hashToken(token), code, now, MaxScreens); {
		case err == nil:
			return ScreenRegistration{Token: token, Code: code}, nil
		case errors.Is(err, repository.ErrLimit):
			return ScreenRegistration{}, ErrScreenLimit
		case errors.Is(err, repository.ErrConflict):
		default:
			return ScreenRegistration{}, err
		}
	}
	return ScreenRegistration{}, ErrConflict
}

func (s *Service) AuthenticateScreen(ctx context.Context, token string) (repository.Screen, error) {
	if token == "" {
		return repository.Screen{}, ErrUnauthenticated
	}
	sc, err := s.repo.ScreenByTokenHash(ctx, hashToken(token))
	if errors.Is(err, repository.ErrNotFound) {
		return repository.Screen{}, ErrUnauthenticated
	}
	if err != nil {
		return repository.Screen{}, err
	}
	if err := s.repo.TouchScreen(ctx, sc.ID, time.Now()); err != nil {
		return repository.Screen{}, err
	}
	return sc, nil
}

func (s *Service) LeaveScreen(ctx context.Context, token string) error {
	sc, err := s.AuthenticateScreen(ctx, token)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteScreen(ctx, sc.ID); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Screens: true})
	return nil
}

func (s *Service) ListScreens(ctx context.Context) ([]Screen, error) {
	rows, err := s.repo.ListApprovedScreens(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := make([]Screen, 0, len(rows))
	for _, r := range rows {
		out = append(out, s.toScreen(r, now))
	}
	return out, nil
}

func (s *Service) ApproveScreen(ctx context.Context, code, name string) (Screen, error) {
	name = strings.TrimSpace(name)
	if len(name) > maxScreenNameLen {
		return Screen{}, ErrInvalid
	}
	id, err := s.repo.ApproveScreenByCode(ctx, strings.ToUpper(strings.TrimSpace(code)), name, time.Now())
	if err != nil {
		return Screen{}, mapRepoErr(err)
	}
	sc, err := s.repo.GetScreen(ctx, id)
	if err != nil {
		return Screen{}, mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "screen.approve", Target: idTarget("screen", id)}, Screens: true})
	return s.toScreen(sc, time.Now()), nil
}

func (s *Service) UpdateScreen(ctx context.Context, id int64, u ScreenUpdate) (Screen, error) {
	if u.Name != nil {
		name := strings.TrimSpace(*u.Name)
		if name == "" || len(name) > maxScreenNameLen {
			return Screen{}, ErrInvalid
		}
		u.Name = &name
	}
	if _, err := s.repo.GetScreen(ctx, id); err != nil {
		return Screen{}, mapRepoErr(err)
	}
	if u.SetMatch && u.MatchID != nil {
		if err := s.checkAssignable(ctx, *u.MatchID); err != nil {
			return Screen{}, err
		}
	}
	if u.Name != nil {
		if err := s.repo.RenameScreen(ctx, id, *u.Name); err != nil {
			return Screen{}, mapRepoErr(err)
		}
	}
	if u.SetMatch {
		if err := s.repo.AssignScreen(ctx, id, u.MatchID); err != nil {
			return Screen{}, mapRepoErr(err)
		}
	}
	sc, err := s.repo.GetScreen(ctx, id)
	if err != nil {
		return Screen{}, mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "screen.update", Target: idTarget("screen", id)}, Screens: true})
	return s.toScreen(sc, time.Now()), nil
}

func (s *Service) checkAssignable(ctx context.Context, matchID int64) error {
	m, err := s.repo.GetMatch(ctx, matchID)
	if err != nil {
		return mapRepoErr(err)
	}
	if MatchStatus(m.Status) == StatusFinished {
		return ErrConflict
	}
	return nil
}

func (s *Service) AssignAllScreens(ctx context.Context, matchID int64) (int64, error) {
	if err := s.checkAssignable(ctx, matchID); err != nil {
		return 0, err
	}
	n, err := s.repo.AssignAllScreens(ctx, matchID)
	if err != nil {
		return 0, mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "screen.assign_all", Target: idTarget("match", matchID)}, Screens: true})
	return n, nil
}

func (s *Service) RemoveScreen(ctx context.Context, id int64) error {
	if err := s.repo.DeleteScreen(ctx, id); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "screen.remove", Target: idTarget("screen", id)}, Screens: true})
	return nil
}

func (s *Service) AssignableMatches(ctx context.Context) ([]repository.Match, error) {
	return s.repo.ListAssignableMatches(ctx)
}

func (s *Service) PurgeStaleScreens(ctx context.Context, now time.Time) (int64, error) {
	n, err := s.repo.PurgeScreensSeenBefore(ctx, now.Add(-ScreenStaleAfter))
	if err == nil && n > 0 {
		s.changes.Record(ctx, Change{Screens: true})
	}
	return n, err
}

func (s *Service) RunScreenSweep(ctx context.Context) {
	ticker := time.NewTicker(screenSweepEvery)
	defer ticker.Stop()
	for {
		_, _ = s.PurgeStaleScreens(ctx, time.Now())
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

const screenTopic int64 = -1

type ScreenState struct {
	State   string
	MatchID int64
}

func (s *Service) screenConnected(id int64) bool {
	s.screenMu.Lock()
	defer s.screenMu.Unlock()
	return s.screenConns[id] > 0
}

func (s *Service) ScreenConnected(id int64) func() {
	s.screenMu.Lock()
	s.screenConns[id]++
	s.screenMu.Unlock()
	return func() {
		s.screenMu.Lock()
		s.screenConns[id]--
		if s.screenConns[id] <= 0 {
			delete(s.screenConns, id)
		}
		s.screenMu.Unlock()
		_ = s.repo.TouchScreen(context.Background(), id, time.Now())
	}
}

func (s *Service) SubscribeScreens() *Subscription {
	return s.bus.Subscribe(screenTopic)
}

func (s *Service) ScreenState(ctx context.Context, sc repository.Screen) (ScreenState, error) {
	cur, err := s.repo.GetScreen(ctx, sc.ID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ScreenState{}, ErrUnauthenticated
		}
		return ScreenState{}, err
	}
	if cur.MatchID != nil {
		m, err := s.repo.GetMatch(ctx, *cur.MatchID)
		switch {
		case err == nil && MatchStatus(m.Status) == StatusDraft:
			return ScreenState{State: "waiting"}, nil
		case err == nil:
			return ScreenState{State: "match", MatchID: m.ID}, nil
		case !errors.Is(err, repository.ErrNotFound):
			return ScreenState{}, err
		}
	}
	id, err := s.repo.DefaultMatchID(ctx)
	if errors.Is(err, repository.ErrNotFound) {
		return ScreenState{State: "waiting"}, nil
	}
	if err != nil {
		return ScreenState{}, err
	}
	return ScreenState{State: "match", MatchID: id}, nil
}
