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

const (
	ticketCleanupInterval = 5 * time.Minute
	ticketTTL             = 30 * time.Second
	ticketEntropyBytes    = 24
)

type SSETicketService struct {
	db *sql.DB
}

func NewSSETicketService(db *sql.DB) *SSETicketService {
	svc := &SSETicketService{db: db}
	go svc.startCleanup()
	return svc
}

func (s *SSETicketService) startCleanup() {
	for {
		time.Sleep(ticketCleanupInterval)
		now := time.Now().UTC().Format(time.RFC3339)
		_, _ = s.db.Exec(
			`DELETE FROM sse_tickets WHERE expires_at < ? OR used = 1`,
			now,
		)
	}
}

func (s *SSETicketService) Issue(ctx context.Context, userID, competitionID int64) (string, error) {
	raw := make([]byte, ticketEntropyBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate ticket: %w", err)
	}
	ticket := base64.RawURLEncoding.EncodeToString(raw)
	expiresAt := time.Now().Add(ticketTTL).UTC().Format(time.RFC3339)

	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO sse_tickets (ticket, user_id, competition_id, expires_at) VALUES (?, ?, ?, ?)`,
		ticket, userID, competitionID, expiresAt,
	); err != nil {
		return "", fmt.Errorf("store ticket: %w", err)
	}
	return ticket, nil
}

func (s *SSETicketService) IssueForStage(ctx context.Context, userID, competitionID int64, lane, stageNumber int) (string, error) {
	raw := make([]byte, ticketEntropyBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate ticket: %w", err)
	}
	ticket := base64.RawURLEncoding.EncodeToString(raw)
	expiresAt := time.Now().Add(ticketTTL).UTC().Format(time.RFC3339)

	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO sse_tickets (ticket, user_id, competition_id, lane, stage_number, expires_at) VALUES (?, ?, ?, ?, ?, ?)`,
		ticket, userID, competitionID, lane, stageNumber, expiresAt,
	); err != nil {
		return "", fmt.Errorf("store ticket: %w", err)
	}
	return ticket, nil
}

func (s *SSETicketService) ConsumeForStage(ctx context.Context, ticket string, competitionID int64, lane, stageNumber int) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var userID int64
	var dbCompID int64
	var dbLane, dbStage int
	var expiresAt string
	var used int
	err = tx.QueryRowContext(ctx,
		`SELECT user_id, competition_id, lane, stage_number, expires_at, used FROM sse_tickets WHERE ticket = ?`,
		ticket,
	).Scan(&userID, &dbCompID, &dbLane, &dbStage, &expiresAt, &used)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrInvalidTicket
	}
	if err != nil {
		return 0, fmt.Errorf("query ticket: %w", err)
	}

	if used != 0 || dbCompID != competitionID || dbLane != lane || dbStage != stageNumber {
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

func (s *SSETicketService) Consume(ctx context.Context, ticket string) (int64, int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var userID, competitionID int64
	var expiresAt string
	var used int
	err = tx.QueryRowContext(ctx,
		`SELECT user_id, competition_id, expires_at, used FROM sse_tickets WHERE ticket = ?`,
		ticket,
	).Scan(&userID, &competitionID, &expiresAt, &used)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, ErrInvalidTicket
	}
	if err != nil {
		return 0, 0, fmt.Errorf("query ticket: %w", err)
	}

	if used != 0 {
		return 0, 0, ErrInvalidTicket
	}

	expiry, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().After(expiry) {
		return 0, 0, ErrInvalidTicket
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE sse_tickets SET used = 1 WHERE ticket = ?`, ticket,
	); err != nil {
		return 0, 0, fmt.Errorf("mark ticket used: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("commit: %w", err)
	}
	return userID, competitionID, nil
}
