package utility

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"

	"msh/lib/errco"

	"golang.org/x/image/draw"
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

// StrBetween returns the string between 2 substrings.
// In case of error the returned parameters are "" and error
func StrBetween(str, a, b string) (string, *errco.MshLog) {
	// ┌--------str---------┐
	// [ ... a target b ... ]

	aIndex := strings.Index(str, a)
	if aIndex == -1 {
		return "", errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_ANALYSIS, fmt.Sprintf("first substring not found (\"%s\")", b))
	}

	bIndex := strings.Index(str[aIndex+len(a):], b)
	if bIndex == -1 {
		return "", errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_ANALYSIS, fmt.Sprintf("second substring not found (\"%s\")", b))
	}

	return str[aIndex+len(a):][:bIndex], nil
}

// BytBetween returns the bytearray between 2 subbytearrays
func BytBetween(data, a, b []byte) ([]byte, *errco.MshLog) {
	// ┌--------data--------┐
	// [ ... a target b ... ]

	aIndex := bytes.Index(data, a)
	if aIndex == -1 {
		return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_ANALYSIS, fmt.Sprintf("first subbytearray not found (%v)", b))
	}

	bIndex := bytes.Index(data[aIndex+len(a):], b)
	if bIndex == -1 {
		return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_ANALYSIS, fmt.Sprintf("second subbytearray not found (%v)", b))
	}

	return data[aIndex+len(a):][:bIndex], nil
}

// ParseVarInt decodes the minecraft protocol VarInt starting at data[offset]
// and returns its value and the number of bytes it is composed of
func ParseVarInt(data []byte, offset int) (int, int, *errco.MshLog) {
	// scheme: [ 1xxxxxxx | 1xxxxxxx | ... | 0xxxxxxx ]
	// every byte carries 7 bits of data and the most significant bit signals that an other byte follows.
	// a VarInt is composed of 5 bytes at most

	var value int

	for i := 0; i < 5; i++ {
		if offset+i >= len(data) {
			return 0, 0, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_ANALYSIS, "VarInt is truncated")
		}

		byt := data[offset+i]
		value |= int(byt&0x7f) << (7 * i)

		// most significant bit not set: this is the last byte of the VarInt
		if byt&0x80 == 0 {
			return value, i + 1, nil
		}
	}

	return 0, 0, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_ANALYSIS, "VarInt is longer than 5 bytes")
}

// EncodeVarInt encodes value as a minecraft protocol VarInt.
// The returned bytes are 1 to 5, depending on the value:
// encoding it with a fixed amount of bytes overflows as soon as
// the value doesn't fit in the bits available
func EncodeVarInt(value int) []byte {
	// scheme: [ 1xxxxxxx | 1xxxxxxx | ... | 0xxxxxxx ]
	// every byte carries 7 bits of data and the most significant bit signals that an other byte follows

	byt := make([]byte, 0, 5)

	for {
		// no bits left apart from the 7 least significant: this is the last byte of the VarInt
		if value&^0x7f == 0 {
			return append(byt, byte(value))
		}

		byt = append(byt, byte(value&0x7f|0x80))
		value >>= 7
	}
}

// SliceContain returns true if the slice contains the element.
// Supported types: string, int, uint32.
// In case of error, false is returned.
func SliceContain(ele, sli interface{}) bool {
	// check if e and sli types are the same
	if reflect.TypeOf(sli).Elem().Kind() != reflect.TypeOf(ele).Kind() {
		return false
	}

	switch sli := sli.(type) {
	case []string:
		for _, e := range sli {
			if e == ele {
				return true
			}
		}
	case []int:
		for _, e := range sli {
			if e == ele {
				return true
			}
		}
	case []uint32:
		for _, e := range sli {
			if e == ele {
				return true
			}
		}
	}

	return false
}

// UnicodeEscape returns the data escaped from unicode characters
func UnicodeEscape(data []byte) ([]byte, *errco.MshLog) {
	dataEscapedStr, err := strconv.Unquote(strings.ReplaceAll(strconv.Quote(string(data)), `\\u`, `\u`))
	if err != nil {
		return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_ANALYSIS, "could not escape unicode characters")
	}

	return []byte(dataEscapedStr), nil
}

// RoundSec rounds a time duration to the nearest second
func RoundSec(t time.Duration) int {
	return int(math.Round(float64(t.Milliseconds() / 1000)))
}

// ScaleImg scales the image to rectangle size
func ScaleImg(srcImg image.Image, rect image.Rectangle) (image.Image, time.Duration) {
	i := time.Now()
	dstImg := image.NewRGBA(rect)
	draw.CatmullRom.Scale(dstImg, dstImg.Bounds(), srcImg, srcImg.Bounds(), draw.Over, nil)
	return dstImg, time.Since(i)
}

// Entropy measures the Shannon entropy of a string.
// Check bearcave.com/misl/misl_tech/wavelets/compression/shannon.html for the algorithmic explanation.
func Entropy(value string) int {
	frq := make(map[rune]float64)

	//get frequency of characters
	for _, i := range value {
		frq[i]++
	}

	var sum float64

	for _, v := range frq {
		f := v / float64(len(value))
		sum += f * math.Log2(f)
	}

	return int(math.Ceil(sum*-1)) * len(value)
}

// Reverse takes a slice and returns the reverse of it.
// Makes use of "golang type parameters".
func Reverse[slice ~[]ele, ele any](sli slice) slice {
	for i, j := 0, len(sli)-1; i < j; i, j = i+1, j-1 {
		sli[i], sli[j] = sli[j], sli[i]
	}
	return sli
}

// FirstNon returns the first string different from the one specified.
func FirstNon(s string, vals ...string) string {
	for _, v := range vals {
		if v != s {
			return v
		}
	}

	return s
}

// GetOutboundIP4 returns the preferred outbound ip of this machine as ip4 string
func GetOutboundIP4() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_DIAL, err.Error())
		return ""
	}
	defer conn.Close()

	return conn.LocalAddr().(*net.UDPAddr).IP.To4().String()
}
