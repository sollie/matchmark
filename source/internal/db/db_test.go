package db_test

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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

func migrateTo(t *testing.T, path string, upTo int) {
	t.Helper()
	conn, err := sql.Open("turso", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	for _, f := range files {
		version, err := strconv.Atoi(strings.SplitN(filepath.Base(f), "_", 2)[0])
		if err != nil {
			t.Fatal(err)
		}
		if version > upTo {
			break
		}
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Exec(string(body)); err != nil {
			t.Fatalf("migration %s: %v", f, err)
		}
		if _, err := conn.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, version); err != nil {
			t.Fatal(err)
		}
	}
}

func execAll(t *testing.T, conn *sql.DB, queries ...string) {
	t.Helper()
	for _, q := range queries {
		if _, err := conn.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMigrationBackfillsLockedAt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mm.db")
	migrateTo(t, path, 11)
	conn, err := sql.Open("turso", path)
	if err != nil {
		t.Fatal(err)
	}
	execAll(t, conn,
		`INSERT INTO shooters (last_name, first_name, sex, country, club) VALUES ('A', 'B', 'M', 'NOR', 'C')`,
		`INSERT INTO competitions (name, date, course) VALUES ('M', '2026-10-19', '150')`,
		`INSERT INTO entries (competition_id, shooter_id, category, lane) VALUES (1, 1, 'Revolver', 1)`,
		`INSERT INTO stages (entry_id, stage_number, expected_shots, locked, scorer_name) VALUES (1, 1, 12, 1, 'Admin')`,
		`INSERT INTO stages (entry_id, stage_number, expected_shots, locked) VALUES (1, 2, 12, 0)`,
	)
	_ = conn.Close()
	reopened, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reopened.Close() }()
	var locked, unlocked *string
	if err := reopened.QueryRow(`SELECT locked_at FROM stages WHERE stage_number = 1`).Scan(&locked); err != nil {
		t.Fatal(err)
	}
	if err := reopened.QueryRow(`SELECT locked_at FROM stages WHERE stage_number = 2`).Scan(&unlocked); err != nil {
		t.Fatal(err)
	}
	if locked == nil || *locked == "" || unlocked != nil {
		t.Fatalf("locked_at = %v / %v", locked, unlocked)
	}
}

func TestMigrationBoundsHitCounts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mm.db")
	migrateTo(t, path, 14)
	conn, err := sql.Open("turso", path)
	if err != nil {
		t.Fatal(err)
	}
	execAll(t, conn,
		`INSERT INTO shooters (last_name, first_name, sex, country, club) VALUES ('A', 'B', 'M', 'NOR', 'C')`,
		`INSERT INTO competitions (name, date, course) VALUES ('M', '2026-10-19', '150')`,
		`INSERT INTO entries (competition_id, shooter_id, category, lane) VALUES (1, 1, 'Revolver', 1)`,
		`INSERT INTO stages (entry_id, stage_number, expected_shots, x, ten) VALUES (1, 1, 12, 7, 5)`,
	)
	_ = conn.Close()
	reopened, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reopened.Close() }()
	var x, ten int
	if err := reopened.QueryRow(`SELECT x, ten FROM stages WHERE stage_number = 1`).Scan(&x, &ten); err != nil || x != 7 || ten != 5 {
		t.Fatalf("stage after migration: %d %d %v", x, ten, err)
	}
	if _, err := reopened.Exec(`UPDATE stages SET x = 13 WHERE stage_number = 1`); err == nil {
		t.Fatal("expected out-of-range count to be rejected")
	}
}

func TestMigrationMapsLockedStagesToAccepted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mm.db")
	migrateTo(t, path, 15)
	conn, err := sql.Open("turso", path)
	if err != nil {
		t.Fatal(err)
	}
	execAll(t, conn,
		`INSERT INTO shooters (last_name, first_name, sex, country, club) VALUES ('A', 'B', 'M', 'NOR', 'C')`,
		`INSERT INTO competitions (name, date, course) VALUES ('M', '2026-10-19', '150')`,
		`INSERT INTO entries (competition_id, shooter_id, category, lane) VALUES (1, 1, 'Revolver', 1)`,
		`INSERT INTO stages (entry_id, stage_number, expected_shots, locked, scorer_name, locked_at) VALUES (1, 1, 12, 1, 'Admin', '2026-10-19T10:00:00Z')`,
		`INSERT INTO stages (entry_id, stage_number, expected_shots, locked) VALUES (1, 2, 12, 0)`,
	)
	_ = conn.Close()
	reopened, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reopened.Close() }()
	rows, err := reopened.Query(`SELECT status, COALESCE(accepted_by, ''), COALESCE(accepted_at, '') FROM stages ORDER BY stage_number`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var got []string
	for rows.Next() {
		var s, by, at string
		if err := rows.Scan(&s, &by, &at); err != nil {
			t.Fatal(err)
		}
		got = append(got, s+"|"+by+"|"+at)
	}
	want := []string{"accepted|Admin|2026-10-19T10:00:00Z", "open||"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("stages = %v", got)
	}
}

func reopenBeforeNormalization(t *testing.T, rows [][2]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mm.db")
	conn, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`DELETE FROM schema_migrations WHERE version = 20`); err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if _, err := conn.Exec(`INSERT INTO shooters (last_name, first_name, sex, country, club) VALUES (?, ?, 'M', 'NOR', 'C')`, r[0], r[1]); err != nil {
			t.Fatal(err)
		}
	}
	_ = conn.Close()
	return path
}

func shooterNames(t *testing.T, conn *sql.DB) [][2]string {
	t.Helper()
	rs, err := conn.Query(`SELECT last_name, first_name FROM shooters ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rs.Close() }()
	var out [][2]string
	for rs.Next() {
		var n [2]string
		if err := rs.Scan(&n[0], &n[1]); err != nil {
			t.Fatal(err)
		}
		out = append(out, n)
	}
	return out
}

func captureLogs(t *testing.T) *strings.Builder {
	t.Helper()
	var buf strings.Builder
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func TestMigrationNormalizesShooterNames(t *testing.T) {
	path := reopenBeforeNormalization(t, [][2]string{
		{"anderson", "MIKE"},
		{"  smith   jones ", "anna-lisa"},
		{"østergård", "ÉMILIE"},
		{"ALREADY", "Fine"},
	})
	conn, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	want := [][2]string{
		{"ANDERSON", "Mike"},
		{"SMITH JONES", "Anna-Lisa"},
		{"ØSTERGÅRD", "Émilie"},
		{"ALREADY", "Fine"},
	}
	got := shooterNames(t, conn)
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("shooter %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestMigrationLogsCollisionsWithoutMerging(t *testing.T) {
	logs := captureLogs(t)
	path := reopenBeforeNormalization(t, [][2]string{
		{"anderson", "mike"},
		{"ANDERSON", "MIKE"},
		{"Other", "Person"},
	})
	conn, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	got := shooterNames(t, conn)
	if len(got) != 3 || got[0] != [2]string{"ANDERSON", "Mike"} || got[1] != got[0] {
		t.Fatalf("shooters = %v", got)
	}
	out := logs.String()
	if !strings.Contains(out, "shooter name collision") || !strings.Contains(out, `"shooter_ids":[1,2]`) {
		t.Fatalf("collision not logged: %s", out)
	}
	if strings.Contains(out, "Other") {
		t.Fatalf("non-colliding shooter logged: %s", out)
	}
}

func TestMigrationNormalizationIsIdempotent(t *testing.T) {
	path := reopenBeforeNormalization(t, [][2]string{{"anderson", "mike"}})
	conn, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`UPDATE shooters SET first_name = 'edited'`); err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
	conn, err = db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if got := shooterNames(t, conn); got[0][1] != "edited" {
		t.Fatalf("normalization ran again: %v", got)
	}
}

func TestMigrationRenamesRifleToCarbineKeepingStages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mm.db")
	migrateTo(t, path, 20)
	conn, err := sql.Open("turso", path)
	if err != nil {
		t.Fatal(err)
	}
	execAll(t, conn,
		`INSERT INTO shooters (last_name, first_name, sex, country, club) VALUES ('A', 'B', 'M', 'NOR', 'C')`,
		`INSERT INTO shooters (last_name, first_name, sex, country, club) VALUES ('D', 'E', 'M', 'NOR', 'C')`,
		`INSERT INTO competitions (name, date, course) VALUES ('M', '2026-10-19', '150')`,
		`INSERT INTO entries (competition_id, shooter_id, category, lane) VALUES (1, 1, 'Rifle', 1)`,
		`INSERT INTO entries (competition_id, shooter_id, category, lane) VALUES (1, 2, 'Revolver', 2)`,
		`INSERT INTO stages (entry_id, stage_number, expected_shots, x) VALUES (1, 1, 12, 3)`,
		`INSERT INTO stages (entry_id, stage_number, expected_shots) VALUES (2, 1, 12)`,
	)
	_ = conn.Close()
	reopened, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reopened.Close() }()
	var carbine, revolver, stages, x int
	if err := reopened.QueryRow(`SELECT COUNT(*) FROM entries WHERE category = 'Carbine'`).Scan(&carbine); err != nil || carbine != 1 {
		t.Fatalf("carbine entries = %d, err %v", carbine, err)
	}
	if err := reopened.QueryRow(`SELECT COUNT(*) FROM entries WHERE category = 'Revolver'`).Scan(&revolver); err != nil || revolver != 1 {
		t.Fatalf("revolver entries = %d, err %v", revolver, err)
	}
	if err := reopened.QueryRow(`SELECT COUNT(*), MAX(x) FROM stages`).Scan(&stages, &x); err != nil || stages != 2 || x != 3 {
		t.Fatalf("stages = %d x = %d, err %v", stages, x, err)
	}
	if _, err := reopened.Exec(`UPDATE entries SET category = 'Rifle' WHERE id = 1`); err == nil {
		t.Fatal("expected Rifle to be rejected")
	}
}
