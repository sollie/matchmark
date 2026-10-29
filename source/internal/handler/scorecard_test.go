package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sollie/matchmark/internal/apitest"
)

type scorecardFixture struct {
	*fixture
	entry int
}

func newScorecardFixture(t *testing.T) *scorecardFixture {
	t.Helper()
	return newNamedScorecardFixture(t, "")
}

func newNamedScorecardFixture(t *testing.T, name string) *scorecardFixture {
	t.Helper()
	f := newFixture(t, "150")
	if name != "" {
		if status, out := f.env.Do("PUT", fmt.Sprintf("/api/matches/%d", f.comp), map[string]any{"name": name, "date": "2026-05-01", "course": "150"}); status != http.StatusOK {
			t.Fatalf("rename: %d %v", status, out)
		}
	}
	entry := f.addEntry(t, "HANSEN", 3)
	f.addEntry(t, "BERG", 5)
	base := fmt.Sprintf("/api/matches/%d", f.comp)
	f.env.Do("POST", base+"/status", map[string]any{"status": "active"})
	segment := fmt.Sprintf("%s/entries/%d/segments/1", base, entry)
	if status, out := f.env.Do("PUT", segment, counts112); status != http.StatusOK {
		t.Fatalf("set counts: %d %v", status, out)
	}
	if status, out := f.env.Do("POST", segment+"/lock", nil); status != http.StatusOK {
		t.Fatalf("lock: %d %v", status, out)
	}
	return &scorecardFixture{fixture: f, entry: entry}
}

func (f *scorecardFixture) finish(t *testing.T) {
	t.Helper()
	if status, out := f.env.Do("POST", fmt.Sprintf("/api/matches/%d/status", f.comp), map[string]any{"status": "finished"}); status != http.StatusOK {
		t.Fatalf("finish: %d %v", status, out)
	}
}

func (f *scorecardFixture) path(lane int, query string) string {
	return fmt.Sprintf("/api/scorer/scorecard/%d/%d%s", f.comp, lane, query)
}

func anonymousGet(t *testing.T, env *apitest.Env, path string) (int, http.Header, []byte) {
	t.Helper()
	resp, err := http.Get(env.Server.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, resp.Header, body
}

func TestScorecardIsPublicPNGOnceFinished(t *testing.T) {
	f := newScorecardFixture(t)
	if status, _, _ := anonymousGet(t, f.env, f.path(3, "")); status != http.StatusNotFound {
		t.Fatalf("unfinished: %d", status)
	}
	f.finish(t)
	status, header, body := anonymousGet(t, f.env, f.path(3, ""))
	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	if ct := header.Get("Content-Type"); ct != "image/png" {
		t.Fatalf("content type %q", ct)
	}
	if cc := header.Get("Cache-Control"); cc != "private, no-cache" {
		t.Fatalf("cache control %q", cc)
	}
	if header.Get("Content-Disposition") != "" {
		t.Fatalf("unexpected disposition %q", header.Get("Content-Disposition"))
	}
	img, err := png.Decode(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if b := img.Bounds(); b.Dx() < 400 || b.Dy() < 400 {
		t.Fatalf("size %v", b)
	}
	if status, _, _ := anonymousGet(t, f.env, f.path(3, "?format=png")); status != http.StatusOK {
		t.Fatalf("explicit png: %d", status)
	}
}

func TestScorecardRejectsOtherFormats(t *testing.T) {
	f := newScorecardFixture(t)
	f.finish(t)
	for _, q := range []string{"?format=pdf", "?format=", "?format=PNG"} {
		if status, _, _ := anonymousGet(t, f.env, f.path(3, q)); status != http.StatusUnprocessableEntity {
			t.Fatalf("%s: %d", q, status)
		}
	}
}

func TestScorecardNotFoundForEmptyLaneOrWithdrawnEntry(t *testing.T) {
	f := newScorecardFixture(t)
	f.finish(t)
	if status, _, _ := anonymousGet(t, f.env, f.path(4, "")); status != http.StatusNotFound {
		t.Fatalf("empty lane: %d", status)
	}
	if status, _, _ := anonymousGet(t, f.env, f.path(5, "")); status != http.StatusOK {
		t.Fatalf("unscored entry: %d", status)
	}
	if status, _, _ := anonymousGet(t, f.env, "/api/scorer/scorecard/999/3"); status != http.StatusNotFound {
		t.Fatalf("unknown match: %d", status)
	}
	if status, _, _ := anonymousGet(t, f.env, "/api/scorer/scorecard/x/3"); status != http.StatusNotFound {
		t.Fatalf("bad id: %d", status)
	}
}

func TestScorecardNotFoundForWithdrawnEntry(t *testing.T) {
	f := newFixture(t, "150")
	f.addEntry(t, "HANSEN", 3)
	gone := f.addEntry(t, "BERG", 5)
	base := fmt.Sprintf("/api/matches/%d", f.comp)
	if status, out := f.env.Do("POST", fmt.Sprintf("%s/entries/%d/withdraw", base, gone), nil); status != http.StatusOK {
		t.Fatalf("withdraw: %d %v", status, out)
	}
	f.env.Do("POST", base+"/status", map[string]any{"status": "active"})
	f.env.Do("POST", base+"/status", map[string]any{"status": "finished"})
	if status, _, _ := anonymousGet(t, f.env, fmt.Sprintf("/api/scorer/scorecard/%d/5", f.comp)); status != http.StatusNotFound {
		t.Fatalf("withdrawn: %d", status)
	}
	if status, _, _ := anonymousGet(t, f.env, fmt.Sprintf("/api/scorer/scorecard/%d/3", f.comp)); status != http.StatusOK {
		t.Fatalf("kept: %d", status)
	}
}

func TestScorecardDownloadFilename(t *testing.T) {
	f := newScorecardFixture(t)
	f.finish(t)
	_, header, _ := anonymousGet(t, f.env, f.path(3, "?download=1"))
	got := header.Get("Content-Disposition")
	if got != `attachment; filename="Club-Match-2026-05-01-lane-3.png"` {
		t.Fatalf("disposition %q", got)
	}
}

func TestScorecardReflectsCurrentData(t *testing.T) {
	f := newScorecardFixture(t)
	f.finish(t)
	_, _, before := anonymousGet(t, f.env, f.path(3, ""))
	segment := fmt.Sprintf("/api/matches/%d/entries/%d/segments/1", f.comp, f.entry)
	if status, out := f.env.Do("PUT", segment, map[string]any{"x": 2, "10": 8, "9": 6, "8": 2, "7": 2, "0": 2, "miss": 2}); status != http.StatusOK {
		t.Fatalf("edit: %d %v", status, out)
	}
	_, _, after := anonymousGet(t, f.env, f.path(3, ""))
	if bytes.Equal(before, after) {
		t.Fatal("scorecard did not change after the segment was edited")
	}
}

func TestScorecardWithClubLogo(t *testing.T) {
	f := newScorecardFixture(t)
	f.finish(t)
	if status, _, _ := f.env.RawBody("PUT", "/api/club/logo", "image/png", pngBytes(t, 120, 60)); status != http.StatusNoContent {
		t.Fatalf("logo: %d", status)
	}
	f.env.Do("PUT", "/api/club", map[string]string{"name": "Oslo PPC"})
	status, _, body := anonymousGet(t, f.env, f.path(3, ""))
	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	if _, err := png.Decode(bytes.NewReader(body)); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.HasPrefix(string(body), "\x89PNG") {
		t.Fatal("not a png")
	}
}

func TestScorecardSummaryNamesShooterAndScorers(t *testing.T) {
	f := newScorecardFixture(t)
	if status, _, _ := anonymousGet(t, f.env, f.path(3, "/summary")); status != http.StatusNotFound {
		t.Fatalf("unfinished: %d", status)
	}
	f.finish(t)
	status, header, body := anonymousGet(t, f.env, f.path(3, "/summary"))
	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	if ct := header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content type %q", ct)
	}
	want := `{"lane":3,"scorers":[{"scorer":"Range admin","segment":1,"segment_name":"Match 1"}],"shooter":"A HANSEN"}`
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	canonical, _ := json.Marshal(got)
	if string(canonical) != want {
		t.Fatalf("summary %s", canonical)
	}
	if status, _, _ := anonymousGet(t, f.env, f.path(4, "/summary")); status != http.StatusNotFound {
		t.Fatalf("empty lane: %d", status)
	}
	if status, _, _ := anonymousGet(t, f.env, f.path(5, "/summary")); status != http.StatusOK {
		t.Fatalf("unscored entry: %d", status)
	}
}

func TestScorecardDownloadFilenameWithNonASCIIName(t *testing.T) {
	f := newNamedScorecardFixture(t, "Åre cup")
	f.finish(t)
	_, header, _ := anonymousGet(t, f.env, f.path(3, "?download=1"))
	want := `attachment; filename="_re-cup-2026-05-01-lane-3.png"; filename*=UTF-8''%C3%85re-cup-2026-05-01-lane-3.png`
	if got := header.Get("Content-Disposition"); got != want {
		t.Fatalf("disposition %q", got)
	}
}
