package handler_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/sollie/matchmark/internal/apitest"
	"github.com/sollie/matchmark/internal/service"
	"github.com/sollie/matchmark/internal/wa1500"
)

type fakeRegister struct {
	mu      sync.Mutex
	rows    []wa1500.Shooter
	err     error
	queries []string
}

func (f *fakeRegister) Search(_ context.Context, q string) ([]wa1500.Shooter, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queries = append(f.queries, q)
	return f.rows, f.err
}

func (f *fakeRegister) set(rows []wa1500.Shooter, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows, f.err = rows, err
}

func (f *fakeRegister) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.queries)
}

func registerShooter(waid, last, club string, classes map[string]wa1500.Classification) wa1500.Shooter {
	return wa1500.Shooter{WAID: waid, LastName: last, FirstName: "Mike", Country: "Canada", Club: club, Classes: classes}
}

func newRegisterEnv(t *testing.T, fake *fakeRegister) *apitest.Env {
	t.Helper()
	return apitest.NewAdminWithService(t, []service.Option{service.WithShooterRegister(fake)})
}

func createShooter(env *apitest.Env, waid string) int {
	body := map[string]any{"last_name": "ANDERSON", "first_name": "Mike", "country": "Norway", "club": "Local"}
	if waid != "" {
		body["waid"] = waid
	}
	return env.ID("POST", "/api/shooters", body)
}

func addToMatch(env *apitest.Env, match, shooter int, category string, lane int) (int, map[string]any) {
	return env.Do("POST", fmt.Sprintf("/api/matches/%d/entries", match), map[string]any{"shooter_id": shooter, "category": category, "lane": lane})
}

func newMatch(env *apitest.Env) int {
	return env.ID("POST", "/api/matches", map[string]any{"name": "Club Match", "date": "2026-05-01", "course": "150"})
}

func classOf(t *testing.T, env *apitest.Env, shooter int, category string) any {
	t.Helper()
	_, got := env.Do("GET", fmt.Sprintf("/api/shooters/%d", shooter), nil)
	classes, _ := got["classifications"].(map[string]any)
	return classes[category]
}

func TestAddEntryRefreshesStaleShooterFromRegister(t *testing.T) {
	fake := &fakeRegister{}
	fake.set([]wa1500.Shooter{
		registerShooter("33289", "DECOY", "Other", map[string]wa1500.Classification{"Revolver": wa1500.HighMaster}),
		registerShooter("3289", "anderson", "CCSC", map[string]wa1500.Classification{"Revolver": wa1500.Master, "Carbine": wa1500.Unclassified}),
	}, nil)
	env := newRegisterEnv(t, fake)
	shooter := createShooter(env, "3289")
	match := newMatch(env)

	if status, out := addToMatch(env, match, shooter, "Revolver", 1); status != http.StatusCreated {
		t.Fatalf("add: %d %v", status, out)
	}
	_, got := env.Do("GET", fmt.Sprintf("/api/shooters/%d", shooter), nil)
	revolver := got["classifications"].(map[string]any)["Revolver"].(map[string]any)
	if revolver["value"] != float64(4) || revolver["name"] != "Master" {
		t.Fatalf("revolver %v", revolver)
	}
	carbine := got["classifications"].(map[string]any)["Carbine"].(map[string]any)
	if carbine["value"] != float64(0) || carbine["name"] != "Unclassified" {
		t.Fatalf("carbine %v", carbine)
	}
	if _, ok := got["classifications"].(map[string]any)["Pistol"]; ok {
		t.Fatalf("unfetched category present: %v", got["classifications"])
	}
	if got["club"] != "CCSC" || got["last_name"] != "ANDERSON" || got["country"] != "Canada" || got["wa1500_synced_at"] == nil {
		t.Fatalf("overwritten fields %v", got)
	}
}

func TestAddEntryDoesNotRefreshFreshShooter(t *testing.T) {
	fake := &fakeRegister{}
	fake.set([]wa1500.Shooter{registerShooter("3289", "ANDERSON", "CCSC", nil)}, nil)
	env := newRegisterEnv(t, fake)
	shooter := createShooter(env, "3289")
	match := newMatch(env)
	other := newMatch(env)
	addToMatch(env, match, shooter, "Revolver", 1)
	env.Advance(23 * time.Hour)
	env.Login()
	addToMatch(env, other, shooter, "Revolver", 1)
	if fake.calls() != 1 {
		t.Fatalf("calls %d", fake.calls())
	}
	env.Advance(2 * time.Hour)
	env.Login()
	third := newMatch(env)
	addToMatch(env, third, shooter, "Revolver", 1)
	if fake.calls() != 2 {
		t.Fatalf("calls after 25h %d", fake.calls())
	}
}

func TestAddEntrySurvivesRegisterFailure(t *testing.T) {
	fake := &fakeRegister{}
	fake.set(nil, wa1500.ErrUpstream)
	env := newRegisterEnv(t, fake)
	shooter := createShooter(env, "3289")
	match := newMatch(env)
	if status, out := addToMatch(env, match, shooter, "Revolver", 1); status != http.StatusCreated {
		t.Fatalf("add: %d %v", status, out)
	}
	_, got := env.Do("GET", fmt.Sprintf("/api/shooters/%d", shooter), nil)
	if got["club"] != "Local" || got["wa1500_synced_at"] != nil {
		t.Fatalf("stored changed after failure: %v", got)
	}
}

func TestRefreshWithoutExactWAIDMatchIsAFailure(t *testing.T) {
	fake := &fakeRegister{}
	fake.set([]wa1500.Shooter{registerShooter("32890", "OTHER", "X", map[string]wa1500.Classification{"Revolver": wa1500.Master})}, nil)
	env := newRegisterEnv(t, fake)
	shooter := createShooter(env, "3289")
	if status, _ := addToMatch(env, newMatch(env), shooter, "Revolver", 1); status != http.StatusCreated {
		t.Fatalf("status %d", status)
	}
	_, got := env.Do("GET", fmt.Sprintf("/api/shooters/%d", shooter), nil)
	if got["last_name"] != "ANDERSON" || got["wa1500_synced_at"] != nil || classOf(t, env, shooter, "Revolver") != nil {
		t.Fatalf("stored %v", got)
	}
}

func TestRefreshKeepsOldValueForMissingOrUnknownClass(t *testing.T) {
	fake := &fakeRegister{}
	fake.set([]wa1500.Shooter{registerShooter("3289", "ANDERSON", "CCSC", map[string]wa1500.Classification{"Revolver": wa1500.Expert, "Pistol": wa1500.Marksman})}, nil)
	env := newRegisterEnv(t, fake)
	shooter := createShooter(env, "3289")
	addToMatch(env, newMatch(env), shooter, "Revolver", 1)

	env.Advance(48 * time.Hour)
	env.Login()
	fake.set([]wa1500.Shooter{registerShooter("3289", "ANDERSON", "CCSC", map[string]wa1500.Classification{"Revolver": wa1500.Master})}, nil)
	addToMatch(env, newMatch(env), shooter, "Revolver", 1)

	if v := classOf(t, env, shooter, "Revolver").(map[string]any)["value"]; v != float64(4) {
		t.Fatalf("revolver %v", v)
	}
	if v := classOf(t, env, shooter, "Pistol").(map[string]any)["value"]; v != float64(1) {
		t.Fatalf("pistol kept %v", v)
	}
}

func TestShooterWithoutWAIDIsNeverRefreshed(t *testing.T) {
	fake := &fakeRegister{}
	fake.set([]wa1500.Shooter{registerShooter("3289", "ANDERSON", "CCSC", nil)}, nil)
	env := newRegisterEnv(t, fake)
	shooter := createShooter(env, "")
	if status, _ := addToMatch(env, newMatch(env), shooter, "Revolver", 1); status != http.StatusCreated || fake.calls() != 0 {
		t.Fatalf("status %d calls %d", status, fake.calls())
	}
}

func TestFailedRefreshIsBackedOff(t *testing.T) {
	fake := &fakeRegister{}
	fake.set(nil, errors.New("boom"))
	env := newRegisterEnv(t, fake)
	shooter := createShooter(env, "3289")
	addToMatch(env, newMatch(env), shooter, "Revolver", 1)
	addToMatch(env, newMatch(env), shooter, "Revolver", 1)
	if fake.calls() != 1 {
		t.Fatalf("calls %d", fake.calls())
	}
	env.Advance(time.Hour)
	env.Login()
	addToMatch(env, newMatch(env), shooter, "Revolver", 1)
	if fake.calls() != 2 {
		t.Fatalf("calls after backoff %d", fake.calls())
	}
}

func TestImportStoresClassifications(t *testing.T) {
	fake := &fakeRegister{}
	fake.set([]wa1500.Shooter{registerShooter("3289", "ANDERSON", "CCSC", map[string]wa1500.Classification{"Pistol OS": wa1500.Sharpshooter})}, nil)
	env := newRegisterEnv(t, fake)
	status, out := env.Do("POST", "/api/shooters/import", importBody(nil))
	if status != http.StatusCreated {
		t.Fatalf("import %d %v", status, out)
	}
	if v := classOf(t, env, int(out["id"].(float64)), "Pistol OS").(map[string]any)["name"]; v != "Sharpshooter" {
		t.Fatalf("pistol os %v", v)
	}
}

func TestImportSurvivesRegisterFailure(t *testing.T) {
	fake := &fakeRegister{}
	fake.set(nil, wa1500.ErrUpstream)
	env := newRegisterEnv(t, fake)
	if status, out := env.Do("POST", "/api/shooters/import", importBody(nil)); status != http.StatusCreated {
		t.Fatalf("import %d %v", status, out)
	}
}

func TestBackgroundBatchRefreshesDueShootersOnly(t *testing.T) {
	fake := &fakeRegister{}
	fake.set([]wa1500.Shooter{registerShooter("3289", "ANDERSON", "CCSC", map[string]wa1500.Classification{"Revolver": wa1500.Expert})}, nil)
	env := newRegisterEnv(t, fake)
	due := createShooter(env, "3289")
	archived := env.ID("POST", "/api/shooters", map[string]any{"last_name": "OLD", "first_name": "A", "country": "N", "club": "C", "waid": "5"})
	noWAID := env.ID("POST", "/api/shooters", map[string]any{"last_name": "NONE", "first_name": "A", "country": "N", "club": "C"})
	env.Do("POST", fmt.Sprintf("/api/shooters/%d/archive", archived), nil)

	env.RunShooterSyncBatch()
	if classOf(t, env, due, "Revolver") == nil {
		t.Fatal("due shooter not refreshed")
	}
	if classOf(t, env, archived, "Revolver") != nil || classOf(t, env, noWAID, "Revolver") != nil {
		t.Fatal("archived or WAID-less shooter refreshed")
	}
	calls := fake.calls()
	env.Advance(6 * 24 * time.Hour)
	env.Login()
	env.RunShooterSyncBatch()
	if fake.calls() != calls {
		t.Fatal("refreshed before 7 days")
	}
	env.Advance(2 * 24 * time.Hour)
	env.Login()
	env.RunShooterSyncBatch()
	if fake.calls() != calls+1 {
		t.Fatalf("calls %d want %d", fake.calls(), calls+1)
	}
}

func TestResultsShowClassForEntryCategory(t *testing.T) {
	fake := &fakeRegister{}
	fake.set([]wa1500.Shooter{registerShooter("1001", "ALPHA", "Club", map[string]wa1500.Classification{"Revolver": wa1500.Master, "Pistol": wa1500.Marksman})}, nil)
	f := newFixture(t, "150")
	f.env = newRegisterEnv(t, fake)
	f.comp = newMatch(f.env)
	revolver := f.addShooterEntry(t, "ALPHA", "1001", "Revolver", 1)
	f.addShooterEntry(t, "Echo", "", "Pistol", 2)
	f.activate(t)
	f.scoreAccepted(t, revolver, 1, map[string]any{"x": 24})
	f.finishActive(t)

	f.env.Do("PUT", fmt.Sprintf("/api/matches/%d/visibility", f.comp), map[string]any{"show_names": true})
	_, public := publicGet(t, f, resultsPath(f))
	pub := public["categories"].([]any)
	class := pub[0].(map[string]any)["rows"].([]any)[0].(map[string]any)["classification"].(map[string]any)
	if class["value"] != float64(4) || class["name"] != "Master" {
		t.Fatalf("revolver row class %v", class)
	}
	if pistol := pub[1].(map[string]any)["rows"].([]any)[0].(map[string]any); pistol["classification"] != nil {
		t.Fatalf("shooter without WAID has class %v", pistol)
	}
}
