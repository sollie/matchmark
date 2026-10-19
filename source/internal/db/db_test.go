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
