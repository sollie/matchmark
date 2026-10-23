package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/sollie/matchmark/internal/wa1500"
)

const (
	wa1500CacheTTL      = 5 * time.Minute
	wa1500CacheMax      = 200
	wa1500RateWindow    = time.Minute
	wa1500RateMax       = 10
	wa1500MaxQueryRunes = 100
)

var (
	ErrUpstream    = errors.New("wa1500 upstream failure")
	ErrRateLimited = errors.New("too many wa1500 searches")
)

type WA1500Lookup struct {
	searcher wa1500.Searcher
	now      func() time.Time

	mu    sync.Mutex
	cache map[string]wa1500Cached
	calls []time.Time
}

type wa1500Cached struct {
	results []wa1500.Shooter
	expires time.Time
}

func NewWA1500Lookup(searcher wa1500.Searcher) *WA1500Lookup {
	return &WA1500Lookup{searcher: searcher, now: time.Now, cache: map[string]wa1500Cached{}}
}

func (l *WA1500Lookup) Search(ctx context.Context, query string) ([]wa1500.Shooter, error) {
	query = strings.Join(strings.Fields(query), " ")
	if query == "" || len([]rune(query)) > wa1500MaxQueryRunes {
		return nil, ErrInvalid
	}
	if l == nil || l.searcher == nil {
		return nil, ErrUpstream
	}
	key := strings.ToLower(query)
	if results, ok := l.cached(key); ok {
		return results, nil
	}
	if !l.allow() {
		return nil, ErrRateLimited
	}
	results, err := l.searcher.Search(ctx, query)
	if err != nil {
		return nil, errors.Join(ErrUpstream, err)
	}
	if results == nil {
		results = []wa1500.Shooter{}
	}
	l.store(key, results)
	return results, nil
}

func (l *WA1500Lookup) cached(key string) ([]wa1500.Shooter, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry, ok := l.cache[key]
	if !ok || !l.now().Before(entry.expires) {
		delete(l.cache, key)
		return nil, false
	}
	return entry.results, true
}

func (l *WA1500Lookup) store(key string, results []wa1500.Shooter) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if len(l.cache) >= wa1500CacheMax {
		for k, e := range l.cache {
			if !now.Before(e.expires) {
				delete(l.cache, k)
			}
		}
		if len(l.cache) >= wa1500CacheMax {
			clear(l.cache)
		}
	}
	l.cache[key] = wa1500Cached{results: results, expires: now.Add(wa1500CacheTTL)}
}

func (l *WA1500Lookup) allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	kept := l.calls[:0]
	for _, t := range l.calls {
		if now.Sub(t) < wa1500RateWindow {
			kept = append(kept, t)
		}
	}
	l.calls = kept
	if len(l.calls) >= wa1500RateMax {
		return false
	}
	l.calls = append(l.calls, now)
	return true
}
