package service_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

type fakePusher struct {
	calls atomic.Int32
	fail  atomic.Bool
	block chan struct{}
}

func (f *fakePusher) Push(ctx context.Context) error {
	f.calls.Add(1)
	if f.block != nil {
		select {
		case <-f.block:
		case <-ctx.Done():
		}
	}
	if f.fail.Load() {
		return errors.New("sync server unreachable")
	}
	return nil
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func startReplicator(t *testing.T, p service.Pusher, interval time.Duration) *service.Replicator {
	t.Helper()
	r := service.NewReplicator(p, interval, func(string, ...any) {})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { r.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	return r
}

func TestReplicatorPushesOnTimer(t *testing.T) {
	p := &fakePusher{}
	startReplicator(t, p, 10*time.Millisecond)
	waitFor(t, func() bool { return p.calls.Load() >= 3 })
}

func TestReplicatorPushesOnRequest(t *testing.T) {
	p := &fakePusher{}
	r := startReplicator(t, p, time.Hour)
	r.Request()
	waitFor(t, func() bool { return p.calls.Load() == 1 })
}

func TestReplicatorRetriesAfterFailure(t *testing.T) {
	p := &fakePusher{}
	p.fail.Store(true)
	startReplicator(t, p, 10*time.Millisecond)
	waitFor(t, func() bool { return p.calls.Load() >= 2 })
	p.fail.Store(false)
	before := p.calls.Load()
	waitFor(t, func() bool { return p.calls.Load() > before+1 })
}

func TestReplicatorRequestNeverBlocks(t *testing.T) {
	p := &fakePusher{block: make(chan struct{})}
	r := startReplicator(t, p, time.Hour)
	r.Request()
	waitFor(t, func() bool { return p.calls.Load() == 1 })
	done := make(chan struct{})
	go func() {
		for i := 0; i < 10; i++ {
			r.Request()
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Request blocked while a push was in flight")
	}
	close(p.block)
}

type countRequester struct{ n atomic.Int32 }

func (c *countRequester) Request() { c.n.Add(1) }

func TestFinishingMatchRequestsReplication(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	rq := &countRequester{}
	svc := service.New(repository.New(conn), service.WithReplicator(rq))
	ctx := context.Background()
	id, err := svc.CreateMatch(ctx, repository.Match{Name: "Club match", Date: "2026-06-01", Course: "150"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SetMatchStatus(ctx, id, service.StatusActive); err != nil {
		t.Fatal(err)
	}
	if rq.n.Load() != 0 {
		t.Fatalf("requests before finish = %d", rq.n.Load())
	}
	if err := svc.SetMatchStatus(ctx, id, service.StatusFinished); err != nil {
		t.Fatal(err)
	}
	if rq.n.Load() != 1 {
		t.Fatalf("requests after finish = %d", rq.n.Load())
	}
}
