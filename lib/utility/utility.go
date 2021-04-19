package utility

import (
	"fmt"
	"strings"
)

// Boxify creates an ascii box around a list of text lines
func Boxify(strList []string) string {
	// find longest string in list
	max := 0
	for _, l := range strList {
		if len(l) > max {
			max = len(l)
		}
	}

	// text box generation
	textBox := ""
	textBox += "╔═" + strings.Repeat("═", max) + "═╗" + "\n"
	for _, l := range strList {
		textBox += "║ " + l + strings.Repeat(" ", max-len(l)) + " ║" + "\n"
	}
	textBox += "╚═" + strings.Repeat("═", max) + "═╝"

	return textBox
}

// StrBetween returns the substring between 2 substrings
func StrBetween(str string, a string, b string) (string, error) {
	aIndex := strings.Index(str, a)
	if aIndex == -1 {
		return "", fmt.Errorf("StrBetween: first substring not found")
	}
	bIndex := strings.Index(str, b)
	if bIndex == -1 {
		return "", fmt.Errorf("StrBetween: second substring not found")
	}

	// the position of the last letter of a is needed
	aEndIndex := aIndex + len(a)
	if aEndIndex >= bIndex {
		return "", fmt.Errorf("StrBetween: second substring index is before first")
	}

	return str[aEndIndex:bIndex], nil
}
