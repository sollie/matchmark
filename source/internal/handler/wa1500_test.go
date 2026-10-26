package handler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/sollie/matchmark/internal/apitest"
	"github.com/sollie/matchmark/internal/handler"
	"github.com/sollie/matchmark/internal/service"
	"github.com/sollie/matchmark/internal/wa1500"
)

type fakeSearcher struct {
	calls   int
	results []wa1500.Shooter
	err     error
}

func (f *fakeSearcher) Search(context.Context, string) ([]wa1500.Shooter, error) {
	f.calls++
	return f.results, f.err
}

func TestWA1500Search(t *testing.T) {
	fake := &fakeSearcher{results: []wa1500.Shooter{{WAID: "3289", FirstName: "Mike", LastName: "ANDERSON", Country: "Canada", Club: "CCSC"}}}
	env := apitest.NewAdmin(t, handler.WithWA1500(service.NewWA1500Lookup(fake)))

	status, out := env.Do("GET", "/api/wa1500/search?q=anderson", nil)
	results, _ := out["results"].([]any)
	if status != http.StatusOK || len(results) != 1 || results[0].(map[string]any)["waid"] != "3289" {
		t.Fatalf("got %d %v", status, out)
	}
	if status, _ = env.Do("GET", "/api/wa1500/search?q=Anderson", nil); status != http.StatusOK || fake.calls != 1 {
		t.Fatalf("status %d calls %d", status, fake.calls)
	}
	if status, _ = env.Do("GET", "/api/wa1500/search?q=", nil); status != http.StatusUnprocessableEntity {
		t.Fatalf("empty query status %d", status)
	}
}

func TestWA1500SearchUpstreamFailure(t *testing.T) {
	fake := &fakeSearcher{err: wa1500.ErrUpstream}
	env := apitest.NewAdmin(t, handler.WithWA1500(service.NewWA1500Lookup(fake)))
	if status, _ := env.Do("GET", "/api/wa1500/search?q=x", nil); status != http.StatusServiceUnavailable {
		t.Fatalf("status %d", status)
	}
	fake.err = errors.New("boom")
	if status, _ := env.Do("GET", "/api/wa1500/search?q=y", nil); status != http.StatusServiceUnavailable {
		t.Fatalf("status %d", status)
	}
}

func TestWA1500SearchRateLimited(t *testing.T) {
	env := apitest.NewAdmin(t, handler.WithWA1500(service.NewWA1500Lookup(&fakeSearcher{})))
	var last int
	for i := 0; i < 11; i++ {
		last, _ = env.Do("GET", "/api/wa1500/search?q="+string(rune('a'+i)), nil)
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("status %d", last)
	}
}

func TestWA1500SearchRequiresAdmin(t *testing.T) {
	env := apitest.NewWithBaseURL(t, "http://localhost", handler.WithWA1500(service.NewWA1500Lookup(&fakeSearcher{})))
	if status, _ := env.Do("GET", "/api/wa1500/search?q=x", nil); status != http.StatusUnauthorized {
		t.Fatalf("status %d", status)
	}
}
