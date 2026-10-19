package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestLoggerWritesJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	h := requestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("nope"))
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/thing?token=secret", nil)
	req.RemoteAddr = "203.0.113.7:4242"
	h.ServeHTTP(httptest.NewRecorder(), req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("not json: %v: %s", err, buf.String())
	}
	if entry["method"] != "GET" || entry["path"] != "/api/thing" || entry["status"] != float64(404) ||
		entry["bytes"] != float64(4) || entry["remote"] != "203.0.113.7" || entry["level"] != "WARN" {
		t.Fatalf("unexpected entry: %v", entry)
	}
	if bytes.Contains(buf.Bytes(), []byte("secret")) {
		t.Fatal("query string must not be logged")
	}
}
