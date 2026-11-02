package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
)

type fileSink struct {
	file   *os.File
	stdout *slog.Logger
}

func (s fileSink) Write(p []byte) (int, error) {
	if _, err := s.file.Write(p); err != nil {
		s.stdout.Error("log file write failed", "error", err.Error())
	}
	return len(p), nil
}

func openEventLog(path string, stdout io.Writer) (*slog.Logger, io.Closer, error) {
	if path == "" {
		return slog.New(slog.NewJSONHandler(stdout, nil)), io.NopCloser(nil), nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("LOG_FILE: %w", err)
	}
	sink := fileSink{file: f, stdout: slog.New(slog.NewJSONHandler(stdout, nil))}
	return slog.New(slog.NewJSONHandler(io.MultiWriter(stdout, sink), nil)), f, nil
}
