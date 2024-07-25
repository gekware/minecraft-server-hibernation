package minequery

import (
	"errors"
	"image"
	"io"
	"regexp"
	"strings"
)

var errStackEmpty = errors.New("stack is empty")

// stack provides a simple array-based stack implementation for internal use within minequery.
type stack []interface{}

// Push pushes an item on top of the stack.
func (s *stack) Push(value interface{}) { *s = append(*s, value) }

// Pop removes an item from the top of the stack, returning errStackEmpty if stack is empty.
func (s *stack) Pop() (interface{}, error) {
	if len(*s) == 0 {
		return nil, errStackEmpty
	}
	ret := (*s)[len(*s)-1]
	*s = (*s)[0 : len(*s)-1]
	return ret, nil
}

// maxInt returns greatest of two of the passed integer numbers.
func maxInt(a int, b int) int {
	if a < b {
		return b
	} else {
		return a
	}
}

// UnmarshalFunc is a function that conforms to json.Unmarshal function signature.
type UnmarshalFunc func([]byte, interface{}) error

// ImageDecodeFunc is a function that conforms to png.Decode function signature.
type ImageDecodeFunc func(io.Reader) (image.Image, error)

// naturalizeMOTD 'naturalizes' MOTD (or since 1.7+, description) strings and turns them
// into one-line, stripped of any formatting strings. Newlines are replaced with spaces and
// legacy §-formatting is omitted.
func naturalizeMOTD(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = regexp.MustCompile("\u00a7[a-f0-9k-or]").ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	return s
}
