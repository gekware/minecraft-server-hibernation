package minequery

import (
	"bytes"
	"io"

	"golang.org/x/text/encoding/unicode"
)

var (
	utf16BEEncoder = unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewEncoder()
	utf16BEDecoder = unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()
)

func readAllUntilZero(reader io.ByteReader) ([]byte, error) {
	buf := &bytes.Buffer{}

	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				return buf.Bytes(), nil
			}
			return nil, err
		}

		if b != 0 {
			buf.WriteByte(b)
		} else {
			return buf.Bytes(), nil
		}
	}
}
