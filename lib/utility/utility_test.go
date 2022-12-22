package utility

import (
	"fmt"
	"testing"
)

func Test_FirstNon(t *testing.T) {
	input := [][]string{
		{"", "aaa", "bbb", "ccc"},
		{"", "", "aaa", "bbb"},
		{"", "", "", ""},
		{""},
		{"test", "aaa", "bbb"},
	}
	expected := []string{
		"aaa",
		"aaa",
		"",
		"",
		"aaa",
	}

	for n, i := range input {
		var fn string
		if fn = FirstNon(i[0], i[0:]...); fn != expected[n] {
			t.Fatalf("fn (%s) different from expected (%s)", fn, expected[n])
		}
		fmt.Println(fn)
	}
}
