package apitest

import (
	"bufio"
	"net/http"
	"strings"
	"time"
)

type Event struct {
	Name    string
	Data    string
	Comment string
}

type Stream struct {
	Status int
	Header http.Header
	events chan Event
	body   interface{ Close() error }
}

func (e *Env) OpenStream(path string, header http.Header) *Stream {
	e.t.Helper()
	req, err := http.NewRequest(http.MethodGet, e.Server.URL+path, nil)
	if err != nil {
		e.t.Fatalf("new request: %v", err)
	}
	for k, v := range header {
		req.Header[k] = v
	}
	resp, err := e.client.Do(req)
	if err != nil {
		e.t.Fatalf("open stream: %v", err)
	}
	s := &Stream{Status: resp.StatusCode, Header: resp.Header, events: make(chan Event, 64), body: resp.Body}
	e.t.Cleanup(s.Close)
	go func() {
		defer close(s.events)
		sc := bufio.NewScanner(resp.Body)
		var cur Event
		for sc.Scan() {
			line := sc.Text()
			switch {
			case line == "":
				if cur != (Event{}) {
					s.events <- cur
				}
				cur = Event{}
			case strings.HasPrefix(line, ":"):
				cur.Comment = strings.TrimSpace(line[1:])
			case strings.HasPrefix(line, "event:"):
				cur.Name = strings.TrimSpace(line[len("event:"):])
			case strings.HasPrefix(line, "data:"):
				cur.Data = strings.TrimSpace(line[len("data:"):])
			}
		}
	}()
	return s
}

func (s *Stream) Next(timeout time.Duration) (Event, bool) {
	select {
	case ev, ok := <-s.events:
		return ev, ok
	case <-time.After(timeout):
		return Event{}, false
	}
}

func (s *Stream) Closed(timeout time.Duration) bool {
	deadline := time.After(timeout)
	for {
		select {
		case _, ok := <-s.events:
			if !ok {
				return true
			}
		case <-deadline:
			return false
		}
	}
}

func (s *Stream) Close() { _ = s.body.Close() }
