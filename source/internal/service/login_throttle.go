package service

import (
	"strings"
	"sync"
	"time"
)

const (
	loginMaxFailures = 5
	loginWindow      = 15 * time.Minute

	usernameWindow       = time.Hour
	usernameFreeFailures = 3
	usernameBaseDelay    = time.Second
	usernameMaxDelay     = 30 * time.Second
	usernameMaxTracked   = 10000

	loginAuditWindow     = loginWindow
	loginAuditMaxTracked = 10000
)

type LoginThrottle struct {
	mu        sync.Mutex
	clients   map[string][]time.Time
	usernames map[string][]time.Time
	audited   map[string]time.Time
}

func NewLoginThrottle() *LoginThrottle {
	return &LoginThrottle{
		clients:   map[string][]time.Time{},
		usernames: map[string][]time.Time{},
		audited:   map[string]time.Time{},
	}
}

func prune(m map[string][]time.Time, key string, now time.Time, window time.Duration) []time.Time {
	kept := m[key][:0]
	for _, t := range m[key] {
		if now.Sub(t) < window {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(m, key)
	} else {
		m[key] = kept
	}
	return kept
}

func (t *LoginThrottle) Allowed(client string, now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(prune(t.clients, client, now, loginWindow)) < loginMaxFailures
}

func (t *LoginThrottle) FailClient(client string, now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.clients[client] = append(prune(t.clients, client, now, loginWindow), now)
}

func (t *LoginThrottle) Fail(client, username string, now time.Time) (delay time.Duration, audit bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.clients[client] = append(prune(t.clients, client, now, loginWindow), now)
	delay = t.failUsername(username, now)
	return delay, t.shouldAudit(client, username, "", now)
}

func (t *LoginThrottle) Success(username string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.usernames, username)
}

func (t *LoginThrottle) ShouldAudit(client, username, details string, now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.shouldAudit(client, username, details, now)
}

func (t *LoginThrottle) failUsername(username string, now time.Time) time.Duration {
	if _, tracked := t.usernames[username]; !tracked && len(t.usernames) >= usernameMaxTracked {
		t.makeRoomForUsername(now)
	}
	failures := append(prune(t.usernames, username, now, usernameWindow), now)
	t.usernames[username] = failures
	excess := min(len(failures)-usernameFreeFailures, 6)
	if excess <= 0 {
		return 0
	}
	return min(usernameBaseDelay<<(excess-1), usernameMaxDelay)
}

func (t *LoginThrottle) makeRoomForUsername(now time.Time) {
	for name := range t.usernames {
		prune(t.usernames, name, now, usernameWindow)
	}
	if len(t.usernames) < usernameMaxTracked {
		return
	}
	var oldest string
	var oldestAt time.Time
	for name, failures := range t.usernames {
		if last := failures[len(failures)-1]; oldest == "" || last.Before(oldestAt) {
			oldest, oldestAt = name, last
		}
	}
	delete(t.usernames, oldest)
}

func (t *LoginThrottle) shouldAudit(client, username, details string, now time.Time) bool {
	key := details + "|" + client + "|" + strings.ToLower(username)
	if at, ok := t.audited[key]; ok && now.Sub(at) < loginAuditWindow {
		return false
	}
	if _, ok := t.audited[key]; !ok && len(t.audited) >= loginAuditMaxTracked {
		var oldest string
		var oldestAt time.Time
		for k, at := range t.audited {
			if now.Sub(at) >= loginAuditWindow {
				delete(t.audited, k)
			} else if oldest == "" || at.Before(oldestAt) {
				oldest, oldestAt = k, at
			}
		}
		if len(t.audited) >= loginAuditMaxTracked {
			delete(t.audited, oldest)
		}
	}
	t.audited[key] = now
	return true
}
