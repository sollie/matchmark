package service

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"

	"github.com/sollie/matchmark/internal/repository"
)

const (
	MinAdminPasswordLength = 16
	SessionTTL             = 12 * time.Hour
	loginMaxFailures       = 5
	loginWindow            = 15 * time.Minute
)

var (
	ErrWeakPassword       = errors.New("admin password must be at least 16 characters")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthenticated    = errors.New("unauthenticated")
	ErrTooManyAttempts    = errors.New("too many login attempts")
)

type AdminAuth struct {
	repo *repository.Repository
	salt []byte
	hash []byte
	now  func() time.Time

	mu       sync.Mutex
	failures map[string][]time.Time
}

func NewAdminAuth(repo *repository.Repository, password string) (*AdminAuth, error) {
	if len(password) < MinAdminPasswordLength {
		return nil, ErrWeakPassword
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	return &AdminAuth{
		repo:     repo,
		salt:     salt,
		hash:     hashPassword(password, salt),
		now:      time.Now,
		failures: map[string][]time.Time{},
	}, nil
}

func hashPassword(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (a *AdminAuth) Login(clientKey, password string) (token string, expires time.Time, err error) {
	now := a.now()
	if a.limited(clientKey, now) {
		return "", time.Time{}, ErrTooManyAttempts
	}
	if subtle.ConstantTimeCompare(hashPassword(password, a.salt), a.hash) != 1 {
		a.recordFailure(clientKey, now)
		return "", time.Time{}, ErrInvalidCredentials
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	expires = now.Add(SessionTTL)
	if err := a.repo.DeleteExpiredAdminSessions(now); err != nil {
		return "", time.Time{}, err
	}
	if err := a.repo.CreateAdminSession(hashToken(token), expires); err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

func (a *AdminAuth) Authenticate(token string) error {
	if token == "" {
		return ErrUnauthenticated
	}
	ok, err := a.repo.AdminSessionValid(hashToken(token), a.now())
	if err != nil {
		return err
	}
	if !ok {
		return ErrUnauthenticated
	}
	return nil
}

func (a *AdminAuth) Logout(token string) error {
	if token == "" {
		return nil
	}
	return a.repo.DeleteAdminSession(hashToken(token))
}

func (a *AdminAuth) recent(clientKey string, now time.Time) []time.Time {
	kept := a.failures[clientKey][:0]
	for _, t := range a.failures[clientKey] {
		if now.Sub(t) < loginWindow {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(a.failures, clientKey)
	} else {
		a.failures[clientKey] = kept
	}
	return kept
}

func (a *AdminAuth) limited(clientKey string, now time.Time) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.recent(clientKey, now)) >= loginMaxFailures
}

func (a *AdminAuth) recordFailure(clientKey string, now time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.failures[clientKey] = append(a.recent(clientKey, now), now)
}
