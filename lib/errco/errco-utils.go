package errco

import (
	"strings"
	"unicode"
)

// StringGraphic returns the input string without non-graphic characters
func StringGraphic(s string) string {
	f := func(r rune) rune {
		// unicode.IsPrint		considers only characters that occupy space on a page or screen.
		// unicode.IsGraphic	considers all characters that have a visible representation (including spaces and control characters).
		if unicode.IsGraphic(r) {
			return r
		}
		return -1
	}

	return strings.Map(f, s)
}
