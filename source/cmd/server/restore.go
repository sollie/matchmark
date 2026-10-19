package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/restore"
	"github.com/sollie/matchmark/internal/service"
)

const restoreUsage = "usage: matchmark restore [--apply [--merge]] [score-log-file]"

func runRestore(args []string, getenv func(string) string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("restore", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	apply := fs.Bool("apply", false, "")
	merge := fs.Bool("merge", false, "")
	var paths []string
	for {
		if err := fs.Parse(args); err != nil {
			return errors.New(restoreUsage)
		}
		if fs.NArg() == 0 {
			break
		}
		paths = append(paths, fs.Arg(0))
		args = fs.Args()[1:]
	}
	if *merge && !*apply {
		return errors.New(restoreUsage)
	}
	if len(paths) > 1 {
		return errors.New(restoreUsage)
	}
	args = paths
	in := stdin
	if len(args) == 1 {
		f, err := os.Open(args[0])
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		in = f
	}
	events, stats, err := restore.Parse(in)
	if err != nil {
		return fmt.Errorf("parse score log: %w", err)
	}
	plan := restore.Fold(events)
	if !*apply {
		restore.WriteSummary(stdout, plan, stats)
		return nil
	}
	cfg, err := LoadConfig(getenv)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o750); err != nil {
		return err
	}
	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	result, err := service.New(repository.New(conn)).RestoreScoreLog(context.Background(), plan, *merge)
	if err != nil {
		return err
	}
	restore.WriteApplied(stdout, stats, result)
	return nil
}
