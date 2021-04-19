package utility

import (
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
