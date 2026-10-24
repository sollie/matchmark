package handler_test

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/sollie/matchmark/internal/handler"
)

func publicPDFPath(f *fixture, query string) string {
	return fmt.Sprintf("/api/competitions/%d/results%s", f.comp, query)
}

func TestPublicResultsPDF(t *testing.T) {
	f := resultsFixture(t)
	for _, query := range []string{"", "?format=pdf"} {
		status, header, body := anonymousGet(t, f.env, publicPDFPath(f, query))
		if status != http.StatusOK {
			t.Fatalf("%q: %d", query, status)
		}
		if header.Get("Content-Type") != "application/pdf" || header.Get("Cache-Control") != "private, no-cache" {
			t.Fatalf("%q headers: %v", query, header)
		}
		if got := header.Get("Content-Disposition"); got != `attachment; filename="Club-Match-2026-05-01-results.pdf"` {
			t.Fatalf("disposition %q", got)
		}
		if !bytes.HasPrefix(body, []byte("%PDF-")) {
			t.Fatalf("%q: not a PDF", query)
		}
	}
}

func TestPublicResultsPDFRejectsOtherFormats(t *testing.T) {
	f := resultsFixture(t)
	if status, _, _ := anonymousGet(t, f.env, publicPDFPath(f, "?format=png")); status != http.StatusUnprocessableEntity {
		t.Fatalf("status %d", status)
	}
}

func TestPublicResultsPDFOnlyForFinishedCompetitions(t *testing.T) {
	f := newFixture(t, "150")
	f.addEntry(t, "One", 1)
	if status, _, _ := anonymousGet(t, f.env, publicPDFPath(f, "")); status != http.StatusNotFound {
		t.Fatalf("draft: %d", status)
	}
	if status, _, _ := anonymousGet(t, f.env, "/api/competitions/999/results"); status != http.StatusNotFound {
		t.Fatalf("unknown: %d", status)
	}
}

func TestPublicResultsPDFHonoursVisibility(t *testing.T) {
	f := resultsFixture(t)
	_, _, shown := anonymousGet(t, f.env, publicPDFPath(f, ""))
	off := map[string]any{"show_names": false, "show_clubs": false, "show_countries": false, "show_stage_scores": false, "show_ring_counts": false}
	if status, out := f.env.Do("PUT", fmt.Sprintf("/api/competitions/%d/visibility", f.comp), off); status != http.StatusOK {
		t.Fatalf("visibility: %d %v", status, out)
	}
	_, _, hidden := anonymousGet(t, f.env, publicPDFPath(f, ""))
	if len(hidden) >= len(shown) {
		t.Fatalf("hidden PDF is %d bytes, shown %d", len(hidden), len(shown))
	}
	status, _, admin := f.env.Raw("GET", fmt.Sprintf("/api/admin/competitions/%d/results", f.comp))
	if status != http.StatusOK || len(admin) <= len(hidden) {
		t.Fatalf("admin PDF: status %d, %d bytes, hidden %d", status, len(admin), len(hidden))
	}
}

func TestAdminResultsPDFRequiresLoginAndIsAudited(t *testing.T) {
	f := resultsFixture(t)
	path := fmt.Sprintf("/api/admin/competitions/%d/results", f.comp)
	if status, _, _ := anonymousGet(t, f.env, path); status != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", status)
	}
	status, header, body := f.env.Raw("GET", path+"?format=pdf")
	if status != http.StatusOK || header.Get("Content-Type") != "application/pdf" || !bytes.HasPrefix(body, []byte("%PDF-")) {
		t.Fatalf("admin: %d %v", status, header)
	}
	if header.Get("Cache-Control") != "private, no-cache" {
		t.Fatalf("cache: %v", header)
	}
	if status, _, _ := f.env.Raw("GET", path+"?format=csv"); status != http.StatusUnprocessableEntity {
		t.Fatalf("format: %d", status)
	}
	rows := auditEntries(t, f.env, "?action=report.download")
	if len(rows) != 1 || rows[0]["details"] != "pdf" {
		t.Fatalf("audit: %v", rows)
	}
}

func TestAdminResultsPDFIsRateLimitedAndAuditedPerRequest(t *testing.T) {
	guard := handler.NewRenderGuard(handler.RenderLimits{Concurrent: 4, Queue: 4, PerResource: 2})
	f := resultsFixture(t, handler.WithRenderGuard(guard))
	path := fmt.Sprintf("/api/admin/competitions/%d/results", f.comp)
	for i := 0; i < 2; i++ {
		if status, _, _ := f.env.Raw("GET", path); status != http.StatusOK {
			t.Fatalf("request %d: %d", i, status)
		}
	}
	status, header, _ := f.env.Raw("GET", path)
	if status != http.StatusTooManyRequests || header.Get("Retry-After") == "" {
		t.Fatalf("over limit: %d %v", status, header)
	}
	if rows := auditEntries(t, f.env, "?action=report.download"); len(rows) != 2 {
		t.Fatalf("audit: %v", rows)
	}
}
