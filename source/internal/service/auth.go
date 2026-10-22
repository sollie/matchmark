package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

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

	loginAuditWindow     = loginWindow
	loginAuditMaxTracked = 10000

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
	ErrBackupNotPermitted = errors.New("backup download is not permitted for this admin")
	ErrForbidden          = errors.New("forbidden")
	ErrLastAdmin          = errors.New("the last active admin cannot be demoted or deactivated")

	usernamePattern = regexp.MustCompile(`^[a-z0-9._-]{3,32}$`)
)

type User = repository.User

type Auth struct {
	repo      *repository.Repository
	now       func() time.Time
	dummyHash string
	audit     *Audit

	adminMu  sync.Mutex
	mu       sync.Mutex
	failures map[string][]time.Time
	sleep    func(time.Duration)
	userFail map[string][]time.Time

	auditMu   sync.Mutex
	auditSeen map[string]time.Time
}

type AuthOption func(*Auth)

func WithAuthClock(now func() time.Time) AuthOption {
	return func(a *Auth) { a.now = now }
}

func WithAuthSleep(sleep func(time.Duration)) AuthOption {
	return func(a *Auth) { a.sleep = sleep }
}

func WithAuthAudit(audit *Audit) AuthOption {
	return func(a *Auth) { a.audit = audit }
}

func NewAuth(repo *repository.Repository, opts ...AuthOption) *Auth {
	a := &Auth{repo: repo, now: time.Now, sleep: time.Sleep, failures: map[string][]time.Time{}, userFail: map[string][]time.Time{}, auditSeen: map[string]time.Time{}}
	for _, opt := range opts {
		opt(a)
	}
	a.dummyHash, _ = hashPassword("dummy")
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

func (a *Auth) CreateUser(ctx context.Context, username, role, password string) (User, error) {
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
	id, err := a.repo.CreateUser(username, role, hash, false, true)
	if err != nil {
		return User{}, mapRepoErr(err)
	}
	a.audit.Record(ctx, AuditEvent{Action: "user.create", Target: userTarget(id), Details: username + " " + role})
	return a.repo.UserByUsername(username)
}

func (a *Auth) ListUsers() ([]User, error) {
	return a.repo.ListUsers()
}

var ErrUnknownUser = errors.New("unknown user")

func (a *Auth) ResetPassword(ctx context.Context, username, password string) error {
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
	if err := a.repo.ResetPassword(user.ID, hash); err != nil {
		return err
	}
	a.audit.Record(ctx, AuditEvent{Action: "user.reset_password", Target: userTarget(user.ID), Details: user.Username})
	return nil
}

func RequireAdmin(user User) error {
	if user.Role != RoleAdmin {
		return ErrForbidden
	}
	return nil
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

func (a *Auth) SetActive(ctx context.Context, id int, active bool) (User, error) {
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
	action := "user.reactivate"
	if !active {
		action = "user.deactivate"
	}
	a.audit.Record(ctx, AuditEvent{Action: action, Target: userTarget(id), Details: user.Username})
	return a.repo.UserByID(id)
}

func (a *Auth) SetRole(ctx context.Context, id int, role string) (User, error) {
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
	a.audit.Record(ctx, AuditEvent{Action: "user.set_role", Target: userTarget(id), Details: user.Username + " " + role})
	return a.repo.UserByID(id)
}

func (a *Auth) SetCanDownloadBackup(ctx context.Context, id int, allowed bool) (User, error) {
	a.adminMu.Lock()
	defer a.adminMu.Unlock()
	user, err := a.lifecycleUser(id)
	if err != nil {
		return User{}, err
	}
	if user.Role != RoleAdmin {
		return User{}, ErrInvalid
	}
	if err := a.repo.SetCanDownloadBackup(id, allowed); err != nil {
		return User{}, err
	}
	a.audit.Record(ctx, AuditEvent{Action: "user.set_backup_permission", Target: userTarget(id), Details: user.Username + " " + strconv.FormatBool(allowed)})
	return a.repo.UserByID(id)
}

func (a *Auth) ResetUserPassword(ctx context.Context, id int, password string) error {
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
	if err := a.repo.ResetPassword(user.ID, hash); err != nil {
		return err
	}
	a.audit.Record(ctx, AuditEvent{Action: "user.reset_password", Target: userTarget(id), Details: user.Username})
	return nil
}

func userTarget(id int) string {
	return "user:" + strconv.Itoa(id)
}

func (a *Auth) ChangePassword(ctx context.Context, token, current, next string) error {
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
		a.audit.RecordAs(ctx, Actor{UserID: user.ID, Username: user.Username, IP: ActorFrom(ctx).IP}, AuditEvent{Action: "user.change_password_failed", Target: userTarget(user.ID), Details: user.Username})
		return ErrInvalidCredentials
	}
	if !ValidPassword(next) {
		return ErrWeakPassword
	}
	hash, err := hashPassword(next)
	if err != nil {
		return err
	}
	if err := a.repo.ChangePassword(user.ID, hash, hashToken(token)); err != nil {
		return err
	}
	a.audit.RecordAs(ctx, Actor{UserID: user.ID, Username: user.Username, IP: ActorFrom(ctx).IP}, AuditEvent{Action: "user.change_password", Target: userTarget(user.ID), Details: user.Username})
	return nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (a *Auth) Login(clientKey, username, password string) (token string, expires time.Time, err error) {
	now := a.now()
	if a.limited(clientKey, now) {
		a.auditFailedLogin(clientKey, username, "throttled")
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
		a.auditFailedLogin(clientKey, username, "")
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
	a.audit.RecordAs(context.Background(), Actor{UserID: user.ID, Username: user.Username, IP: clientKey}, AuditEvent{Action: "login", Target: userTarget(user.ID)})
	return token, expires, nil
}

func (a *Auth) shouldAuditFailedLogin(clientKey, username, details string, now time.Time) bool {
	a.auditMu.Lock()
	defer a.auditMu.Unlock()
	key := details + "|" + clientKey + "|" + strings.ToLower(username)
	if at, ok := a.auditSeen[key]; ok && now.Sub(at) < loginAuditWindow {
		return false
	}
	if _, ok := a.auditSeen[key]; !ok && len(a.auditSeen) >= loginAuditMaxTracked {
		var oldest string
		var oldestAt time.Time
		for k, at := range a.auditSeen {
			if now.Sub(at) >= loginAuditWindow {
				delete(a.auditSeen, k)
			} else if oldest == "" || at.Before(oldestAt) {
				oldest, oldestAt = k, at
			}
		}
		if len(a.auditSeen) >= loginAuditMaxTracked {
			delete(a.auditSeen, oldest)
		}
	}
	a.auditSeen[key] = now
	return true
}

func (a *Auth) auditFailedLogin(clientKey, username, details string) {
	if !a.shouldAuditFailedLogin(clientKey, username, details, a.now()) {
		return
	}
	a.audit.RecordAs(context.Background(), Actor{Username: strings.ToLower(username), IP: clientKey}, AuditEvent{Action: "login_failed", Details: details})
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

func (a *Auth) makeRoomForUsername(now time.Time) {
	for name := range a.userFail {
		a.recentUsername(name, now)
	}
	if len(a.userFail) < usernameMaxTracked {
		return
	}
	var oldest string
	var oldestAt time.Time
	for name, failures := range a.userFail {
		if last := failures[len(failures)-1]; oldest == "" || last.Before(oldestAt) {
			oldest, oldestAt = name, last
		}
	}
	delete(a.userFail, oldest)
}

func (a *Auth) recordUsernameFailure(username string, now time.Time) time.Duration {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, tracked := a.userFail[username]; !tracked && len(a.userFail) >= usernameMaxTracked {
		a.makeRoomForUsername(now)
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
