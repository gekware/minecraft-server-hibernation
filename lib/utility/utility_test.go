package utility

import (
	"bytes"
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

func Test_VarInt(t *testing.T) {
	// values around the boundaries where the VarInt grows by one byte
	tests := []struct {
		value  int
		expect []byte
	}{
		{0, []byte{0}},
		{1, []byte{1}},
		{127, []byte{127}},
		{128, []byte{128, 1}},
		{758, []byte{246, 5}},        // minecraft 1.18.2 protocol version
		{16383, []byte{255, 127}},    // last value that fits in 2 bytes
		{16384, []byte{128, 128, 1}}, // the value that overflowed the old 2 bytes encoder
		{2097151, []byte{255, 255, 127}},
	}

	for _, test := range tests {
		// encode
		encoded := EncodeVarInt(test.value)
		if !bytes.Equal(encoded, test.expect) {
			t.Errorf("EncodeVarInt(%d) = %v, expected %v", test.value, encoded, test.expect)
			continue
		}

		// decode back
		value, size, logMsh := ParseVarInt(encoded, 0)
		if logMsh != nil {
			t.Errorf("ParseVarInt(%v) failed: %s", encoded, fmt.Sprintf(logMsh.Mex, logMsh.Arg...))
			continue
		}
		if value != test.value || size != len(test.expect) {
			t.Errorf("ParseVarInt(%v) = (%d, %d), expected (%d, %d)", encoded, value, size, test.value, len(test.expect))
		}
	}
}

func Test_ParseVarInt_errors(t *testing.T) {
	tests := []struct {
		title  string
		data   []byte
		offset int
	}{
		{"empty data", []byte{}, 0},
		{"offset out of range", []byte{1}, 1},
		{"truncated VarInt", []byte{128, 128}, 0},
		{"VarInt longer than 5 bytes", []byte{128, 128, 128, 128, 128, 1}, 0},
	}

	for _, test := range tests {
		if _, _, logMsh := ParseVarInt(test.data, test.offset); logMsh == nil {
			t.Errorf("%s: expected an error, got none (data: %v, offset: %d)", test.title, test.data, test.offset)
		}
	}
}
