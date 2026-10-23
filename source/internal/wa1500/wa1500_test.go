package wa1500

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func latin1(s string) []byte {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		out = append(out, byte(r))
	}
	return out
}

func TestParseResults(t *testing.T) {
	got, err := Parse(bytes.NewReader(fixture(t, "results.html")))
	if err != nil {
		t.Fatal(err)
	}
	want := []Shooter{
		{WAID: "3289", LastName: "ANDERSON", FirstName: "Mike", Country: "Canada", Club: "CCSC"},
		{WAID: "412", LastName: "ÅSEN", FirstName: "Kari Anne", Country: "Norway"},
		{WAID: "77", LastName: "MONONYM", Country: "Sweden", Club: "SSK"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestParseNoResults(t *testing.T) {
	got, err := Parse(bytes.NewReader(fixture(t, "empty.html")))
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func newClient(t *testing.T, h http.HandlerFunc, opts ...ClientOption) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewClient(append([]ClientOption{WithBaseURL(srv.URL)}, opts...)...)
}

func TestClientSearch(t *testing.T) {
	var form url.Values
	var path string
	c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_ = r.ParseForm()
		form = r.PostForm
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		_, _ = w.Write(fixture(t, "results.html"))
	})
	got, err := c.Search(context.Background(), "Åsen")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/class.php" || form.Get("action") != "show" || form.Get("Suche") != "Åsen" || form.Get("Nation") != "" {
		t.Fatalf("request %s %v", path, form)
	}
	if len(got) != 3 || got[1].LastName != "ÅSEN" {
		t.Fatalf("utf-8 not decoded: %+v", got)
	}
}

func TestClientSendsUTF8Query(t *testing.T) {
	var raw string
	c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		raw = string(b)
		_, _ = w.Write(fixture(t, "empty.html"))
	})
	if _, err := c.Search(context.Background(), "Åsen"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, "Suche=%C3%85sen") {
		t.Fatalf("body %q", raw)
	}
}

func TestClientHonoursDeclaredCharset(t *testing.T) {
	c := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=ISO-8859-1")
		_, _ = w.Write(latin1(string(fixture(t, "results.html"))))
	})
	got, err := c.Search(context.Background(), "x")
	if err != nil || len(got) != 3 || got[1].LastName != "ÅSEN" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestClientFailures(t *testing.T) {
	t.Run("status", func(t *testing.T) {
		c := newClient(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) })
		if _, err := c.Search(context.Background(), "x"); !errors.Is(err, ErrUpstream) {
			t.Fatalf("err %v", err)
		}
	})
	t.Run("size cap", func(t *testing.T) {
		c := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(bytes.Repeat([]byte("a"), 2048))
		}, WithMaxBytes(1024))
		if _, err := c.Search(context.Background(), "x"); !errors.Is(err, ErrUpstream) {
			t.Fatalf("err %v", err)
		}
	})
	t.Run("redirect", func(t *testing.T) {
		c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "http://127.0.0.1:1/", http.StatusFound)
		})
		if _, err := c.Search(context.Background(), "x"); !errors.Is(err, ErrUpstream) {
			t.Fatalf("err %v", err)
		}
	})
	t.Run("unreachable", func(t *testing.T) {
		c := NewClient(WithBaseURL("http://127.0.0.1:1"))
		if _, err := c.Search(context.Background(), "x"); !errors.Is(err, ErrUpstream) {
			t.Fatalf("err %v", err)
		}
	})
}

func TestDefaultHostIsFixed(t *testing.T) {
	if NewClient().baseURL != "https://wa1500.org" {
		t.Fatal("unexpected default host")
	}
}
