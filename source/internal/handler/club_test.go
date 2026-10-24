package handler_test

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"strings"
	"testing"

	"github.com/sollie/matchmark/internal/apitest"
)

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func jpegBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h)), nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestClubDefaultsToNoNameAndNoLogo(t *testing.T) {
	env := apitest.New(t)
	status, out := env.Do("GET", "/api/club", nil)
	if status != http.StatusOK || out["name"] != "" || out["has_logo"] != false {
		t.Fatalf("club: %d %v", status, out)
	}
	if status, _, _ := env.Raw("GET", "/api/club/logo"); status != http.StatusNotFound {
		t.Fatalf("logo: %d", status)
	}
}

func TestAdminSetsClubNamePublicly(t *testing.T) {
	env := apitest.NewAdmin(t)
	status, out := env.Do("PUT", "/api/club", map[string]string{"name": "  Oslo PPC  "})
	if status != http.StatusOK || out["name"] != "Oslo PPC" {
		t.Fatalf("put: %d %v", status, out)
	}
	_, got := env.Do("GET", "/api/club", nil)
	if got["name"] != "Oslo PPC" {
		t.Fatalf("get: %v", got)
	}
	if status, _ := env.Do("PUT", "/api/club", map[string]string{"name": strings.Repeat("x", 101)}); status != http.StatusUnprocessableEntity {
		t.Fatalf("long name: %d", status)
	}
}

func TestClubLogoRoundTrip(t *testing.T) {
	env := apitest.NewAdmin(t)
	img := pngBytes(t, 64, 32)
	status, _, _ := env.RawBody("PUT", "/api/club/logo", "image/png", img)
	if status != http.StatusNoContent {
		t.Fatalf("put: %d", status)
	}
	status, hdr, body := env.Raw("GET", "/api/club/logo?v=3")
	if status != http.StatusOK || hdr.Get("Content-Type") != "image/png" || !bytes.Equal(body, img) {
		t.Fatalf("get: %d %v", status, hdr)
	}
	if hdr.Get("Cache-Control") != "no-cache" {
		t.Fatalf("cache-control %q", hdr.Get("Cache-Control"))
	}
	_, club := env.Do("GET", "/api/club", nil)
	if club["has_logo"] != true {
		t.Fatalf("club: %v", club)
	}
	version := club["version"]

	if status, _, _ := env.RawBody("PUT", "/api/club/logo", "image/jpeg", jpegBytes(t, 10, 10)); status != http.StatusNoContent {
		t.Fatalf("jpeg: %d", status)
	}
	_, club = env.Do("GET", "/api/club", nil)
	if club["version"] == version {
		t.Fatalf("version did not change: %v", club)
	}

	if status, _, _ := env.Raw("DELETE", "/api/club/logo"); status != http.StatusNoContent {
		t.Fatalf("delete: %d", status)
	}
	if status, _, _ := env.Raw("GET", "/api/club/logo"); status != http.StatusNotFound {
		t.Fatalf("after delete: %d", status)
	}
}

func TestClubLogoRejectsInvalidInput(t *testing.T) {
	env := apitest.NewAdmin(t)
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)
	cases := map[string]struct {
		contentType string
		body        []byte
	}{
		"svg":             {"image/svg+xml", svg},
		"svg as png":      {"image/png", svg},
		"gif":             {"image/gif", []byte("GIF89a")},
		"png as jpeg":     {"image/jpeg", pngBytes(t, 10, 10)},
		"too many pixels": {"image/png", pngBytes(t, 2049, 10)},
		"too many bytes":  {"image/png", append(pngBytes(t, 10, 10), make([]byte, 512*1024)...)},
		"empty":           {"image/png", nil},
		"corrupt":         {"image/png", []byte("not an image")},
	}
	for name, c := range cases {
		if status, _, _ := env.RawBody("PUT", "/api/club/logo", c.contentType, c.body); status != http.StatusUnprocessableEntity {
			t.Errorf("%s: status %d", name, status)
		}
	}
	if status, _, _ := env.Raw("GET", "/api/club/logo"); status != http.StatusNotFound {
		t.Fatalf("logo stored after rejection: %d", status)
	}
}

func TestClubSettingsRequireAdmin(t *testing.T) {
	env := apitest.New(t)
	if status, _ := env.Do("PUT", "/api/club", map[string]string{"name": "X"}); status != http.StatusUnauthorized {
		t.Fatalf("name: %d", status)
	}
	if status, _, _ := env.RawBody("PUT", "/api/club/logo", "image/png", pngBytes(t, 4, 4)); status != http.StatusUnauthorized {
		t.Fatalf("logo put: %d", status)
	}
	if status, _, _ := env.Raw("DELETE", "/api/club/logo"); status != http.StatusUnauthorized {
		t.Fatalf("logo delete: %d", status)
	}
}

func TestClubChangesAreAudited(t *testing.T) {
	env := apitest.NewAdmin(t)
	env.Do("PUT", "/api/club", map[string]string{"name": "Oslo PPC"})
	env.RawBody("PUT", "/api/club/logo", "image/png", pngBytes(t, 4, 4))
	env.Raw("DELETE", "/api/club/logo")
	var actions []any
	for _, l := range env.AuditLines() {
		actions = append(actions, l["action"])
	}
	want := []string{"club.update", "club.logo.set", "club.logo.delete"}
	for _, w := range want {
		found := false
		for _, a := range actions {
			found = found || a == w
		}
		if !found {
			t.Errorf("missing %s in %v", w, actions)
		}
	}
}
