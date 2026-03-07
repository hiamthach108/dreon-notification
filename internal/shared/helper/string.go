package helper

import (
	"strings"
	"unicode"

	"github.com/google/uuid"
)

// NormalizeSlug replaces spaces with "_" and removes special characters,
// keeping only letters, digits, and underscores.
func NormalizeSlug(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == ' ':
			b.WriteRune('_')
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_':
			b.WriteRune(r)
		}
	}
	return b.String()
}

func RandomString(n int) string {
	return uuid.New().String()[:n]
}
