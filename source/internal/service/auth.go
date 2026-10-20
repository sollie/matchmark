package service

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"

	"github.com/sollie/matchmark/internal/repository"
)

const (
	MinPasswordLength = 16
	MaxPasswordLength = 128
	SessionTTL        = 12 * time.Hour
	loginMaxFailures  = 5
	loginWindow       = 15 * time.Minute

	usernameWindow       = time.Hour
	usernameFreeFailures = 3
	usernameBaseDelay    = time.Second
	usernameMaxDelay     = 30 * time.Second
	usernameMaxTracked   = 10000

	argonMemory  = 19 * 1024
	argonTime    = 2
	argonThreads = 1
	argonKeyLen  = 32
	argonSaltLen = 16

	RoleAdmin = "admin"
	RoleUser  = "user"
)

var (
	ErrWeakPassword       = errors.New("password must be 16 to 128 characters")
	ErrInvalidUsername    = errors.New("username must be 3 to 32 characters from a-z, 0-9, '.', '_' and '-'")
	ErrSeedMissing        = errors.New("ADMIN_USERNAME and ADMIN_PASSWORD are required when there are no users")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthenticated    = errors.New("unauthenticated")
	ErrTooManyAttempts    = errors.New("too many login attempts")
	ErrLastAdmin          = errors.New("the last active admin cannot be demoted or deactivated")

	usernamePattern = regexp.MustCompile(`^[a-z0-9._-]{3,32}$`)
)

type User = repository.User

type Auth struct {
	repo      *repository.Repository
	now       func() time.Time
	dummyHash string

	adminMu  sync.Mutex
	mu       sync.Mutex
	failures map[string][]time.Time
	sleep    func(time.Duration)
	userFail map[string][]time.Time
}

type AuthOption func(*Auth)

func WithAuthClock(now func() time.Time) AuthOption {
	return func(a *Auth) { a.now = now }
}

func WithAuthSleep(sleep func(time.Duration)) AuthOption {
	return func(a *Auth) { a.sleep = sleep }
}

func NewAuth(repo *repository.Repository, opts ...AuthOption) *Auth {
	a := &Auth{repo: repo, now: time.Now, sleep: time.Sleep, failures: map[string][]time.Time{}, userFail: map[string][]time.Time{}}
	for _, opt := range opts {
		opt(a)
	}
	salt := make([]byte, argonSaltLen)
	_, _ = rand.Read(salt)
	a.dummyHash = encodeHash(salt, argonMemory, argonTime, argonThreads, derive("dummy", salt, argonMemory, argonTime, argonThreads))
	return a
}

func ValidPassword(password string) bool {
	n := utf8.RuneCountInString(password)
	return n >= MinPasswordLength && n <= MaxPasswordLength
}

func (a *Auth) SeedAdmin(username, password string) error {
	n, err := a.repo.CountUsers()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	if username == "" || password == "" {
		return ErrSeedMissing
	}
	username = strings.ToLower(username)
	if !usernamePattern.MatchString(username) {
		return ErrInvalidUsername
	}
	if !ValidPassword(password) {
		return ErrWeakPassword
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	_, err = a.repo.CreateUser(username, RoleAdmin, hash, true, false)
	return err
}

func (a *Auth) CreateUser(username, role, password string) (User, error) {
	username = strings.ToLower(username)
	if !usernamePattern.MatchString(username) {
		return User{}, ErrInvalidUsername
	}
	if role != RoleAdmin && role != RoleUser {
		return User{}, ErrInvalid
	}
	if !ValidPassword(password) {
		return User{}, ErrWeakPassword
	}
	hash, err := hashPassword(password)
	if err != nil {
		return User{}, err
	}
	if _, err := a.repo.CreateUser(username, role, hash, false, true); err != nil {
		return User{}, mapRepoErr(err)
	}
	return a.repo.UserByUsername(username)
}

func (a *Auth) ListUsers() ([]User, error) {
	return a.repo.ListUsers()
}

var ErrUnknownUser = errors.New("unknown user")

func (a *Auth) ResetPassword(username, password string) error {
	if !ValidPassword(password) {
		return ErrWeakPassword
	}
	user, err := a.repo.UserByUsername(strings.ToLower(username))
	if errors.Is(err, repository.ErrNotFound) {
		return ErrUnknownUser
	}
	if err != nil {
		return err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	return a.repo.ResetPassword(user.ID, hash)
}

func (a *Auth) lifecycleUser(id int) (User, error) {
	user, err := a.repo.UserByID(id)
	if errors.Is(err, repository.ErrNotFound) {
		return User{}, ErrNotFound
	}
	return user, err
}

func (a *Auth) guardLastAdmin(user User) error {
	if user.Role != RoleAdmin || !user.Active {
		return nil
	}
	n, err := a.repo.CountActiveAdmins()
	if err != nil {
		return err
	}
	if n <= 1 {
		return ErrLastAdmin
	}
	return nil
}

func (a *Auth) SetActive(id int, active bool) (User, error) {
	a.adminMu.Lock()
	defer a.adminMu.Unlock()
	user, err := a.lifecycleUser(id)
	if err != nil {
		return User{}, err
	}
	if !active {
		if err := a.guardLastAdmin(user); err != nil {
			return User{}, err
		}
	}
	if err := a.repo.SetActive(id, active); err != nil {
		return User{}, err
	}
	return a.repo.UserByID(id)
}

func (a *Auth) SetRole(id int, role string) (User, error) {
	if role != RoleAdmin && role != RoleUser {
		return User{}, ErrInvalid
	}
	a.adminMu.Lock()
	defer a.adminMu.Unlock()
	user, err := a.lifecycleUser(id)
	if err != nil {
		return User{}, err
	}
	if role == user.Role {
		return user, nil
	}
	if role == RoleUser {
		if err := a.guardLastAdmin(user); err != nil {
			return User{}, err
		}
	}
	if err := a.repo.SetRole(id, role); err != nil {
		return User{}, err
	}
	return a.repo.UserByID(id)
}

func (a *Auth) ResetUserPassword(id int, password string) error {
	user, err := a.lifecycleUser(id)
	if err != nil {
		return err
	}
	if !ValidPassword(password) {
		return ErrWeakPassword
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	return a.repo.ResetPassword(user.ID, hash)
}

func (a *Auth) ChangePassword(token, current, next string) error {
	user, err := a.Authenticate(token)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("user:%d", user.ID)
	now := a.now()
	if a.limited(key, now) {
		return ErrTooManyAttempts
	}
	if ok, _ := verifyPassword(current, user.PasswordHash); !ok {
		a.recordFailure(key, now)
		return ErrInvalidCredentials
	}
	if !ValidPassword(next) {
		return ErrWeakPassword
	}
	hash, err := hashPassword(next)
	if err != nil {
		return err
	}
	return a.repo.ChangePassword(user.ID, hash, hashToken(token))
}

func derive(password string, salt []byte, memory, time uint32, threads uint8) []byte {
	return argon2.IDKey([]byte(password), salt, time, memory, threads, argonKeyLen)
}

func encodeHash(salt []byte, memory, time uint32, threads uint8, key []byte) string {
	enc := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memory, time, threads, enc.EncodeToString(salt), enc.EncodeToString(key))
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	return encodeHash(salt, argonMemory, argonTime, argonThreads, derive(password, salt, argonMemory, argonTime, argonThreads)), nil
}

type parsedHash struct {
	memory, time uint32
	threads      uint8
	salt, key    []byte
}

func parseHash(phc string) (parsedHash, bool) {
	var p parsedHash
	parts := strings.Split(phc, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return p, false
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return p, false
	}
	var threads uint32
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.time, &threads); err != nil || threads == 0 || threads > 255 {
		return p, false
	}
	p.threads = uint8(threads)
	enc := base64.RawStdEncoding
	var err error
	if p.salt, err = enc.DecodeString(parts[4]); err != nil {
		return p, false
	}
	if p.key, err = enc.DecodeString(parts[5]); err != nil || len(p.key) == 0 {
		return p, false
	}
	return p, true
}

func verifyPassword(password, phc string) (ok, needsRehash bool) {
	p, valid := parseHash(phc)
	if !valid {
		return false, false
	}
	key := argon2.IDKey([]byte(password), p.salt, p.time, p.memory, p.threads, uint32(len(p.key)))
	ok = subtle.ConstantTimeCompare(key, p.key) == 1
	needsRehash = p.memory < argonMemory || p.time < argonTime || len(p.key) < argonKeyLen
	return ok, needsRehash
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (a *Auth) Login(clientKey, username, password string) (token string, expires time.Time, err error) {
	now := a.now()
	if a.limited(clientKey, now) {
		return "", time.Time{}, ErrTooManyAttempts
	}
	user, lookupErr := a.repo.UserByUsername(strings.ToLower(username))
	if lookupErr != nil && !errors.Is(lookupErr, repository.ErrNotFound) {
		return "", time.Time{}, lookupErr
	}
	known := lookupErr == nil
	hash := user.PasswordHash
	if !known {
		hash = a.dummyHash
	}
	ok, needsRehash := verifyPassword(password, hash)
	if !known || !ok || !user.Active {
		a.recordFailure(clientKey, now)
		if delay := a.recordUsernameFailure(strings.ToLower(username), now); delay > 0 {
			a.sleep(delay)
		}
		return "", time.Time{}, ErrInvalidCredentials
	}
	a.clearUsernameFailures(strings.ToLower(username))
	if needsRehash {
		upgraded, err := hashPassword(password)
		if err != nil {
			return "", time.Time{}, err
		}
		if err := a.repo.SetPasswordHash(user.Username, upgraded); err != nil {
			return "", time.Time{}, err
		}
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	expires = now.Add(SessionTTL)
	if err := a.repo.DeleteExpiredSessions(now); err != nil {
		return "", time.Time{}, err
	}
	if err := a.repo.CreateSession(hashToken(token), user.ID, expires); err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

func (a *Auth) Authenticate(token string) (User, error) {
	if token == "" {
		return User{}, ErrUnauthenticated
	}
	user, err := a.repo.SessionUser(hashToken(token), a.now())
	if errors.Is(err, repository.ErrNotFound) {
		return User{}, ErrUnauthenticated
	}
	return user, err
}

func (a *Auth) Logout(token string) error {
	if token == "" {
		return nil
	}
	return a.repo.DeleteSession(hashToken(token))
}

func (a *Auth) recent(clientKey string, now time.Time) []time.Time {
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

func (a *Auth) limited(clientKey string, now time.Time) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.recent(clientKey, now)) >= loginMaxFailures
}

func (a *Auth) recordFailure(clientKey string, now time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.failures[clientKey] = append(a.recent(clientKey, now), now)
}

func (a *Auth) recentUsername(username string, now time.Time) []time.Time {
	kept := a.userFail[username][:0]
	for _, t := range a.userFail[username] {
		if now.Sub(t) < usernameWindow {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(a.userFail, username)
	} else {
		a.userFail[username] = kept
	}
	return kept
}

func (a *Auth) recordUsernameFailure(username string, now time.Time) time.Duration {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, tracked := a.userFail[username]; !tracked && len(a.userFail) >= usernameMaxTracked {
		for name := range a.userFail {
			a.recentUsername(name, now)
		}
		if len(a.userFail) >= usernameMaxTracked {
			return usernameMaxDelay
		}
	}
	failures := append(a.recentUsername(username, now), now)
	a.userFail[username] = failures
	excess := len(failures) - usernameFreeFailures
	if excess <= 0 {
		return 0
	}
	if excess > 6 {
		excess = 6
	}
	return min(usernameBaseDelay<<(excess-1), usernameMaxDelay)
}

func (a *Auth) clearUsernameFailures(username string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.userFail, username)
}
