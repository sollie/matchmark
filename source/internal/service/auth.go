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
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/sollie/matchmark/internal/repository"
)

const (
	MinPasswordLength = 16
	MaxPasswordLength = 128
	SessionTTL        = 12 * time.Hour

	defaultVerifyEstimate = 100 * time.Millisecond

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
	repo        *repository.Repository
	now         func() time.Time
	verifyNanos atomic.Int64
	audit       *Audit

	adminMu  sync.Mutex
	throttle *LoginThrottle
	sleep    func(time.Duration)
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
	a := &Auth{repo: repo, now: time.Now, sleep: time.Sleep, throttle: NewLoginThrottle()}
	for _, opt := range opts {
		opt(a)
	}
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
	if !a.throttle.Allowed(key, now) {
		return ErrTooManyAttempts
	}
	if ok, _ := verifyPassword(current, user.PasswordHash); !ok {
		a.throttle.FailClient(key, now)
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

func (a *Auth) timedVerify(password, phc string) (ok, needsRehash bool) {
	start := a.now()
	ok, needsRehash = verifyPassword(password, phc)
	elapsed := int64(a.now().Sub(start))
	if elapsed > 0 {
		old := a.verifyNanos.Load()
		if old == 0 {
			a.verifyNanos.Store(elapsed)
		} else {
			a.verifyNanos.Store(old - old/4 + elapsed/4)
		}
	}
	return ok, needsRehash
}

func (a *Auth) verifyEstimate() time.Duration {
	if n := a.verifyNanos.Load(); n > 0 {
		return time.Duration(n)
	}
	return defaultVerifyEstimate
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (a *Auth) Login(clientKey, username, password string) (token string, expires time.Time, err error) {
	now := a.now()
	if !a.throttle.Allowed(clientKey, now) {
		a.auditThrottled(clientKey, username)
		return "", time.Time{}, ErrTooManyAttempts
	}
	user, lookupErr := a.repo.UserByUsername(strings.ToLower(username))
	if lookupErr != nil && !errors.Is(lookupErr, repository.ErrNotFound) {
		return "", time.Time{}, lookupErr
	}
	known := lookupErr == nil
	var ok, needsRehash bool
	if known {
		ok, needsRehash = a.timedVerify(password, user.PasswordHash)
	} else {
		a.sleep(max(0, a.verifyEstimate()-a.now().Sub(now)))
	}
	if !known || !ok || !user.Active {
		delay, audit := a.throttle.Fail(clientKey, username, now)
		if delay > 0 {
			a.sleep(delay)
		}
		if audit {
			a.recordFailedLogin(clientKey, username, "")
		}
		return "", time.Time{}, ErrInvalidCredentials
	}
	a.throttle.Success(strings.ToLower(username))
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

func (a *Auth) auditThrottled(clientKey, username string) {
	if a.throttle.ShouldAudit(clientKey, username, "throttled", a.now()) {
		a.recordFailedLogin(clientKey, username, "throttled")
	}
}

func (a *Auth) recordFailedLogin(clientKey, username, details string) {
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
