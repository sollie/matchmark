package service

import (
	"sync"
	"time"
)

const (
	defaultCoalesceWindow = 250 * time.Millisecond
	maxRetainedVersions   = 1024
)

type Broadcaster struct {
	window   time.Duration
	mu       sync.Mutex
	topics   map[int64]*topic
	retained map[int64]uint64
	global   uint64
}

type topic struct {
	subs    map[*Subscription]struct{}
	version uint64
	timer   *time.Timer
}

type Subscription struct {
	C      <-chan struct{}
	ch     chan struct{}
	b      *Broadcaster
	id     int64
	closed bool
}

func NewBroadcaster(window time.Duration) *Broadcaster {
	return &Broadcaster{window: window, topics: map[int64]*topic{}, retained: map[int64]uint64{}}
}

func (b *Broadcaster) topicLocked(id int64) *topic {
	t, ok := b.topics[id]
	if !ok {
		t = &topic{subs: map[*Subscription]struct{}{}, version: b.retained[id]}
		delete(b.retained, id)
		b.topics[id] = t
	}
	return t
}

func (b *Broadcaster) Subscribe(competitionID int64) *Subscription {
	ch := make(chan struct{}, 1)
	sub := &Subscription{C: ch, ch: ch, b: b, id: competitionID}
	b.mu.Lock()
	b.topicLocked(competitionID).subs[sub] = struct{}{}
	b.mu.Unlock()
	return sub
}

func (s *Subscription) Close() {
	b := s.b
	b.mu.Lock()
	defer b.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	t := b.topics[s.id]
	delete(t.subs, s)
	b.releaseLocked(s.id, t)
}

func (b *Broadcaster) releaseLocked(id int64, t *topic) {
	if len(t.subs) > 0 || t.timer != nil {
		return
	}
	delete(b.topics, id)
	if t.version == 0 {
		return
	}
	b.retained[id] = t.version
	if len(b.retained) > maxRetainedVersions {
		for evicted, v := range b.retained {
			if evicted != id {
				delete(b.retained, evicted)
				b.global += v
				break
			}
		}
	}
}

func (b *Broadcaster) Version(competitionID int64) uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	if t, ok := b.topics[competitionID]; ok {
		return b.global + t.version
	}
	return b.global + b.retained[competitionID]
}

func (b *Broadcaster) Publish(competitionID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.publishLocked(competitionID)
}

func (b *Broadcaster) PublishAll() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.global++
	for id := range b.topics {
		b.scheduleLocked(id)
	}
}

func (b *Broadcaster) publishLocked(id int64) {
	b.topicLocked(id).version++
	b.scheduleLocked(id)
}

func (b *Broadcaster) scheduleLocked(id int64) {
	t := b.topicLocked(id)
	if t.timer != nil {
		return
	}
	t.timer = time.AfterFunc(b.window, func() { b.fire(id) })
}

func (b *Broadcaster) fire(id int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	t := b.topics[id]
	t.timer = nil
	for sub := range t.subs {
		select {
		case sub.ch <- struct{}{}:
		default:
		}
	}
	b.releaseLocked(id, t)
}
