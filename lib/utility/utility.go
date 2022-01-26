package utility

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"msh/lib/errco"
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

// StrBetween returns the string between 2 substrings
func StrBetween(str, a, b string) (string, *errco.Error) {
	// ┌--------str---------┐
	// [ ... a target b ... ]

	aIndex := strings.Index(str, a)
	if aIndex == -1 {
		return "", errco.NewErr(errco.ERROR_ANALYSIS, errco.LVL_D, "StrBetween", fmt.Sprintf("first substring not found (%s)", b))
	}

	bIndex := strings.Index(str[aIndex+len(a):], b)
	if bIndex == -1 {
		return "", errco.NewErr(errco.ERROR_ANALYSIS, errco.LVL_D, "StrBetween", fmt.Sprintf("second substring not found (%s)", b))
	}

	return str[aIndex+len(a):][:bIndex], nil
}

// BytBetween returns the bytearray between 2 subbytearrays
func BytBetween(data, a, b []byte) ([]byte, *errco.Error) {
	// ┌--------data--------┐
	// [ ... a target b ... ]

	aIndex := bytes.Index(data, a)
	if aIndex == -1 {
		return nil, errco.NewErr(errco.ERROR_ANALYSIS, errco.LVL_D, "BytBetween", fmt.Sprintf("first subbytearray not found (%v)", b))
	}

	bIndex := bytes.Index(data[aIndex+len(a):], b)
	if bIndex == -1 {
		return nil, errco.NewErr(errco.ERROR_ANALYSIS, errco.LVL_D, "BytBetween", fmt.Sprintf("second subbytearray not found (%v)", b))
	}

	return data[aIndex+len(a):][:bIndex], nil
}

// SliceContain returns true if the slice contains the element.
// in case of error, false is returned
func SliceContain(e, sli interface{}) bool {
	// check if e and sli types are the same
	if reflect.TypeOf(sli).Elem().Kind() != reflect.TypeOf(e).Kind() {
		return false
	}

	switch sli := sli.(type) {
	case []string:
		for _, slie := range sli {
			if e == slie {
				return true
			}
		}
	case []int:
		for _, slie := range sli {
			if e == slie {
				return true
			}
		}
	}

	return false
}
