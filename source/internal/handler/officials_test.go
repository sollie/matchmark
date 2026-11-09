package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/sollie/matchmark/internal/apitest"
	"github.com/sollie/matchmark/internal/wa1500"
)

func officialsEnv(t *testing.T) (*apitest.Env, *fakeRegister) {
	t.Helper()
	fake := &fakeRegister{}
	fake.set([]wa1500.Shooter{
		{WAID: "1001", FirstName: "Rita", LastName: "REFEREE", Country: "Norway", Club: "A"},
		{WAID: "2002", FirstName: "Ron", LastName: "RANGE", Country: "Norway", Club: "B"},
	}, nil)
	return newRegisterEnv(t, fake), fake
}

func matchBody(extra map[string]any) map[string]any {
	body := map[string]any{"name": "Club Match", "date": "2026-05-01", "course": "150"}
	for k, v := range extra {
		body[k] = v
	}
	return body
}

func getMatch(t *testing.T, env *apitest.Env, id int) map[string]any {
	t.Helper()
	status, out := env.Do("GET", fmt.Sprintf("/api/matches/%d", id), nil)
	if status != http.StatusOK {
		t.Fatalf("get: %d %v", status, out)
	}
	return out
}

func officialList(m map[string]any, key string) []string {
	var out []string
	list, _ := m[key].([]any)
	for _, v := range list {
		o := v.(map[string]any)
		out = append(out, o["waid"].(string)+"="+o["name"].(string))
	}
	return out
}

func sameList(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestMatchOfficialsAreLookedUpSnapshottedAndKeepOrder(t *testing.T) {
	env, _ := officialsEnv(t)
	status, out := env.Do("POST", "/api/matches", matchBody(map[string]any{
		"referee_waids":       []string{"2002", " 1001 "},
		"range_officer_waids": []string{"2002"},
	}))
	if status != http.StatusCreated {
		t.Fatalf("create: %d %v", status, out)
	}
	m := getMatch(t, env, int(out["id"].(float64)))
	if got := officialList(m, "referees"); !sameList(got, []string{"2002=Ron RANGE", "1001=Rita REFEREE"}) {
		t.Fatalf("referees: %v", got)
	}
	if got := officialList(m, "range_officers"); !sameList(got, []string{"2002=Ron RANGE"}) {
		t.Fatalf("range officers: %v", got)
	}
}

func TestMatchOfficialsRejectDuplicateWAIDWithinRole(t *testing.T) {
	env, _ := officialsEnv(t)
	status, out := env.Do("POST", "/api/matches", matchBody(map[string]any{"referee_waids": []string{"1001", "1001"}}))
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("duplicate: %d %v", status, out)
	}
}

func TestMatchOfficialsAreEmptyByDefault(t *testing.T) {
	env, fake := officialsEnv(t)
	m := getMatch(t, env, newMatch(env))
	if len(m["referees"].([]any)) != 0 || len(m["range_officers"].([]any)) != 0 {
		t.Fatalf("officials: %v", m)
	}
	if fake.calls() != 0 {
		t.Fatalf("lookups: %d", fake.calls())
	}
}

func TestOneDayCompetitionRefereeDefaultsToSignedInUserAndMatchInherits(t *testing.T) {
	env, _ := officialsEnv(t)
	if status, out := env.Do("PUT", "/api/admin/waid", map[string]any{"waid": "1001"}); status != http.StatusOK || out["waid"] != "1001" {
		t.Fatalf("set waid: %d %v", status, out)
	}
	m := getMatch(t, env, newMatch(env))
	if len(m["referees"].([]any)) != 0 {
		t.Fatalf("match must start with no officials of its own: %v", m)
	}
	if got := officialList(m, "effective_referees"); !sameList(got, []string{"1001=Rita REFEREE"}) || len(m["effective_range_officers"].([]any)) != 0 {
		t.Fatalf("inherited: %v", m)
	}
	_, comp := env.Do("GET", fmt.Sprintf("/api/competitions/%d", int(m["competition_id"].(float64))), nil)
	if got := officialList(comp, "referees"); !sameList(got, []string{"1001=Rita REFEREE"}) {
		t.Fatalf("competition: %v", comp)
	}
}

func competitionBodyWith(extra map[string]any) map[string]any {
	body := map[string]any{"name": "Autumn Open", "start_date": "2026-05-01", "end_date": "2026-05-03"}
	for k, v := range extra {
		body[k] = v
	}
	return body
}

func TestCompetitionOfficialsAreStoredInOrderAndDefaultToSignedInUser(t *testing.T) {
	env, _ := officialsEnv(t)
	id := env.ID("POST", "/api/competitions", competitionBodyWith(map[string]any{
		"referee_waids": []string{"2002", "1001"}, "range_officer_waids": []string{"1001"},
	}))
	path := fmt.Sprintf("/api/competitions/%d", id)
	_, c := env.Do("GET", path, nil)
	if !sameList(officialList(c, "referees"), []string{"2002=Ron RANGE", "1001=Rita REFEREE"}) || !sameList(officialList(c, "range_officers"), []string{"1001=Rita REFEREE"}) {
		t.Fatalf("competition: %v", c)
	}
	if status, out := env.Do("PUT", path, competitionBodyWith(map[string]any{"referee_waids": []string{"1001"}})); status != http.StatusOK || !sameList(officialList(out, "referees"), []string{"1001=Rita REFEREE"}) || len(out["range_officers"].([]any)) != 1 {
		t.Fatalf("update: %d %v", status, out)
	}
	if status, out := env.Do("PUT", path, competitionBodyWith(map[string]any{"referee_waids": []string{}, "range_officer_waids": []string{}})); status != http.StatusOK || len(out["referees"].([]any)) != 0 || len(out["range_officers"].([]any)) != 0 {
		t.Fatalf("cleared: %d %v", status, out)
	}

	env.Do("PUT", "/api/admin/waid", map[string]any{"waid": "2002"})
	_, c = env.Do("GET", fmt.Sprintf("/api/competitions/%d", env.ID("POST", "/api/competitions", competitionBodyWith(nil))), nil)
	if !sameList(officialList(c, "referees"), []string{"2002=Ron RANGE"}) || len(c["range_officers"].([]any)) != 0 {
		t.Fatalf("default: %v", c)
	}
	_, c = env.Do("GET", fmt.Sprintf("/api/competitions/%d", env.ID("POST", "/api/competitions", competitionBodyWith(map[string]any{"referee_waids": []string{}}))), nil)
	if len(c["referees"].([]any)) != 0 {
		t.Fatalf("explicit empty: %v", c)
	}
}

func TestCompetitionOfficialLookupFailureStoresNothing(t *testing.T) {
	env, fake := officialsEnv(t)
	status, out := env.Do("POST", "/api/competitions", competitionBodyWith(map[string]any{"referee_waids": []string{"9999"}}))
	if status != http.StatusUnprocessableEntity || out["error"] != "WAID not found in the WA1500 register" {
		t.Fatalf("not found: %d %v", status, out)
	}
	if status, _ := env.Do("POST", "/api/competitions", competitionBodyWith(map[string]any{"referee_waids": []string{"1001", "1001"}})); status != http.StatusUnprocessableEntity {
		t.Fatalf("duplicate: %d", status)
	}
	if _, list := env.Do("GET", "/api/competitions", nil); len(list["competitions"].([]any)) != 0 {
		t.Fatalf("partial competition created: %v", list)
	}
	id := env.ID("POST", "/api/competitions", competitionBodyWith(map[string]any{"referee_waids": []string{"1001"}}))
	fake.set(nil, wa1500.ErrUpstream)
	path := fmt.Sprintf("/api/competitions/%d", id)
	if status, _ := env.Do("PUT", path, competitionBodyWith(map[string]any{"name": "Renamed", "referee_waids": []string{"1001", "2002"}})); status != http.StatusServiceUnavailable {
		t.Fatalf("unreachable: %d", status)
	}
	_, c := env.Do("GET", path, nil)
	if c["name"] != "Autumn Open" || !sameList(officialList(c, "referees"), []string{"1001=Rita REFEREE"}) {
		t.Fatalf("partial update: %v", c)
	}
	if status, out := env.Do("PUT", path, competitionBodyWith(map[string]any{"name": "Renamed"})); status != http.StatusOK || !sameList(officialList(out, "referees"), []string{"1001=Rita REFEREE"}) {
		t.Fatalf("omitted keeps: %d %v", status, out)
	}
}

func TestMatchInheritsCompetitionOfficialsPerRoleAndOverrides(t *testing.T) {
	env, _ := officialsEnv(t)
	compID := env.ID("POST", "/api/competitions", competitionBodyWith(map[string]any{
		"referee_waids": []string{"1001"}, "range_officer_waids": []string{"1001"},
	}))
	matchID := env.ID("POST", "/api/matches", matchBody(map[string]any{"competition_id": compID, "date": "2026-05-02"}))
	m := getMatch(t, env, matchID)
	if len(m["referees"].([]any)) != 0 || !sameList(officialList(m, "effective_referees"), []string{"1001=Rita REFEREE"}) || !sameList(officialList(m, "effective_range_officers"), []string{"1001=Rita REFEREE"}) {
		t.Fatalf("inherit: %v", m)
	}

	path := fmt.Sprintf("/api/matches/%d", matchID)
	m = func() map[string]any {
		_, out := env.Do("PUT", path, matchBody(map[string]any{"date": "2026-05-02", "referee_waids": []string{"2002"}}))
		return out
	}()
	if !sameList(officialList(m, "effective_referees"), []string{"2002=Ron RANGE"}) || !sameList(officialList(m, "effective_range_officers"), []string{"1001=Rita REFEREE"}) {
		t.Fatalf("override one role: %v", m)
	}

	env.Do("PUT", fmt.Sprintf("/api/competitions/%d", compID), competitionBodyWith(map[string]any{"referee_waids": []string{"1001", "2002"}, "range_officer_waids": []string{"2002"}}))
	m = getMatch(t, env, matchID)
	if !sameList(officialList(m, "effective_referees"), []string{"2002=Ron RANGE"}) || !sameList(officialList(m, "effective_range_officers"), []string{"2002=Ron RANGE"}) {
		t.Fatalf("competition change: %v", m)
	}

	_, out := env.Do("PUT", path, matchBody(map[string]any{"date": "2026-05-02", "referee_waids": []string{}}))
	if !sameList(officialList(out, "effective_referees"), []string{"1001=Rita REFEREE", "2002=Ron RANGE"}) {
		t.Fatalf("cleared override: %v", out)
	}
}

func TestCompetitionOfficialsAreAudited(t *testing.T) {
	env, _ := officialsEnv(t)
	id := env.ID("POST", "/api/competitions", competitionBodyWith(map[string]any{"referee_waids": []string{"1001", "2002"}}))
	env.Do("PUT", fmt.Sprintf("/api/competitions/%d", id), competitionBodyWith(map[string]any{"referee_waids": []string{}, "range_officer_waids": []string{"2002"}}))
	seen := map[string]bool{}
	for _, line := range env.AuditLines() {
		d, _ := line["details"].(string)
		seen[line["action"].(string)+": "+d] = true
	}
	for _, want := range []string{
		"competition.create: Autumn Open, referees 1001, 2002",
		"competition.update: Autumn Open, referees cleared, range officers 2002",
	} {
		if !seen[want] {
			t.Fatalf("missing %q in %v", want, seen)
		}
	}
}

func TestMatchOfficialLookupFailureStoresNothing(t *testing.T) {
	env, fake := officialsEnv(t)
	id := env.ID("POST", "/api/matches", matchBody(map[string]any{"referee_waids": []string{"1001"}}))

	status, out := env.Do("POST", "/api/matches", matchBody(map[string]any{"range_officer_waids": []string{"2002", "9999"}}))
	if status != http.StatusUnprocessableEntity || out["error"] != "WAID not found in the WA1500 register" {
		t.Fatalf("not found: %d %v", status, out)
	}
	if _, list := env.Do("GET", "/api/matches", nil); len(list["matches"].([]any)) != 1 {
		t.Fatalf("partial match created: %v", list)
	}

	fake.set(nil, wa1500.ErrUpstream)
	path := fmt.Sprintf("/api/matches/%d", id)
	status, out = env.Do("PUT", path, matchBody(map[string]any{"name": "Renamed", "referee_waids": []string{"1001"}, "range_officer_waids": []string{"2002"}}))
	if status != http.StatusServiceUnavailable {
		t.Fatalf("unreachable: %d %v", status, out)
	}
	m := getMatch(t, env, id)
	if m["name"] != "Club Match" || len(m["range_officers"].([]any)) != 0 || !sameList(officialList(m, "referees"), []string{"1001=Rita REFEREE"}) {
		t.Fatalf("partial update: %v", m)
	}
}

func TestMatchOfficialsAddRemoveReorderAndClear(t *testing.T) {
	env, fake := officialsEnv(t)
	id := env.ID("POST", "/api/matches", matchBody(map[string]any{"referee_waids": []string{"1001"}}))
	fake.set(nil, wa1500.ErrUpstream)
	path := fmt.Sprintf("/api/matches/%d", id)
	if status, out := env.Do("PUT", path, matchBody(map[string]any{"name": "Renamed"})); status != http.StatusOK || !sameList(officialList(out, "referees"), []string{"1001=Rita REFEREE"}) {
		t.Fatalf("omitted: %d %v", status, out)
	}
	fake.set([]wa1500.Shooter{{WAID: "2002", FirstName: "Ron", LastName: "RANGE"}}, nil)
	if status, out := env.Do("PUT", path, matchBody(map[string]any{"referee_waids": []string{"2002", "1001"}})); status != http.StatusOK || !sameList(officialList(out, "referees"), []string{"2002=Ron RANGE", "1001=Rita REFEREE"}) {
		t.Fatalf("add first: %d %v", status, out)
	}
	if status, out := env.Do("PUT", path, matchBody(map[string]any{"referee_waids": []string{"2002"}})); status != http.StatusOK || !sameList(officialList(out, "referees"), []string{"2002=Ron RANGE"}) {
		t.Fatalf("remove: %d %v", status, out)
	}
	if status, out := env.Do("PUT", path, matchBody(map[string]any{"referee_waids": []string{}})); status != http.StatusOK || len(out["referees"].([]any)) != 0 {
		t.Fatalf("cleared: %d %v", status, out)
	}
}

func TestMatchOfficialsEditableAfterMatchStarts(t *testing.T) {
	env, _ := officialsEnv(t)
	id := newMatch(env)
	if status, out := env.Do("POST", fmt.Sprintf("/api/matches/%d/status", id), map[string]any{"status": "active"}); status != http.StatusOK {
		t.Fatalf("activate: %d %v", status, out)
	}
	status, out := env.Do("PUT", fmt.Sprintf("/api/matches/%d", id), matchBody(map[string]any{"range_officer_waids": []string{"2002"}}))
	if status != http.StatusOK || !sameList(officialList(out, "range_officers"), []string{"2002=Ron RANGE"}) {
		t.Fatalf("active edit: %d %v", status, out)
	}
}

func TestMatchOfficialsAreAudited(t *testing.T) {
	env, _ := officialsEnv(t)
	id := env.ID("POST", "/api/matches", matchBody(map[string]any{"referee_waids": []string{"1001", "2002"}}))
	env.Do("PUT", fmt.Sprintf("/api/matches/%d", id), matchBody(map[string]any{"referee_waids": []string{"2002"}, "range_officer_waids": []string{"1001"}}))
	seen := map[string]bool{}
	for _, line := range env.AuditLines() {
		d, _ := line["details"].(string)
		seen[line["action"].(string)+": "+d] = true
	}
	for _, want := range []string{
		"match.create: Club Match, referees 1001, 2002",
		"match.update: Club Match, referees 2002, range officers 1001",
	} {
		if !seen[want] {
			t.Fatalf("missing %q in %v", want, seen)
		}
	}
}

func TestUserWAIDIsOptionalUniqueAndAdminManaged(t *testing.T) {
	env, _ := officialsEnv(t)
	status, out := env.Do("GET", "/api/users", nil)
	if status != http.StatusOK || out["users"].([]any)[0].(map[string]any)["waid"] != nil {
		t.Fatalf("list: %d %v", status, out)
	}
	status, out = env.Do("POST", "/api/users", map[string]any{"username": "second", "role": "user", "password": "a-long-enough-password"})
	if status != http.StatusCreated {
		t.Fatalf("create: %d %v", status, out)
	}
	second := int(out["id"].(float64))
	path := fmt.Sprintf("/api/users/%d/waid", second)
	if status, out := env.Do("PUT", path, map[string]any{"waid": " 3003 "}); status != http.StatusOK || out["waid"] != "3003" {
		t.Fatalf("set: %d %v", status, out)
	}
	if status, _ := env.Do("PUT", "/api/admin/waid", map[string]any{"waid": "3003"}); status != http.StatusConflict {
		t.Fatalf("duplicate: %d", status)
	}
	if status, _ := env.Do("PUT", path, map[string]any{"waid": string(make([]byte, 33))}); status != http.StatusUnprocessableEntity {
		t.Fatalf("too long: %d", status)
	}
	if status, out := env.Do("PUT", path, map[string]any{"waid": ""}); status != http.StatusOK || out["waid"] != nil {
		t.Fatalf("clear: %d %v", status, out)
	}
	if status, out := env.Do("PUT", "/api/admin/waid", map[string]any{"waid": "3003"}); status != http.StatusOK || out["waid"] != "3003" {
		t.Fatalf("own: %d %v", status, out)
	}
	if _, session := env.Do("GET", "/api/admin/session", nil); session["waid"] != "3003" {
		t.Fatalf("session: %v", session)
	}
}

func TestOfficialCandidatesListActiveUsersWithAWAID(t *testing.T) {
	env, _ := officialsEnv(t)
	if status, _ := env.Do("PUT", "/api/admin/waid", map[string]any{"waid": "1001"}); status != http.StatusOK {
		t.Fatalf("own waid: %d", status)
	}
	var ids []int
	for _, name := range []string{"second", "third", "fourth"} {
		status, out := env.Do("POST", "/api/users", map[string]any{"username": name, "role": "user", "password": "a-long-enough-password"})
		if status != http.StatusCreated {
			t.Fatalf("create %s: %d %v", name, status, out)
		}
		ids = append(ids, int(out["id"].(float64)))
	}
	env.Do("PUT", fmt.Sprintf("/api/users/%d/waid", ids[0]), map[string]any{"waid": "2002"})
	env.Do("PUT", fmt.Sprintf("/api/users/%d/waid", ids[1]), map[string]any{"waid": "3003"})
	env.Do("POST", fmt.Sprintf("/api/users/%d/deactivate", ids[1]), nil)

	status, out := env.Do("GET", "/api/officials/candidates", nil)
	if status != http.StatusOK {
		t.Fatalf("candidates: %d %v", status, out)
	}
	var got []string
	for _, v := range out["candidates"].([]any) {
		c := v.(map[string]any)
		got = append(got, c["waid"].(string)+"="+c["username"].(string))
	}
	if len(got) != 2 || got[0] == got[1] {
		t.Fatalf("candidates: %v", got)
	}
	seen := map[string]bool{got[0]: true, got[1]: true}
	if !seen["2002=second"] || !seen["1001="+apitest.AdminUsername] {
		t.Fatalf("candidates: %v", got)
	}
}
