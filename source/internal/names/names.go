package names

import (
	"strings"
	"unicode"
)

func LastName(s string) string {
	return strings.ToUpper(strings.Join(strings.Fields(s), " "))
}

func FirstName(s string) string {
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
