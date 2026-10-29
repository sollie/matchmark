package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/repository"
)

const restoreLine = `{"msg":"segment_status","schema_version":4,"event":"segment_status","status":"accepted","accepted_by":"bob","accepted_at":"2026-10-19T10:00:00Z","emitted_at":"2026-10-19T10:00:01.000Z","seq":1,` +
	`"competition":{"id":3,"name":"Nationals"},` +
	`"match":{"id":7,"name":"Club Match","date":"2026-10-19","course_of_fire":{"key":"150"}},` +
	`"entry":{"id":1,"lane":2,"category":"Pistol","shooter":{"first_name":"Ann","last_name":"Lee"}},` +
	`"segment":{"number":1,"expected_shots":10},"ring_counts":[1,2,3,4,0,0,0],"locked_by":"bob","scorer_name":"bob"}`

func TestRestoreCommandReadsStdinWithoutAdminPassword(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "")
	var out bytes.Buffer
	if err := runRestore(nil, os.Getenv, strings.NewReader(restoreLine+"\n"), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Club Match") || !strings.Contains(out.String(), "dry run") {
		t.Fatalf("output = %s", out.String())
	}
}

func TestRestoreCommandReadsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.jsonl")
	if err := os.WriteFile(path, []byte(restoreLine+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := runRestore([]string{path}, os.Getenv, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Club Match") {
		t.Fatalf("output = %s", out.String())
	}
}

func TestRestoreCommandRejectsUnknownSchemaVersion(t *testing.T) {
	bad := strings.Replace(restoreLine, `"schema_version":4`, `"schema_version":9`, 1)
	if err := runRestore(nil, os.Getenv, strings.NewReader(bad), &bytes.Buffer{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestRestoreCommandApplyRefusesNonEmptyUnlessMerge(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sub", "restored.db")
	env := map[string]string{"DB_PATH": dbPath}
	getenv := func(k string) string { return env[k] }
	run := func() string {
		var out bytes.Buffer
		if err := runRestore([]string{"--apply"}, getenv, strings.NewReader(restoreLine+"\n"), &out); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	first := run()
	for _, want := range []string{"restored 1 match(s), 1 entries, 1 scored segment(s)", "re-pair the devices"} {
		if !strings.Contains(first, want) {
			t.Fatalf("first output missing %q: %s", want, first)
		}
	}
	var refused bytes.Buffer
	err := runRestore([]string{"--apply"}, getenv, strings.NewReader(restoreLine+"\n"), &refused)
	if err == nil || !strings.Contains(err.Error(), "--merge") {
		t.Fatalf("err = %v", err)
	}
	var merged bytes.Buffer
	if err := runRestore([]string{"--apply", "--merge"}, getenv, strings.NewReader(restoreLine+"\n"), &merged); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(merged.String(), "restored 0 match(s), 0 entries, 0 scored segment(s)") {
		t.Fatalf("merged output = %s", merged.String())
	}
	conn, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	st, err := repository.New(conn).GetSegment(context.Background(), 7, 1, 1)
	if err != nil || st.Status != repository.StatusAccepted || st.Counts.Ten != 2 || st.ScorerName != "bob" || st.ScorerLane != nil {
		t.Fatalf("segment = %+v, %v", st, err)
	}
}

func TestRestoreCommandMergeReportsConflicts(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "restored.db")
	getenv := func(k string) string {
		if k == "DB_PATH" {
			return dbPath
		}
		return ""
	}
	if err := runRestore([]string{"--apply"}, getenv, strings.NewReader(restoreLine+"\n"), &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(restoreLine, `"ring_counts":[1,2,3,4,0,0,0]`, `"ring_counts":[5,5,0,0,0,0,0]`, 1)
	var out bytes.Buffer
	if err := runRestore([]string{"--merge", "--apply"}, getenv, strings.NewReader(changed+"\n"), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "1 conflict(s) left unchanged") || !strings.Contains(out.String(), "match 7 entry 1 segment 1") {
		t.Fatalf("output = %s", out.String())
	}
}

func TestRestoreCommandMergeRequiresApply(t *testing.T) {
	if err := runRestore([]string{"--merge"}, os.Getenv, strings.NewReader(restoreLine+"\n"), &bytes.Buffer{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestRestoreCommandDryRunDoesNotCreateDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "never.db")
	getenv := func(k string) string {
		if k == "DB_PATH" {
			return dbPath
		}
		return ""
	}
	if err := runRestore(nil, getenv, strings.NewReader(restoreLine+"\n"), &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("stat err = %v", err)
	}
}

func TestRestoreCommandAcceptsFlagsAfterPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "score.log")
	if err := os.WriteFile(path, []byte(restoreLine+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"DB_PATH": filepath.Join(dir, "r.db")}
	var out bytes.Buffer
	if err := runRestore([]string{path, "--apply"}, func(k string) string { return env[k] }, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "restored 1 match(s)") {
		t.Fatalf("output = %s", out.String())
	}
}
