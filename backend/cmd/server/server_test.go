package main

import "testing"

func TestClientIPFromRemoteAddr(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wanted string
	}{
		{name: "host and port", input: "192.0.2.10:12345", wanted: "192.0.2.10"},
		{name: "ipv6 host and port", input: "[2001:db8::1]:443", wanted: "2001:db8::1"},
		{name: "host only fallback", input: "198.51.100.7", wanted: "198.51.100.7"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := clientIPFromRemoteAddr(tc.input); got != tc.wanted {
				t.Fatalf("clientIPFromRemoteAddr(%q) = %q, want %q", tc.input, got, tc.wanted)
			}
		})
	}
}
