package handler_test

import (
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	env.ID("POST", "/api/shooters", map[string]any{"last_name": "BERG", "first_name": "Ola", "country": "NOR", "club": "Oslo PPC"})

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
	if err := conn.QueryRow(`SELECT COUNT(*) FROM shooters WHERE last_name = 'BERG'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("restored shooters = %d, err %v", n, err)
	}
}

func backupPermissionPath(id int) string {
	return userPath(id, "backup-permission")
}

func TestSeededAdminHasBackupPermission(t *testing.T) {
	env := apitest.NewAdmin(t)
	if row := userRow(t, env, apitest.AdminUsername); row["can_download_backup"] != true {
		t.Fatalf("seeded admin: %v", row)
	}
	if _, out := env.Do("GET", "/api/admin/session", nil); out["can_download_backup"] != true {
		t.Fatalf("session: %v", out)
	}
}

func TestBackupRefusedWithoutPermission(t *testing.T) {
	admin := apitest.NewAdmin(t)
	adminID, adminClient := newUserClient(t, admin, "boris", "admin")
	_, userClient := newUserClient(t, admin, "anna", "user")

	for name, c := range map[string]*http.Client{"admin without flag": adminClient, "user": userClient} {
		resp, err := c.Get(admin.Server.URL + "/api/admin/backup")
		if err != nil || resp.StatusCode != http.StatusForbidden {
			t.Fatalf("%s: %v %v", name, resp, err)
		}
	}

	if status, out := admin.Do("PUT", backupPermissionPath(adminID), map[string]bool{"can_download_backup": true}); status != http.StatusOK || out["can_download_backup"] != true {
		t.Fatalf("grant: %d %v", status, out)
	}
	resp, err := adminClient.Get(admin.Server.URL + "/api/admin/backup")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("granted backup: %v %v", resp, err)
	}

	if status, out := admin.Do("PUT", backupPermissionPath(adminID), map[string]bool{"can_download_backup": false}); status != http.StatusOK || out["can_download_backup"] != false {
		t.Fatalf("revoke: %d %v", status, out)
	}
	resp, err = adminClient.Get(admin.Server.URL + "/api/admin/backup")
	if err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("revoked backup: %v %v", resp, err)
	}
}

func TestBackupPermissionCannotBeSetOnUserAndIsClearedOnDemotion(t *testing.T) {
	admin := apitest.NewAdmin(t)
	userID, _ := newUserClient(t, admin, "anna", "user")
	if status, _ := admin.Do("PUT", backupPermissionPath(userID), map[string]bool{"can_download_backup": true}); status != http.StatusUnprocessableEntity {
		t.Fatalf("set on user: %d", status)
	}
	if status, _ := admin.Do("PUT", backupPermissionPath(9999), map[string]bool{"can_download_backup": true}); status != http.StatusNotFound {
		t.Fatalf("unknown user: %d", status)
	}

	adminID, _ := newUserClient(t, admin, "boris", "admin")
	admin.Do("PUT", backupPermissionPath(adminID), map[string]bool{"can_download_backup": true})
	admin.Do("PUT", userPath(adminID, "role"), map[string]string{"role": "user"})
	if row := userRow(t, admin, "boris"); row["can_download_backup"] != false {
		t.Fatalf("after demotion: %v", row)
	}
	admin.Do("PUT", userPath(adminID, "role"), map[string]string{"role": "admin"})
	if row := userRow(t, admin, "boris"); row["can_download_backup"] != false {
		t.Fatalf("after re-promotion: %v", row)
	}
}

func TestOnlyAdminsChangeBackupPermission(t *testing.T) {
	admin := apitest.NewAdmin(t)
	_, userClient := newUserClient(t, admin, "anna", "user")
	req, _ := http.NewRequest("PUT", admin.Server.URL+backupPermissionPath(1), strings.NewReader(`{"can_download_backup":false}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := userClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("user: %v %v", resp, err)
	}
}
