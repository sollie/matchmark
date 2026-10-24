package apitest

import (
	"bytes"
	"io"
	"net/http"
)

func (e *Env) Raw(method, path string) (int, http.Header, []byte) {
	e.t.Helper()
	req, err := http.NewRequest(method, e.Server.URL+path, nil)
	if err != nil {
		e.t.Fatalf("new request: %v", err)
	}
	resp, err := e.client.Do(req)
	if err != nil {
		e.t.Fatalf("do request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		e.t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, resp.Header, body
}

func (e *Env) RawBody(method, path, contentType string, payload []byte) (int, http.Header, []byte) {
	e.t.Helper()
	req, err := http.NewRequest(method, e.Server.URL+path, bytes.NewReader(payload))
	if err != nil {
		e.t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := e.client.Do(req)
	if err != nil {
		e.t.Fatalf("do request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		e.t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, resp.Header, body
}
