package service

import "sync"

type laneKey struct {
	matchID int64
	lane    int
}

type presence struct {
	mu    sync.Mutex
	conns map[laneKey]int
}

func (p *presence) connect(k laneKey) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conns == nil {
		p.conns = map[laneKey]int{}
	}
	p.conns[k]++
}

func (p *presence) disconnect(k laneKey) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conns[k] <= 1 {
		delete(p.conns, k)
		return
	}
	p.conns[k]--
}

func (p *presence) connected(k laneKey) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.conns[k] > 0
}

func (s *Service) DeviceConnected(d Device) (release func()) {
	k := laneKey{d.MatchID, d.Lane}
	s.presence.connect(k)
	s.bus.Publish(d.MatchID)
	return func() {
		s.presence.disconnect(k)
		s.bus.Publish(d.MatchID)
	}
}
