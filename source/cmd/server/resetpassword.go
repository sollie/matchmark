package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
)

const resetPasswordUsage = "usage: matchmark reset-password <username> (new temporary password on stdin)"

func runResetPassword(args []string, getenv func(string) string, stdin io.Reader, stdout io.Writer) error {
	if len(args) != 1 {
		return errors.New(resetPasswordUsage)
	}
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	password := strings.TrimRight(line, "\r\n")
	if password == "" {
		return errors.New("no password on stdin\n" + resetPasswordUsage)
	}
	cfg, err := LoadConfig(getenv)
	if err != nil {
		return err
	}
	if _, err := os.Stat(cfg.DBPath); err != nil {
		return fmt.Errorf("database %s: %w", cfg.DBPath, err)
	}
	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	repo := repository.New(conn)
	audit := service.NewAudit(repo, slog.New(slog.NewJSONHandler(stdout, nil)))
	auth := service.NewAuth(repo, service.WithAuthAudit(audit))
	ctx := service.WithActor(context.Background(), service.Actor{Username: service.CLIActor})
	if err := auth.ResetPassword(ctx, args[0], password); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "password reset for %s; their sessions ended and they must change it at next login\n", strings.ToLower(args[0]))
	return err
}
