package service

import "testing"

func TestValidPalette(t *testing.T) {
	for _, id := range []string{"matchmark", "arctic-blue", "graphite-lime"} {
		if !ValidPalette(id) {
			t.Errorf("%q should be valid", id)
		}
	}
	for _, id := range []string{"", "Matchmark", "neon", "arctic-blue "} {
		if ValidPalette(id) {
			t.Errorf("%q should be invalid", id)
		}
	}
	if !ValidPalette(DefaultPalette) || DefaultPalette != "matchmark" {
		t.Errorf("default palette %q", DefaultPalette)
	}
}
