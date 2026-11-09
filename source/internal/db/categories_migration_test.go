package db

import (
	"path/filepath"
	"strings"
	"testing"
)

func seedLegacy(t *testing.T, course, category string) (path string) {
	t.Helper()
	path = filepath.Join(t.TempDir(), "mm.db")
	conn, err := openConn(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if err := migrate(conn, 4); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO shooters (id, last_name, first_name, country, club) VALUES (1, 'A', 'B', 'NO', 'C')`,
		`INSERT INTO competitions (id, name, start_date, end_date) VALUES (1, 'Cup', '2026-01-01', '2026-01-01')`,
		`INSERT INTO matches (id, competition_id, name, date, course) VALUES (1, 1, 'M', '2026-01-01', '` + course + `')`,
		`INSERT INTO entries (id, match_id, shooter_id, category, lane) VALUES (1, 1, 1, '` + category + `', 1)`,
		`INSERT INTO segments (entry_id, segment_number, expected_shots) VALUES (1, 1, 15)`,
	} {
		if _, err := conn.Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	return path
}

func TestCategoryMigrationRenamesAndKeepsChildren(t *testing.T) {
	for old, want := range map[string]string{
		"Revolver":    "Revolver 1500",
		"Pistol":      "Pistol 1500",
		"Revolver OS": "Revolver Optical Sight 1500",
		"Pistol OS":   "Pistol Optical Sight 1500",
		"Carbine":     "Carbine 1500",
	} {
		path := seedLegacy(t, "150", old)
		conn, err := Open(path)
		if err != nil {
			t.Fatalf("%s: %v", old, err)
		}
		var got string
		var segments int
		if err := conn.QueryRow(`SELECT category FROM entries WHERE id = 1`).Scan(&got); err != nil || got != want {
			t.Errorf("%s -> %q, err %v", old, got, err)
		}
		if err := conn.QueryRow(`SELECT COUNT(*) FROM segments WHERE entry_id = 1`).Scan(&segments); err != nil || segments != 1 {
			t.Errorf("%s: segments = %d, err %v", old, segments, err)
		}
		_ = conn.Close()
	}
}

func TestCategoryMigrationRejectsNon1500Entries(t *testing.T) {
	path := seedLegacy(t, "60", "Revolver")
	_, err := Open(path)
	if err == nil || !strings.Contains(err.Error(), "re_entered_before_upgrading") {
		t.Fatalf("err = %v", err)
	}
}
