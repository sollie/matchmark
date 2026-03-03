package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"
)

var ErrInvalidTicket = errors.New("invalid or expired SSE ticket")

type SSETicketService struct {
	db *sql.DB
}

func NewSSETicketService(db *sql.DB) *SSETicketService {
	return &SSETicketService{db: db}
}

func (s *SSETicketService) Issue(ctx context.Context, userID int64) (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate ticket: %w", err)
	}
	ticket := base64.RawURLEncoding.EncodeToString(raw)
	expiresAt := time.Now().Add(30 * time.Second).UTC().Format(time.RFC3339)

	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO sse_tickets (ticket, user_id, expires_at) VALUES (?, ?, ?)`,
		ticket, userID, expiresAt,
	); err != nil {
		return "", fmt.Errorf("store ticket: %w", err)
	}
	return ticket, nil
}

func (s *SSETicketService) Consume(ctx context.Context, ticket string) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var userID int64
	var expiresAt string
	var used int
	err = tx.QueryRowContext(ctx,
		`SELECT user_id, expires_at, used FROM sse_tickets WHERE ticket = ?`,
		ticket,
	).Scan(&userID, &expiresAt, &used)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrInvalidTicket
	}
	if err != nil {
		return 0, fmt.Errorf("query ticket: %w", err)
	}

	if used != 0 {
		return 0, ErrInvalidTicket
	}

	expiry, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().After(expiry) {
		return 0, ErrInvalidTicket
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE sse_tickets SET used = 1 WHERE ticket = ?`, ticket,
	); err != nil {
		return 0, fmt.Errorf("mark ticket used: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return userID, nil
}
