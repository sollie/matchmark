package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

func TestSegmentTransitionMatrix(t *testing.T) {
	const (
		open     = "open"
		awaiting = "awaiting"
		accepted = "accepted"
	)
	admin := service.WithActor(context.Background(), service.Actor{UserID: 1, Username: "carol"})
	scorer := service.Device{Lane: 1}
	shooter := service.Device{Lane: 2}

	type actor struct {
		name string
		run  func(f *scoreFixture, action service.SegmentAction) (repository.SegmentStatus, error)
	}
	device := func(d service.Device) func(f *scoreFixture, action service.SegmentAction) (repository.SegmentStatus, error) {
		return func(f *scoreFixture, action service.SegmentAction) (repository.SegmentStatus, error) {
			d.MatchID = f.compID
			switch action {
			case service.ActionLock:
				return f.svc.DeviceLockSegment(context.Background(), d, 1)
			case service.ActionAccept:
				return f.svc.DeviceAcceptSegment(context.Background(), d, 1)
			case service.ActionDispute:
				return f.svc.DeviceDisputeSegment(context.Background(), d, 1)
			}
			return "", service.ErrForbidden
		}
	}
	actors := []actor{
		{"scorer device", device(scorer)},
		{"shooter device", device(shooter)},
		{"admin", func(f *scoreFixture, action service.SegmentAction) (repository.SegmentStatus, error) {
			switch action {
			case service.ActionLock:
				return f.svc.AdminLockSegment(admin, f.compID, f.entry2, 1)
			case service.ActionAccept:
				return f.svc.AdminAcceptSegment(admin, f.compID, f.entry2, 1)
			case service.ActionUnlock:
				return f.svc.UnlockSegment(admin, f.compID, f.entry2, 1)
			}
			return "", service.ErrForbidden
		}},
	}

	type want struct {
		status repository.SegmentStatus
		err    error
	}
	ok := func(s repository.SegmentStatus) want { return want{status: s} }
	bad := func(e error) want { return want{err: e} }
	allStates := func(w want) map[string]want { return map[string]want{open: w, awaiting: w, accepted: w} }
	matrix := map[string]map[service.SegmentAction]map[string]want{
		"scorer device": {
			service.ActionLock:    {open: ok(repository.StatusAwaitingAcceptance), awaiting: bad(service.ErrSegmentLocked), accepted: bad(service.ErrSegmentLocked)},
			service.ActionAccept:  allStates(bad(service.ErrSegmentNotAwaitingAcceptance)),
			service.ActionDispute: allStates(bad(service.ErrSegmentNotAwaitingAcceptance)),
			service.ActionUnlock:  allStates(bad(service.ErrForbidden)),
		},
		"shooter device": {
			service.ActionLock:    allStates(bad(service.ErrInvalid)),
			service.ActionAccept:  {open: bad(service.ErrSegmentNotAwaitingAcceptance), awaiting: ok(repository.StatusAccepted), accepted: bad(service.ErrSegmentNotAwaitingAcceptance)},
			service.ActionDispute: {open: bad(service.ErrSegmentNotAwaitingAcceptance), awaiting: ok(repository.StatusOpen), accepted: bad(service.ErrSegmentNotAwaitingAcceptance)},
			service.ActionUnlock:  allStates(bad(service.ErrForbidden)),
		},
		"admin": {
			service.ActionLock:    {open: ok(repository.StatusAccepted), awaiting: bad(service.ErrSegmentLocked), accepted: bad(service.ErrSegmentLocked)},
			service.ActionAccept:  {open: bad(service.ErrSegmentNotAwaitingAcceptance), awaiting: ok(repository.StatusAccepted), accepted: bad(service.ErrSegmentNotAwaitingAcceptance)},
			service.ActionDispute: allStates(bad(service.ErrForbidden)),
			service.ActionUnlock:  {open: bad(service.ErrSegmentNotLocked), awaiting: ok(repository.StatusOpen), accepted: ok(repository.StatusOpen)},
		},
	}

	prepare := func(t *testing.T, state string) *scoreFixture {
		f := newScoreFixture(t)
		f.fillSegment(t, f.entry2, 1)
		if state == open {
			return f
		}
		if _, err := f.svc.DeviceLockSegment(context.Background(), service.Device{MatchID: f.compID, Lane: 1}, 1); err != nil {
			t.Fatal(err)
		}
		if state == accepted {
			if _, err := f.svc.AdminAcceptSegment(admin, f.compID, f.entry2, 1); err != nil {
				t.Fatal(err)
			}
		}
		return f
	}

	actions := []service.SegmentAction{service.ActionLock, service.ActionAccept, service.ActionDispute, service.ActionUnlock}
	for _, a := range actors {
		for _, action := range actions {
			for _, state := range []string{open, awaiting, accepted} {
				t.Run(a.name+"/"+string(action)+"/"+state, func(t *testing.T) {
					w := matrix[a.name][action][state]
					got, err := a.run(prepare(t, state), action)
					if !errors.Is(err, w.err) || (w.err == nil && err != nil) {
						t.Fatalf("err = %v, want %v", err, w.err)
					}
					if got != w.status {
						t.Fatalf("status = %q, want %q", got, w.status)
					}
				})
			}
		}
	}
}
