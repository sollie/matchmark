package main

import (
	"compress/flate"
	"compress/gzip"
	"github.com/andybalholm/brotli"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadConfigTLSMode(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		mode    TLSMode
		addr    string
		wantErr bool
	}{
		{"default plain", map[string]string{}, TLSPlain, ":8080", false},
		{"static", map[string]string{"TLS_CERT_FILE": "c", "TLS_KEY_FILE": "k"}, TLSStatic, ":8443", false},
		{"acme", map[string]string{"ACME_DOMAIN": "x.example", "ACME_EMAIL": "a@x.example"}, TLSACME, ":8443", false},
		{"explicit addr", map[string]string{"LISTEN_ADDR": ":9000"}, TLSPlain, ":9000", false},
		{"cert without key", map[string]string{"TLS_CERT_FILE": "c"}, "", "", true},
		{"key without cert", map[string]string{"TLS_KEY_FILE": "k"}, "", "", true},
		{"acme without email", map[string]string{"ACME_DOMAIN": "x.example"}, "", "", true},
		{"static and acme", map[string]string{"TLS_CERT_FILE": "c", "TLS_KEY_FILE": "k", "ACME_DOMAIN": "x.example", "ACME_EMAIL": "a@x"}, "", "", true},
		{"bad proxy", map[string]string{"TRUSTED_PROXY": "nope"}, "", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, err := LoadConfig(env(tc.env))
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if err == nil && (c.TLSMode != tc.mode || c.ListenAddr != tc.addr) {
				t.Fatalf("mode %q addr %q, want %q %q", c.TLSMode, c.ListenAddr, tc.mode, tc.addr)
			}
		})
	}
}

func TestTrustedProxyRealIP(t *testing.T) {
	proxies, err := parseTrustedProxies("10.0.0.0/8, 127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, remote, xff, want string
	}{
		{"untrusted peer ignores header", "203.0.113.9:1234", "1.2.3.4", "203.0.113.9:1234"},
		{"trusted peer uses header", "10.1.1.1:1234", "198.51.100.7", "198.51.100.7:0"},
		{"spoofed leftmost ignored", "10.1.1.1:1234", "6.6.6.6, 198.51.100.7", "198.51.100.7:0"},
		{"trusted hops skipped", "10.1.1.1:1234", "198.51.100.7, 10.2.2.2", "198.51.100.7:0"},
		{"garbage header ignored", "10.1.1.1:1234", "garbage", "10.1.1.1:1234"},
		{"no header", "10.1.1.1:1234", "", "10.1.1.1:1234"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			h := trustedProxyRealIP(proxies)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { got = r.RemoteAddr }))
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tc.remote
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestSPAHandler(t *testing.T) {
	dist := fstest.MapFS{
		"index.html":    {Data: []byte("<html>app</html>")},
		"assets/app.js": {Data: []byte("js")},
	}
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "api") })
	srv := httptest.NewServer(spaHandler(dist, api))
	defer srv.Close()
	tests := []struct {
		method, path string
		status       int
		body         string
	}{
		{"GET", "/", 200, "app"},
		{"GET", "/competitions/3", 200, "app"},
		{"GET", "/assets/app.js", 200, "js"},
		{"GET", "/assets/missing.js", 404, ""},
		{"GET", "/api/anything", 200, "api"},
		{"POST", "/api/anything", 200, "api"},
		{"POST", "/other", 405, ""},
	}
	for _, tc := range tests {
		req, _ := http.NewRequest(tc.method, srv.URL+tc.path, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != tc.status || !strings.Contains(string(b), tc.body) {
			t.Errorf("%s %s: %d %q", tc.method, tc.path, resp.StatusCode, b)
		}
	}
}

func TestSPAHandlerWithoutBuild(t *testing.T) {
	rec := httptest.NewRecorder()
	spaHandler(fstest.MapFS{".gitkeep": {}}, http.NotFoundHandler()).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestLoadConfigReplication(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		url      string
		interval time.Duration
		wantErr  bool
	}{
		{"disabled by default", map[string]string{}, "", time.Minute, false},
		{"enabled with default interval", map[string]string{"REPLICATION_URL": "http://10.0.0.2:8080"}, "http://10.0.0.2:8080", time.Minute, false},
		{"custom interval", map[string]string{"REPLICATION_URL": "http://10.0.0.2:8080", "REPLICATION_INTERVAL": "15s"}, "http://10.0.0.2:8080", 15 * time.Second, false},
		{"unparsable interval", map[string]string{"REPLICATION_INTERVAL": "soon"}, "", 0, true},
		{"non-positive interval", map[string]string{"REPLICATION_INTERVAL": "0s"}, "", 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, err := LoadConfig(env(tc.env))
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if err == nil && (c.ReplicationURL != tc.url || c.ReplicationInterval != tc.interval) {
				t.Fatalf("url %q interval %v", c.ReplicationURL, c.ReplicationInterval)
			}
		})
	}
}

func TestCompression(t *testing.T) {
	body := strings.Repeat("hello matchmark ", 100)
	h := compression(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/small":
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Length", "2")
			_, _ = w.Write([]byte("hi"))
		case "/xlsx":
			w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
			_, _ = w.Write([]byte(body))
		default:
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte(body))
		}
	}))
	tests := []struct {
		name, path, accept, want string
		method                   string
	}{
		{"br preferred", "/", "gzip, deflate, br", "br", "GET"},
		{"gzip", "/", "gzip", "gzip", "GET"},
		{"deflate", "/", "deflate", "deflate", "GET"},
		{"q-value wins", "/", "br;q=0.1, gzip;q=0.9", "gzip", "GET"},
		{"q=0 refuses", "/", "br;q=0, gzip;q=0", "", "GET"},
		{"none", "/", "", "", "GET"},
		{"small skipped", "/small", "gzip", "", "GET"},
		{"xlsx skipped", "/xlsx", "gzip", "", "GET"},
		{"head skipped", "/", "gzip", "", "HEAD"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.accept != "" {
				req.Header.Set("Accept-Encoding", tc.accept)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			res := rec.Result()
			if got := res.Header.Get("Content-Encoding"); got != tc.want {
				t.Fatalf("Content-Encoding = %q, want %q", got, tc.want)
			}
			if tc.want == "" {
				return
			}
			if res.Header.Get("Vary") != "Accept-Encoding" || res.Header.Get("Content-Length") != "" {
				t.Fatalf("bad headers: %v", res.Header)
			}
			var r io.Reader
			switch tc.want {
			case "br":
				r = brotli.NewReader(res.Body)
			case "gzip":
				zr, err := gzip.NewReader(res.Body)
				if err != nil {
					t.Fatal(err)
				}
				r = zr
			default:
				r = flate.NewReader(res.Body)
			}
			got, err := io.ReadAll(r)
			if err != nil || string(got) != body {
				t.Fatalf("round trip failed: %v", err)
			}
		})
	}
}

func TestCompressionDisabled(t *testing.T) {
	h := compression(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(strings.Repeat("x", 1000)))
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Content-Encoding") != "" {
		t.Fatal("compressed while disabled")
	}
}
