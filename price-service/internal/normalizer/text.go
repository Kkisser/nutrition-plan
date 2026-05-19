package normalizer

import (
	"strings"
	"unicode"
)

func NormalizeQuery(input string) string {
	lowered := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(input, "ё", "е")))
	var b strings.Builder
	space := false
	for _, r := range lowered {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if space && b.Len() > 0 {
				b.WriteRune(' ')
			}
			b.WriteRune(r)
			space = false
		case unicode.IsSpace(r), r == '-', r == '_', r == '.':
			space = true
		default:
			space = true
		}
	}
	return strings.TrimSpace(b.String())
}
