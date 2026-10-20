package handler_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/sollie/matchmark/internal/apitest"
)

func auditRows(t *testing.T, env *apitest.Env, query string) map[string]any {
	t.Helper()
	status, out := env.Do("GET", "/api/audit-log"+query, nil)
	if status != http.StatusOK {
		t.Fatalf("audit log: %d %v", status, out)
	}
	return out
}

func auditEntries(t *testing.T, env *apitest.Env, query string) []map[string]any {
	t.Helper()
	var rows []map[string]any
	for _, e := range auditRows(t, env, query)["entries"].([]any) {
		rows = append(rows, e.(map[string]any))
	}
	return rows
}

func findAudit(rows []map[string]any, action string) map[string]any {
	for _, r := range rows {
		if r["action"] == action {
			return r
		}
	}
	return nil
}

func findAuditLine(lines []map[string]any, action string) map[string]any {
	for _, l := range lines {
		if l["action"] == action {
			return l
		}
	}
	return nil
}

func TestEveryEventTypeIsAuditedInTableAndStdout(t *testing.T) {
	f := &scoringFixture{fixture: newFixture(t, "150"), entries: map[int]int{}}
	env := f.env
	comp := f.comp
	e1 := f.addEntry(t, "One", 1)
	e3 := f.addEntry(t, "Three", 3)
	e4 := f.addEntry(t, "Four", 4)
	e5 := f.addEntry(t, "Five", 5)

	extra := env.ID("POST", "/api/shooters", map[string]any{"last_name": "Extra", "first_name": "A", "sex": "M", "country": "NOR", "club": "Club"})
	env.Do("PUT", fmt.Sprintf("/api/shooters/%d", extra), map[string]any{"last_name": "Extra2", "first_name": "A", "sex": "M", "country": "NOR", "club": "Club"})
	env.Do("DELETE", fmt.Sprintf("/api/shooters/%d", extra), nil)
	env.Do("PUT", fmt.Sprintf("/api/competitions/%d", comp), map[string]any{"name": "Renamed", "date": "2026-05-02", "course": "150"})
	env.Do("PUT", fmt.Sprintf("/api/competitions/%d/visibility", comp), map[string]any{"names": true})
	env.Do("PUT", fmt.Sprintf("/api/competitions/%d/entries/%d", comp, e1), map[string]any{"category": "Pistol", "lane": 1})
	env.Do("POST", fmt.Sprintf("/api/competitions/%d/entries/%d/withdraw", comp, e4), nil)
	env.Do("DELETE", fmt.Sprintf("/api/competitions/%d/entries/%d", comp, e5), nil)
	doomed := env.ID("POST", "/api/competitions", map[string]any{"name": "Doomed", "date": "2026-05-01", "course": "150"})
	env.Do("DELETE", fmt.Sprintf("/api/competitions/%d", doomed), nil)
	env.Do("POST", fmt.Sprintf("/api/competitions/%d/status", comp), map[string]any{"status": "active"})
	stage := fmt.Sprintf("/api/competitions/%d/entries/%d/stages/1", comp, e3)
	env.Do("PUT", stage, counts(10, 2))
	token := f.pair(t, 1)
	if status, out := env.DoToken(token, "POST", "/api/scorer/stages/1/lock", nil); status != http.StatusOK {
		t.Fatalf("device lock: %d %v", status, out)
	}
	env.Do("POST", stage+"/unlock", nil)
	env.Do("POST", stage+"/lock", nil)
	_, pairings := env.Do("GET", fmt.Sprintf("/api/competitions/%d/pairings", comp), nil)
	pairingID := int(pairings["pairings"].([]any)[0].(map[string]any)["id"].(float64))
	env.Do("DELETE", fmt.Sprintf("/api/competitions/%d/pairings/%d", comp, pairingID), nil)
	env.Raw("GET", fmt.Sprintf("/api/competitions/%d/results.xlsx", comp))
	env.Raw("GET", fmt.Sprintf("/api/competitions/%d/results.csv", comp))
	env.Raw("GET", "/api/admin/backup")

	anna, _ := createUser(env, "anna", "user", tempPassword)
	_ = anna
	_, users := env.Do("GET", "/api/users", nil)
	annaID := int(users["users"].([]any)[1].(map[string]any)["id"].(float64))
	userPath := fmt.Sprintf("/api/users/%d", annaID)
	env.Do("POST", userPath+"/reset-password", map[string]any{"password": "another-temporary-pw"})
	env.Do("PUT", userPath+"/role", map[string]any{"role": "admin"})
	env.Do("PUT", userPath+"/backup-permission", map[string]any{"can_download_backup": true})
	env.Do("POST", userPath+"/deactivate", nil)
	env.Do("POST", userPath+"/reactivate", nil)
	env.Do("POST", "/api/admin/login", map[string]any{"username": "anna", "password": "wrong-password-123456"})
	env.Do("POST", "/api/admin/login", map[string]any{"username": "anna", "password": "another-temporary-pw"})
	env.Do("POST", "/api/admin/password", map[string]any{"current_password": "another-temporary-pw", "new_password": "a-brand-new-password"})

	env.Do("POST", "/api/admin/login", map[string]any{"username": apitest.AdminUsername, "password": apitest.AdminPassword})
	rows := auditEntries(t, env, "?page_size=200")
	lines := env.AuditLines()

	want := map[string]string{
		"shooter.create": apitest.AdminUsername, "shooter.update": apitest.AdminUsername, "shooter.delete": apitest.AdminUsername,
		"competition.create": apitest.AdminUsername, "competition.update": apitest.AdminUsername,
		"competition.delete": apitest.AdminUsername, "competition.status": apitest.AdminUsername,
		"competition.visibility": apitest.AdminUsername,
		"entry.add":              apitest.AdminUsername, "entry.update": apitest.AdminUsername,
		"entry.withdraw": apitest.AdminUsername, "entry.delete": apitest.AdminUsername,
		"stage.set_counts": apitest.AdminUsername, "stage.lock": "", "stage.unlock": apitest.AdminUsername,
		"device.pair": apitest.AdminUsername, "device.revoke": apitest.AdminUsername,
		"report.download": apitest.AdminUsername, "backup.download": apitest.AdminUsername,
		"user.create": apitest.AdminUsername, "user.reset_password": apitest.AdminUsername,
		"user.set_role": apitest.AdminUsername, "user.set_backup_permission": apitest.AdminUsername,
		"user.deactivate": apitest.AdminUsername, "user.reactivate": apitest.AdminUsername,
		"user.change_password": "anna", "login": apitest.AdminUsername, "login_failed": "anna",
	}
	for action, username := range want {
		row, line := findAudit(rows, action), findAuditLine(lines, action)
		if row == nil || line == nil {
			t.Errorf("%s: row=%v line=%v", action, row, line)
			continue
		}
		if username != "" && (row["username"] != username || line["username"] != username) {
			t.Errorf("%s: row user %v, line user %v, want %s", action, row["username"], line["username"], username)
		}
		if row["ip"] == "" || line["ip"] != row["ip"] || row["time"] == "" {
			t.Errorf("%s: ip/time row=%v line=%v", action, row, line)
		}
	}
	if got := findAudit(rows, "report.download"); got["details"] == "" {
		t.Errorf("report download lacks format: %v", got)
	}

	if annaLogins := auditEntries(t, env, "?user=anna&action=login"); len(annaLogins) != 1 {
		t.Errorf("anna logins: %v", annaLogins)
	}

	var deviceLocks int
	for _, r := range rows {
		if r["action"] == "stage.lock" && r["username"] == "device" {
			deviceLocks++
			if r["user_id"] != nil || r["details"] != fmt.Sprintf("lane 1 competition %d", comp) {
				t.Errorf("device lock row: %v", r)
			}
		}
	}
	if deviceLocks != 1 {
		t.Errorf("device locks = %d", deviceLocks)
	}
	for _, l := range lines {
		if l["action"] == "stage.lock" && l["username"] == "device" && l["details"] != fmt.Sprintf("lane 1 competition %d", comp) {
			t.Errorf("device lock line: %v", l)
		}
	}
	if adminLock := findAudit(rows, "login"); adminLock["user_id"] == nil {
		t.Errorf("login row lacks user id: %v", adminLock)
	}
}

func TestFailedLoginForUnknownUserKeepsAttemptedName(t *testing.T) {
	env := apitest.New(t)
	env.Do("POST", "/api/admin/login", map[string]any{"username": "Nobody", "password": "whatever-password-1"})
	env.Do("POST", "/api/admin/login", map[string]any{"username": apitest.AdminUsername, "password": apitest.AdminPassword})
	rows := auditEntries(t, env, "")
	failed := findAudit(rows, "login_failed")
	if failed == nil || failed["username"] != "nobody" || failed["user_id"] != nil {
		t.Fatalf("failed login row: %v", failed)
	}
}

func TestAuditRowsKeepUsernameAfterDeactivation(t *testing.T) {
	env := apitest.NewAdmin(t)
	createUser(env, "anna", "user", tempPassword)
	_, users := env.Do("GET", "/api/users", nil)
	annaID := int(users["users"].([]any)[1].(map[string]any)["id"].(float64))
	env.Do("POST", "/api/admin/logout", nil)
	env.Do("POST", "/api/admin/login", map[string]any{"username": "anna", "password": tempPassword})
	env.Do("POST", "/api/admin/password", map[string]any{"current_password": tempPassword, "new_password": "anna-real-password-1"})
	env.Do("POST", "/api/admin/login", map[string]any{"username": "anna", "password": "anna-real-password-1"})
	env.Do("POST", "/api/shooters", map[string]any{"last_name": "Berg", "first_name": "Ola", "sex": "M", "country": "NOR", "club": "Club"})
	env.Do("POST", "/api/admin/logout", nil)
	env.Do("POST", "/api/admin/login", map[string]any{"username": apitest.AdminUsername, "password": apitest.AdminPassword})
	env.Do("POST", fmt.Sprintf("/api/users/%d/deactivate", annaID), nil)

	rows := auditEntries(t, env, "?user=anna&action=shooter.create")
	if len(rows) != 1 || rows[0]["username"] != "anna" || int(rows[0]["user_id"].(float64)) != annaID {
		t.Fatalf("anna shooter.create rows: %v", rows)
	}
}

func TestAuditLogIsAdminOnlyAndReadOnly(t *testing.T) {
	env := apitest.New(t)
	if status, _ := env.Do("GET", "/api/audit-log", nil); status != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", status)
	}
	env.Do("POST", "/api/admin/login", map[string]any{"username": apitest.AdminUsername, "password": apitest.AdminPassword})
	createUser(env, "anna", "user", tempPassword)
	env.Do("POST", "/api/admin/logout", nil)
	env.Do("POST", "/api/admin/login", map[string]any{"username": "anna", "password": tempPassword})
	env.Do("POST", "/api/admin/password", map[string]any{"current_password": tempPassword, "new_password": "anna-real-password-1"})
	env.Do("POST", "/api/admin/login", map[string]any{"username": "anna", "password": "anna-real-password-1"})
	if status, _ := env.Do("GET", "/api/audit-log", nil); status != http.StatusForbidden {
		t.Fatalf("user: %d", status)
	}

	env.Do("POST", "/api/admin/login", map[string]any{"username": apitest.AdminUsername, "password": apitest.AdminPassword})
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		for _, path := range []string{"/api/audit-log", "/api/audit-log/1"} {
			if status, _ := env.Do(method, path, map[string]any{}); status != http.StatusNotFound && status != http.StatusMethodNotAllowed {
				t.Fatalf("%s %s: %d", method, path, status)
			}
		}
	}
}

func TestAuditLogPagesAndFilters(t *testing.T) {
	env := apitest.NewAdmin(t)
	for i := 0; i < 5; i++ {
		env.ID("POST", "/api/competitions", map[string]any{"name": fmt.Sprintf("C%d", i), "date": "2026-05-01", "course": "150"})
	}
	first := auditRows(t, env, "?page_size=2&page=1")
	second := auditRows(t, env, "?page_size=2&page=2")
	if first["total"].(float64) < 6 || len(first["entries"].([]any)) != 2 || len(second["entries"].([]any)) != 2 {
		t.Fatalf("paging: %v %v", first, second)
	}
	if first["entries"].([]any)[0].(map[string]any)["id"] == second["entries"].([]any)[0].(map[string]any)["id"] {
		t.Fatal("pages overlap")
	}

	byAction := auditEntries(t, env, "?action=competition.create&page_size=200")
	if len(byAction) != 5 {
		t.Fatalf("action filter: %d", len(byAction))
	}
	if rows := auditEntries(t, env, "?user=nobody"); len(rows) != 0 {
		t.Fatalf("user filter: %v", rows)
	}
	if rows := auditEntries(t, env, "?user="+url.QueryEscape("ADMIN")+"&action=login"); len(rows) != 1 {
		t.Fatalf("user+action filter: %v", rows)
	}
	if rows := auditEntries(t, env, "?from=2999-01-01"); len(rows) != 0 {
		t.Fatalf("from filter: %v", rows)
	}
	if rows := auditEntries(t, env, "?to=2000-01-01"); len(rows) != 0 {
		t.Fatalf("to filter: %v", rows)
	}
	today := env.Today()
	if rows := auditEntries(t, env, "?from="+today+"&to="+today+"&action=login"); len(rows) != 1 {
		t.Fatalf("same-day filter: %v", rows)
	}
	for _, q := range []string{"?page=0", "?page_size=x", "?from=yesterday", "?to=2026-13-01"} {
		if status, _ := env.Do("GET", "/api/audit-log"+q, nil); status != http.StatusUnprocessableEntity {
			t.Fatalf("%s: %d", q, status)
		}
	}
}
