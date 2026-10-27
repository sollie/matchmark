package handler_test

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/sollie/matchmark/internal/apitest"
)

func zipNames(t *testing.T, body []byte) map[string]*zip.File {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	out := map[string]*zip.File{}
	for _, f := range zr.File {
		out[f.Name] = f
	}
	return out
}

func sortedKeys(m map[string]*zip.File) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func seedDownloadCompetition(t *testing.T, env *apitest.Env) int {
	t.Helper()
	comp := env.ID("POST", "/api/competitions", map[string]any{"name": "Nationals", "start_date": "2026-05-01", "end_date": "2026-05-02"})
	finish := func(name, course string) {
		id := env.ID("POST", "/api/matches", map[string]any{"name": name, "date": "2026-05-01", "course": course, "competition_id": comp})
		for _, s := range []string{"active", "finished"} {
			if status, out := env.Do("POST", fmt.Sprintf("/api/matches/%d/status", id), map[string]any{"status": s}); status != http.StatusOK {
				t.Fatalf("status %s: %d %v", s, status, out)
			}
		}
	}
	finish("Day one", "150")
	finish("Day two", "40")
	env.ID("POST", "/api/matches", map[string]any{"name": "Day three", "date": "2026-05-02", "course": "150", "competition_id": comp})
	return comp
}

func TestCompetitionDownloadBundles(t *testing.T) {
	env := apitest.NewAdmin(t)
	comp := seedDownloadCompetition(t, env)
	url := func(format string) string {
		return fmt.Sprintf("/api/competitions/%d/results.zip?format=%s", comp, format)
	}

	for format, want := range map[string][]string{
		"xlsx": {"Day-one-2026-05-01-150.xlsx", "Day-two-2026-05-01-40.xlsx"},
		"csv":  {"Day-one-2026-05-01-150.csv", "Day-two-2026-05-01-40.csv"},
		"pdf":  {"Day-one-2026-05-01-150.pdf", "Day-two-2026-05-01-40.pdf"},
		"all": {
			"Day-one-2026-05-01-150/Day-one-2026-05-01-150.csv", "Day-one-2026-05-01-150/Day-one-2026-05-01-150.pdf", "Day-one-2026-05-01-150/Day-one-2026-05-01-150.xlsx",
			"Day-two-2026-05-01-40/Day-two-2026-05-01-40.csv", "Day-two-2026-05-01-40/Day-two-2026-05-01-40.pdf", "Day-two-2026-05-01-40/Day-two-2026-05-01-40.xlsx",
		},
	} {
		t.Run(format, func(t *testing.T) {
			status, header, body := env.Raw("GET", url(format))
			if status != http.StatusOK {
				t.Fatalf("status %d", status)
			}
			if ct := header.Get("Content-Type"); ct != "application/zip" {
				t.Fatalf("content type %q", ct)
			}
			if cd := header.Get("Content-Disposition"); cd != `attachment; filename="Nationals-2026-05-01-`+format+`.zip"` {
				t.Fatalf("disposition %q", cd)
			}
			files := zipNames(t, body)
			want = append(want, "SKIPPED.txt")
			sort.Strings(want)
			if got := sortedKeys(files); strings.Join(got, "|") != strings.Join(want, "|") {
				t.Fatalf("files %v, want %v", got, want)
			}
			rc, err := files["SKIPPED.txt"].Open()
			if err != nil {
				t.Fatal(err)
			}
			var skipped bytes.Buffer
			_, _ = skipped.ReadFrom(rc)
			_ = rc.Close()
			if text := skipped.String(); !strings.Contains(text, "Day three") || strings.Contains(text, "Day one") {
				t.Fatalf("skipped list %q", text)
			}
		})
	}

	if lines := auditEntries(t, env, "?action=report.download"); len(lines) != 4 {
		t.Fatalf("audit entries %v", lines)
	}
}

func TestCompetitionDownloadWithoutSkippedMatches(t *testing.T) {
	env := apitest.NewAdmin(t)
	comp := env.ID("POST", "/api/competitions", map[string]any{"name": "Empty", "start_date": "2026-05-01", "end_date": "2026-05-01"})
	status, _, body := env.Raw("GET", fmt.Sprintf("/api/competitions/%d/results.zip?format=csv", comp))
	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	if files := zipNames(t, body); len(files) != 0 {
		t.Fatalf("files %v", sortedKeys(files))
	}
}

func TestCompetitionDownloadRejectsBadRequests(t *testing.T) {
	env := apitest.NewAdmin(t)
	comp := seedDownloadCompetition(t, env)
	for path, want := range map[string]int{
		fmt.Sprintf("/api/competitions/%d/results.zip", comp):               http.StatusUnprocessableEntity,
		fmt.Sprintf("/api/competitions/%d/results.zip?format=docx", comp):   http.StatusUnprocessableEntity,
		"/api/competitions/99999/results.zip?format=csv":                    http.StatusNotFound,
		fmt.Sprintf("/api/competitions/%d/results.zip?format=all", comp+99): http.StatusNotFound,
	} {
		if status, _, _ := env.Raw("GET", path); status != want {
			t.Errorf("%s: status %d, want %d", path, status, want)
		}
	}
}
