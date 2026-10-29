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
	if err := conn.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil || n < 1 {
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

func TestMatchRequiresMatch(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "mm.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.Exec(`INSERT INTO matches (competition_id, name, date, course) VALUES (99, 'M', '2026-10-19', '150')`); err == nil {
		t.Fatalf("match without match inserted")
	}
	if _, err := conn.Exec(`INSERT INTO competitions (name, start_date, end_date) VALUES ('C', '2026-10-19', '2026-10-20')`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`INSERT INTO matches (competition_id, name, date, course) VALUES (1, 'M', '2026-10-19', '150')`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`INSERT INTO competitions (name, start_date, end_date) VALUES ('C', '2026-10-20', '2026-10-19')`); err == nil {
		t.Fatalf("inverted match range inserted")
	}
}

func TestActivatedAtBackfill(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mm.db")
	conn, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO competitions (name, start_date, end_date) VALUES ('C', '2026-10-19', '2026-10-19')`,
		`INSERT INTO matches (competition_id, name, date, course, status) VALUES (1, 'D', '2026-10-19', '150', 'draft')`,
		`INSERT INTO matches (competition_id, name, date, course, status) VALUES (1, 'A', '2026-10-18', '150', 'active')`,
		`INSERT INTO matches (competition_id, name, date, course, status) VALUES (1, 'F', '2026-10-17', '150', 'finished')`,
		`ALTER TABLE matches DROP COLUMN activated_at`,
		`DELETE FROM schema_migrations WHERE version = 3`,
	} {
		if _, err := conn.Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	_ = conn.Close()
	conn, err = db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	got := map[string]*string{}
	rows, err := conn.Query(`SELECT name, activated_at FROM matches`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name string
		var at *string
		if err := rows.Scan(&name, &at); err != nil {
			t.Fatal(err)
		}
		got[name] = at
	}
	if got["D"] != nil {
		t.Errorf("draft activated_at = %v", *got["D"])
	}
	if got["A"] == nil || *got["A"] != "2026-10-18" || got["F"] == nil || *got["F"] != "2026-10-17" {
		t.Errorf("backfill = %v", got)
	}
}
