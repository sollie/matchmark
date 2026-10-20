package service

import (
	"context"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sollie/matchmark/internal/repository"
)

const (
	DeviceActor      = "device"
	AuditDefaultPage = 50
	AuditMaxPage     = 200
	maxAuditText     = 128
)

type Actor struct {
	UserID   int
	Username string
	IP       string
}

type actorKey struct{}

func WithActor(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, a)
}

func ActorFrom(ctx context.Context) Actor {
	a, _ := ctx.Value(actorKey{}).(Actor)
	return a
}

type AuditEvent struct {
	Action  string
	Target  string
	Details string
}

type AuditQuery struct {
	Username string
	Action   string
	From     time.Time
	To       time.Time
	Page     int
	PageSize int
}

type AuditPage struct {
	Entries  []repository.AuditEntry
	Total    int
	Page     int
	PageSize int
}

type Audit struct {
	repo *repository.Repository
	log  *slog.Logger
	now  func() time.Time
}

func NewAudit(repo *repository.Repository, log *slog.Logger) *Audit {
	return &Audit{repo: repo, log: log, now: time.Now}
}

func truncate(s string) string {
	if utf8.RuneCountInString(s) <= maxAuditText {
		return s
	}
	return string([]rune(s)[:maxAuditText])
}

func (a *Audit) Record(ctx context.Context, ev AuditEvent) {
	a.RecordAs(ctx, ActorFrom(ctx), ev)
}

func (a *Audit) RecordAs(ctx context.Context, actor Actor, ev AuditEvent) {
	if a == nil {
		return
	}
	ctx = context.WithoutCancel(ctx)
	var userID *int
	if actor.UserID > 0 {
		userID = &actor.UserID
	}
	entry := repository.AuditEntry{
		CreatedAt: a.now(),
		UserID:    userID,
		Username:  truncate(actor.Username),
		Action:    ev.Action,
		Target:    truncate(ev.Target),
		Details:   truncate(ev.Details),
		IP:        actor.IP,
	}
	if err := a.repo.InsertAudit(ctx, entry); err != nil && a.log != nil {
		a.log.ErrorContext(ctx, "audit write failed", "action", ev.Action, "error", err)
	}
	if a.log == nil {
		return
	}
	attrs := []slog.Attr{
		slog.String("username", entry.Username),
		slog.String("action", entry.Action),
		slog.String("target", entry.Target),
		slog.String("details", entry.Details),
		slog.String("ip", entry.IP),
	}
	if userID != nil {
		attrs = append(attrs, slog.Int("user_id", *userID))
	}
	a.log.LogAttrs(ctx, slog.LevelInfo, "audit", attrs...)
}

func (a *Audit) List(ctx context.Context, q AuditQuery) (AuditPage, error) {
	if a == nil {
		return AuditPage{}, ErrNotFound
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = AuditDefaultPage
	}
	q.PageSize = min(q.PageSize, AuditMaxPage)
	entries, total, err := a.repo.ListAudit(ctx, repository.AuditFilter{
		Username: strings.TrimSpace(q.Username),
		Action:   strings.TrimSpace(q.Action),
		From:     q.From,
		To:       q.To,
		Limit:    q.PageSize,
		Offset:   (q.Page - 1) * q.PageSize,
	})
	if err != nil {
		return AuditPage{}, err
	}
	return AuditPage{Entries: entries, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}
