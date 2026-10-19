package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const restoreLine = `{"msg":"stage_locked","schema_version":1,"event":"stage_locked","emitted_at":"2026-10-19T10:00:01.000Z","seq":1,` +
	`"competition":{"id":7,"name":"Club Match","date":"2026-10-19","course_of_fire":{"key":"wa1500"}},` +
	`"entry":{"id":1,"lane":2,"category":"Open","shooter":{"first_name":"Ann","last_name":"Lee"}},` +
	`"stage":{"number":1,"expected_shots":10},"ring_counts":[1,2,3,4,0,0,0],"locked_by":"Admin"}`

func TestRestoreCommandReadsStdinWithoutAdminPassword(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "")
	var out bytes.Buffer
	if err := runRestore(nil, strings.NewReader(restoreLine+"\n"), &out); err != nil {
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
	if err := runRestore([]string{path}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Club Match") {
		t.Fatalf("output = %s", out.String())
	}
}

func TestRestoreCommandRejectsUnknownSchemaVersion(t *testing.T) {
	bad := strings.Replace(restoreLine, `"schema_version":1`, `"schema_version":9`, 1)
	if err := runRestore(nil, strings.NewReader(bad), &bytes.Buffer{}); err == nil {
		t.Fatal("expected error")
	}
}
