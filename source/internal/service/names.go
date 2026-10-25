package service

import (
	"strings"
	"unicode"
)

func NormalizeLastName(s string) string {
	return strings.ToUpper(strings.Join(strings.Fields(s), " "))
}

func NormalizeFirstName(s string) string {
	var b strings.Builder
	upper := true
	for _, r := range strings.Join(strings.Fields(s), " ") {
		if upper {
			b.WriteRune(unicode.ToUpper(r))
		} else {
			b.WriteRune(unicode.ToLower(r))
		}
		upper = r == ' ' || r == '-' || r == '\'' || r == '’'
	}
	return b.String()
}

func validShooterNames(first, last string) bool {
	return first == NormalizeFirstName(first) && last == NormalizeLastName(last)
}
