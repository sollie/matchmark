package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

func TestStageTransitionMatrix(t *testing.T) {
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
		run  func(f *scoreFixture, action service.StageAction) (repository.StageStatus, error)
	}
	device := func(d service.Device) func(f *scoreFixture, action service.StageAction) (repository.StageStatus, error) {
		return func(f *scoreFixture, action service.StageAction) (repository.StageStatus, error) {
			d.MatchID = f.compID
			switch action {
			case service.ActionLock:
				return f.svc.DeviceLockStage(context.Background(), d, 1)
			case service.ActionAccept:
				return f.svc.DeviceAcceptStage(context.Background(), d, 1)
			case service.ActionDispute:
				return f.svc.DeviceDisputeStage(context.Background(), d, 1)
			}
			return "", service.ErrForbidden
		}
	}
	actors := []actor{
		{"scorer device", device(scorer)},
		{"shooter device", device(shooter)},
		{"admin", func(f *scoreFixture, action service.StageAction) (repository.StageStatus, error) {
			switch action {
			case service.ActionLock:
				return f.svc.AdminLockStage(admin, f.compID, f.entry2, 1)
			case service.ActionAccept:
				return f.svc.AdminAcceptStage(admin, f.compID, f.entry2, 1)
			case service.ActionUnlock:
				return f.svc.UnlockStage(admin, f.compID, f.entry2, 1)
			}
			return "", service.ErrForbidden
		}},
	}

	type want struct {
		status repository.StageStatus
		err    error
	}
	ok := func(s repository.StageStatus) want { return want{status: s} }
	bad := func(e error) want { return want{err: e} }
	allStates := func(w want) map[string]want { return map[string]want{open: w, awaiting: w, accepted: w} }
	matrix := map[string]map[service.StageAction]map[string]want{
		"scorer device": {
			service.ActionLock:    {open: ok(repository.StatusAwaitingAcceptance), awaiting: bad(service.ErrStageLocked), accepted: bad(service.ErrStageLocked)},
			service.ActionAccept:  allStates(bad(service.ErrStageNotAwaitingAcceptance)),
			service.ActionDispute: allStates(bad(service.ErrStageNotAwaitingAcceptance)),
			service.ActionUnlock:  allStates(bad(service.ErrForbidden)),
		},
		"shooter device": {
			service.ActionLock:    allStates(bad(service.ErrInvalid)),
			service.ActionAccept:  {open: bad(service.ErrStageNotAwaitingAcceptance), awaiting: ok(repository.StatusAccepted), accepted: bad(service.ErrStageNotAwaitingAcceptance)},
			service.ActionDispute: {open: bad(service.ErrStageNotAwaitingAcceptance), awaiting: ok(repository.StatusOpen), accepted: bad(service.ErrStageNotAwaitingAcceptance)},
			service.ActionUnlock:  allStates(bad(service.ErrForbidden)),
		},
		"admin": {
			service.ActionLock:    {open: ok(repository.StatusAccepted), awaiting: bad(service.ErrStageLocked), accepted: bad(service.ErrStageLocked)},
			service.ActionAccept:  {open: bad(service.ErrStageNotAwaitingAcceptance), awaiting: ok(repository.StatusAccepted), accepted: bad(service.ErrStageNotAwaitingAcceptance)},
			service.ActionDispute: allStates(bad(service.ErrForbidden)),
			service.ActionUnlock:  {open: bad(service.ErrStageNotLocked), awaiting: ok(repository.StatusOpen), accepted: ok(repository.StatusOpen)},
		},
	}

	prepare := func(t *testing.T, state string) *scoreFixture {
		f := newScoreFixture(t)
		f.fillStage(t, f.entry2, 1)
		if state == open {
			return f
		}
		if _, err := f.svc.DeviceLockStage(context.Background(), service.Device{MatchID: f.compID, Lane: 1}, 1); err != nil {
			t.Fatal(err)
		}
		if state == accepted {
			if _, err := f.svc.AdminAcceptStage(admin, f.compID, f.entry2, 1); err != nil {
				t.Fatal(err)
			}
		}
		return f
	}

	actions := []service.StageAction{service.ActionLock, service.ActionAccept, service.ActionDispute, service.ActionUnlock}
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
