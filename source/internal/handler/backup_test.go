package handler_test

import (
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	_ "turso.tech/database/tursogo"

	"github.com/sollie/matchmark/internal/apitest"
)

func TestBackupRequiresAdmin(t *testing.T) {
	env := apitest.New(t)
	if status, _, _ := env.Raw("GET", "/api/admin/backup"); status == http.StatusOK {
		t.Fatalf("anonymous backup allowed")
	}
}

func TestBackupDownloadsUsableDatabase(t *testing.T) {
	env := apitest.NewAdmin(t)
	env.ID("POST", "/api/shooters", map[string]any{"last_name": "Berg", "first_name": "Ola", "sex": "M", "country": "NOR", "club": "Oslo PPC"})

	status, header, body := env.Raw("GET", "/api/admin/backup")
	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	if header.Get("Content-Disposition") == "" {
		t.Fatalf("missing content disposition")
	}
	path := filepath.Join(t.TempDir(), "restored.db")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	conn, err := sql.Open("turso", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	var n int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM shooters WHERE last_name = 'Berg'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("restored shooters = %d, err %v", n, err)
	}
}
