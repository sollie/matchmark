package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenEventLogUnsetWritesStdoutOnly(t *testing.T) {
	var out bytes.Buffer
	l, c, err := openEventLog("", &out)
	if err != nil {
		t.Fatal(err)
	}
	l.Info("hello")
	if err := c.Close(); err != nil || !strings.Contains(out.String(), `"msg":"hello"`) {
		t.Fatalf("out = %s, err = %v", out.String(), err)
	}
}

func TestOpenEventLogAppendsToFileWith0600(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.log")
	for _, msg := range []string{"one", "two"} {
		var out bytes.Buffer
		l, c, err := openEventLog(path, &out)
		if err != nil {
			t.Fatal(err)
		}
		l.Info(msg)
		_ = c.Close()
		if !strings.Contains(out.String(), msg) {
			t.Fatalf("stdout = %s", out.String())
		}
	}
	data, err := os.ReadFile(path)
	if err != nil || strings.Count(string(data), "\n") != 2 {
		t.Fatalf("file = %q, err = %v", data, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, err = %v", info.Mode(), err)
	}
}

func TestOpenEventLogFailsWhenFileCannotBeOpened(t *testing.T) {
	_, _, err := openEventLog(filepath.Join(t.TempDir(), "missing", "events.log"), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "LOG_FILE") {
		t.Fatalf("err = %v", err)
	}
}

func TestOpenEventLogWriteFailureLogsErrorToStdoutOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.log")
	var out bytes.Buffer
	l, c, err := openEventLog(path, &out)
	if err != nil {
		t.Fatal(err)
	}
	_ = c.Close()
	l.Info("score")
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], `"msg":"score"`) ||
		!strings.Contains(lines[1], `"level":"ERROR"`) || !strings.Contains(lines[1], "log file write failed") {
		t.Fatalf("stdout = %s", out.String())
	}
	data, _ := os.ReadFile(path)
	if len(data) != 0 {
		t.Fatalf("file = %q", data)
	}
}
