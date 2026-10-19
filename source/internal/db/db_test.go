package db_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/sollie/matchmark/internal/db"
)

func TestOpenRunsMigrations(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "mm.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	var n int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil || n < 2 {
		t.Fatalf("migrations applied = %d, err %v", n, err)
	}
}

func TestForeignKeysEnforced(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "mm.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	var on int
	if err := conn.QueryRow(`PRAGMA foreign_keys`).Scan(&on); err != nil || on != 1 {
		t.Fatalf("foreign_keys = %d, err %v", on, err)
	}
	if _, err := conn.Exec(`CREATE TABLE fk_parent (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`CREATE TABLE fk_child (id INTEGER PRIMARY KEY, pid INTEGER NOT NULL REFERENCES fk_parent(id))`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`INSERT INTO fk_child (id, pid) VALUES (1, 99)`); err == nil {
		t.Fatalf("orphan insert succeeded")
	}
}

func TestSingleConnection(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "mm.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if got := conn.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("max open connections = %d", got)
	}
	held, err := conn.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if _, err := conn.Conn(ctx); err == nil {
		t.Fatalf("second connection acquired while first held")
	}
	_ = held.Close()
}

func TestBusyTimeout(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "mm.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	var ms int
	if err := conn.QueryRow(`PRAGMA busy_timeout`).Scan(&ms); err != nil || ms != 5000 {
		t.Fatalf("busy_timeout = %d, err %v", ms, err)
	}
}

func TestMigrationBackfillsLockedAt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mm.db")
	conn, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO shooters (last_name, first_name, sex, country, club) VALUES ('A', 'B', 'M', 'NOR', 'C')`,
		`INSERT INTO competitions (name, date, course) VALUES ('M', '2026-10-19', '150')`,
		`INSERT INTO entries (competition_id, shooter_id, category, lane) VALUES (1, 1, 'Revolver', 1)`,
		`INSERT INTO stages (entry_id, stage_number, expected_shots, locked, scorer_name) VALUES (1, 1, 12, 1, 'Admin')`,
		`INSERT INTO stages (entry_id, stage_number, expected_shots, locked) VALUES (1, 2, 12, 0)`,
		`DELETE FROM schema_migrations WHERE version = 12`,
	} {
		if _, err := conn.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	_ = conn.Close()
	conn, err = db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	var locked, unlocked *string
	if err := conn.QueryRow(`SELECT locked_at FROM stages WHERE stage_number = 1`).Scan(&locked); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(`SELECT locked_at FROM stages WHERE stage_number = 2`).Scan(&unlocked); err != nil {
		t.Fatal(err)
	}
	if locked == nil || *locked == "" || unlocked != nil {
		t.Fatalf("locked_at = %v / %v", locked, unlocked)
	}
}
