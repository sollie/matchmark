package service

import (
	"context"
	"time"
)

const pushTimeout = 5 * time.Minute

type Pusher interface {
	Push(ctx context.Context) error
}

type ReplicationRequester interface {
	Request()
}

type Replicator struct {
	pusher   Pusher
	interval time.Duration
	logf     func(format string, args ...any)
	requests chan struct{}
}

func NewReplicator(p Pusher, interval time.Duration, logf func(string, ...any)) *Replicator {
	return &Replicator{pusher: p, interval: interval, logf: logf, requests: make(chan struct{}, 1)}
}

func (r *Replicator) Request() {
	select {
	case r.requests <- struct{}{}:
	default:
	}
}

func (r *Replicator) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-r.requests:
		}
		r.push(ctx)
	}
}

func (r *Replicator) push(ctx context.Context) {
	pushCtx, cancel := context.WithTimeout(ctx, pushTimeout)
	defer cancel()
	if err := r.pusher.Push(pushCtx); err != nil && ctx.Err() == nil {
		r.logf("replication push failed, will retry: %v", err)
	}
}

type Option func(*Service)

func WithReplicator(r ReplicationRequester) Option {
	return func(s *Service) { s.replicator = r }
}
