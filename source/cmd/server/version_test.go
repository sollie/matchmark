package main

import (
	"strings"
	"testing"
)

func TestIsVersionRequest(t *testing.T) {
	for args, want := range map[string]bool{"--version": true, "version": true, "restore": false, "": false} {
		var in []string
		if args != "" {
			in = []string{args}
		}
		if got := isVersionRequest(in); got != want {
			t.Errorf("isVersionRequest(%q) = %v, want %v", args, got, want)
		}
	}
	if isVersionRequest([]string{"--version", "x"}) {
		t.Error("extra args must not count as a version request")
	}
}

func TestPrintVersion(t *testing.T) {
	prev := version
	version = "v1.2.3"
	t.Cleanup(func() { version = prev })
	var b strings.Builder
	printVersion(&b)
	if !strings.Contains(b.String(), "v1.2.3") {
		t.Errorf("output %q missing version", b.String())
	}
}
