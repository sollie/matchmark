package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/sollie/matchmark/internal/restore"
)

func runRestore(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) > 1 {
		return errors.New("usage: matchmark restore [score-log-file]")
	}
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
	restore.WriteSummary(stdout, restore.Fold(events), stats)
	return nil
}
