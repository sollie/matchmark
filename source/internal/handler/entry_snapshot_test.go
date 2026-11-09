package handler_test

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sollie/matchmark/internal/apitest"
	"github.com/sollie/matchmark/internal/wa1500"
)

type snapshotEnv struct {
	env   *apitest.Env
	fake  *fakeRegister
	match int
}

func newSnapshotEnv(t *testing.T, class wa1500.Classification) *snapshotEnv {
	t.Helper()
	fake := &fakeRegister{}
	fake.set([]wa1500.Shooter{registerShooter("1001", "ANDERSON", "Old Club", map[string]wa1500.Classification{"Revolver 1500": class})}, nil)
	env := newRegisterEnv(t, fake)
	return &snapshotEnv{env: env, fake: fake, match: newMatch(env)}
}

func (s *snapshotEnv) addEntry(t *testing.T) (shooter, entry int) {
	t.Helper()
	shooter = createShooter(s.env, "1001")
	status, out := addToMatch(s.env, s.match, shooter, "Revolver 1500", 1)
	if status != http.StatusCreated {
		t.Fatalf("add: %d %v", status, out)
	}
	return shooter, int(out["id"].(float64))
}

func (s *snapshotEnv) setStatus(t *testing.T, status string) {
	t.Helper()
	if code, out := s.env.Do("POST", fmt.Sprintf("/api/matches/%d/status", s.match), map[string]string{"status": status}); code != http.StatusOK {
		t.Fatalf("%s: %d %v", status, code, out)
	}
}

func (s *snapshotEnv) entry(t *testing.T) map[string]any {
	t.Helper()
	_, out := s.env.Do("GET", fmt.Sprintf("/api/matches/%d/entries", s.match), nil)
	return out["entries"].([]any)[0].(map[string]any)
}

func (s *snapshotEnv) reclassify(t *testing.T, class wa1500.Classification, club string) {
	t.Helper()
	s.fake.set([]wa1500.Shooter{registerShooter("1001", "ANDERSON", club, map[string]wa1500.Classification{"Revolver 1500": class})}, nil)
	s.env.Advance(8 * 24 * time.Hour)
	s.env.Login()
	s.env.RunShooterSyncBatch()
}

func name(v any) any {
	if m, ok := v.(map[string]any); ok {
		return m["name"]
	}
	return v
}

func TestEntrySnapshotIsFrozenOnActivationAndClearedOnBackToDraft(t *testing.T) {
	s := newSnapshotEnv(t, wa1500.Expert)
	s.addEntry(t)
	e := s.entry(t)
	if e["classification_at_match"] != nil || e["class"] != nil || e["club"] != nil || e["country"] != nil {
		t.Fatalf("draft entry frozen: %v", e)
	}
	s.setStatus(t, "active")
	e = s.entry(t)
	if name(e["classification_at_match"]) != "Expert" || name(e["class"]) != "Expert" || e["club"] != "Old Club" || e["country"] != "Canada" {
		t.Fatalf("frozen entry: %v", e)
	}
	if status, out := s.env.Do("POST", fmt.Sprintf("/api/matches/%d/back-to-draft", s.match), nil); status != http.StatusOK {
		t.Fatalf("back to draft: %d %v", status, out)
	}
	e = s.entry(t)
	if e["classification_at_match"] != nil || e["class"] != nil || e["club"] != nil || e["country"] != nil {
		t.Fatalf("entry still frozen: %v", e)
	}
	s.reclassify(t, wa1500.Master, "New Club")
	s.setStatus(t, "active")
	e = s.entry(t)
	if name(e["classification_at_match"]) != "Master" || e["club"] != "New Club" {
		t.Fatalf("re-frozen entry: %v", e)
	}
}

func TestUnclassifiedShooterClassFollowsClubSetting(t *testing.T) {
	for _, tc := range []struct{ setting, want string }{{"", "Master"}, {"Master", "Master"}, {"Unclassified", "Unclassified"}} {
		t.Run(tc.setting, func(t *testing.T) {
			s := newSnapshotEnv(t, wa1500.Unclassified)
			if tc.setting != "" {
				if status, out := s.env.Do("PUT", "/api/club", map[string]any{"unclassified_class": tc.setting}); status != http.StatusOK || out["unclassified_class"] != tc.setting {
					t.Fatalf("club: %d %v", status, out)
				}
			}
			s.addEntry(t)
			s.setStatus(t, "active")
			e := s.entry(t)
			if name(e["classification_at_match"]) != "Unclassified" || name(e["class"]) != tc.want {
				t.Fatalf("entry: %v", e)
			}
		})
	}
	s := newSnapshotEnv(t, wa1500.Master)
	if _, out := s.env.Do("GET", "/api/club", nil); out["unclassified_class"] != "Master" {
		t.Fatalf("default: %v", out)
	}
	if status, _ := s.env.Do("PUT", "/api/club", map[string]any{"unclassified_class": "Expert"}); status != http.StatusUnprocessableEntity {
		t.Fatalf("invalid setting: %d", status)
	}
}

func TestResultsIgnoreReclassificationAfterActivation(t *testing.T) {
	s := newSnapshotEnv(t, wa1500.Expert)
	s.addEntry(t)
	s.setStatus(t, "active")
	s.reclassify(t, wa1500.HighMaster, "New Club")
	s.setStatus(t, "finished")
	f := &fixture{env: s.env, comp: s.match}
	status, out := publicGet(t, f, resultsPath(f))
	if status != http.StatusOK {
		t.Fatalf("results: %d %v", status, out)
	}
	row := out["categories"].([]any)[0].(map[string]any)["rows"].([]any)[0].(map[string]any)
	if name(row["classification"]) != "Expert" || name(row["class"]) != "Expert" || row["club"] != "Old Club" {
		t.Fatalf("row follows live values: %v", row)
	}
	_, header, body := s.env.Raw("GET", fmt.Sprintf("/api/matches/%d/results.csv", s.match))
	if header.Get("Content-Type") == "" {
		t.Fatal("no content type")
	}
	rows, err := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(body, []byte("\xef\xbb\xbf")))).ReadAll()
	if err != nil || len(rows) < 2 {
		t.Fatalf("csv: %v %v", err, rows)
	}
	tail := rows[1][len(rows[1])-4:]
	if strings.Join(tail, "|") != "Expert|Expert|Old Club|Canada" {
		t.Fatalf("csv tail %v header %v", tail, rows[0][len(rows[0])-4:])
	}
}

func TestStandingsReadSnapshot(t *testing.T) {
	s := newSnapshotEnv(t, wa1500.Expert)
	s.addEntry(t)
	s.setStatus(t, "active")
	s.reclassify(t, wa1500.HighMaster, "New Club")
	f := &fixture{env: s.env, comp: s.match}
	_, out := publicGet(t, f, fmt.Sprintf("/api/spectator/matches/%d/standings", s.match))
	row := out["standings"].([]any)[0].(map[string]any)
	if name(row["classification"]) != "Expert" || name(row["class"]) != "Expert" || row["club"] != "Old Club" {
		t.Fatalf("standing %v", row)
	}
}

func TestAdminEditsEntryClassificationWithAudit(t *testing.T) {
	s := newSnapshotEnv(t, wa1500.Expert)
	_, entry := s.addEntry(t)
	path := fmt.Sprintf("/api/matches/%d/entries/%d/classification", s.match, entry)
	body := map[string]any{"classification_at_match": 3, "class": 4}
	if status, _ := s.env.Do("PUT", path, body); status != http.StatusConflict {
		t.Fatalf("draft edit: %d", status)
	}
	s.setStatus(t, "active")
	if status, _ := s.env.Do("PUT", path, map[string]any{"classification_at_match": 6, "class": 4}); status != http.StatusUnprocessableEntity {
		t.Fatalf("invalid: %d", status)
	}
	status, out := s.env.Do("PUT", path, body)
	if status != http.StatusOK || name(out["classification_at_match"]) != "Expert" || name(out["class"]) != "Master" {
		t.Fatalf("edit: %d %v", status, out)
	}
	found := false
	for _, line := range s.env.AuditLines() {
		if line["action"] == "entry.classification" {
			found = true
		}
	}
	if !found {
		t.Fatal("edit not audited")
	}
}

func TestCategoryChangeRetakesSnapshotForFrozenEntry(t *testing.T) {
	s := newSnapshotEnv(t, wa1500.Expert)
	_, entry := s.addEntry(t)
	s.setStatus(t, "active")
	if status, _ := s.env.Do("PUT", fmt.Sprintf("/api/matches/%d/entries/%d/classification", s.match, entry), map[string]any{"classification_at_match": 3, "class": 3}); status != http.StatusOK {
		t.Fatal("edit failed")
	}
	status, out := s.env.Do("PUT", fmt.Sprintf("/api/matches/%d/entries/%d", s.match, entry), map[string]any{"category": "Pistol 1500", "lane": 1})
	if status != http.StatusOK || out["classification_at_match"] != nil || out["club"] != "Old Club" {
		t.Fatalf("category change: %d %v", status, out)
	}
}
