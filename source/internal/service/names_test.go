package service_test

import (
	"testing"

	"github.com/sollie/matchmark/internal/service"
)

func TestNormalizeLastName(t *testing.T) {
	cases := map[string]string{
		"anderson":         "ANDERSON",
		"  Smith   Jones ": "SMITH JONES",
		"o'brien":          "O'BRIEN",
		"østergård":        "ØSTERGÅRD",
		"ÉLUARD":           "ÉLUARD",
		"":                 "",
	}
	for in, want := range cases {
		if got := service.NormalizeLastName(in); got != want {
			t.Errorf("NormalizeLastName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeFirstName(t *testing.T) {
	cases := map[string]string{
		"mike":               "Mike",
		"MIKE":               "Mike",
		"anna-lisa":          "Anna-Lisa",
		"d'artagnan":         "D'Artagnan",
		"o’neil":             "O’Neil",
		"  jean   claude  ":  "Jean Claude",
		"jean-claude van":    "Jean-Claude Van",
		"åse":                "Åse",
		"ÉMILIE":             "Émilie",
		"\tbjörn\u00a0erik ": "Björn Erik",
		"":                   "",
	}
	for in, want := range cases {
		if got := service.NormalizeFirstName(in); got != want {
			t.Errorf("NormalizeFirstName(%q) = %q, want %q", in, got, want)
		}
	}
}
