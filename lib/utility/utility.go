package utility

import (
	"bytes"
	"fmt"
	"image"
	"math"
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
func StrBetween(str, a, b string) (string, *errco.Error) {
	// ┌--------str---------┐
	// [ ... a target b ... ]

	aIndex := strings.Index(str, a)
	if aIndex == -1 {
		return "", errco.NewErr(errco.ERROR_ANALYSIS, errco.LVL_3, "StrBetween", fmt.Sprintf("first substring not found (%s)", b))
	}

	bIndex := strings.Index(str[aIndex+len(a):], b)
	if bIndex == -1 {
		return "", errco.NewErr(errco.ERROR_ANALYSIS, errco.LVL_3, "StrBetween", fmt.Sprintf("second substring not found (%s)", b))
	}

	return str[aIndex+len(a):][:bIndex], nil
}

// BytBetween returns the bytearray between 2 subbytearrays
func BytBetween(data, a, b []byte) ([]byte, *errco.Error) {
	// ┌--------data--------┐
	// [ ... a target b ... ]

	aIndex := bytes.Index(data, a)
	if aIndex == -1 {
		return nil, errco.NewErr(errco.ERROR_ANALYSIS, errco.LVL_3, "BytBetween", fmt.Sprintf("first subbytearray not found (%v)", b))
	}

	bIndex := bytes.Index(data[aIndex+len(a):], b)
	if bIndex == -1 {
		return nil, errco.NewErr(errco.ERROR_ANALYSIS, errco.LVL_3, "BytBetween", fmt.Sprintf("second subbytearray not found (%v)", b))
	}

	return data[aIndex+len(a):][:bIndex], nil
}

// SliceContain returns true if the slice contains the element.
// in case of error, false is returned
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
func UnicodeEscape(data []byte) ([]byte, *errco.Error) {
	dataEscapedStr, err := strconv.Unquote(strings.ReplaceAll(strconv.Quote(string(data)), `\\u`, `\u`))
	if err != nil {
		return nil, errco.NewErr(errco.ERROR_CONFIG_SAVE, errco.LVL_3, "UnicodeEscape", "could not escape unicode characters")
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
func Entropy(value string) int {
	// Check http://bearcave.com/misl/misl_tech/wavelets/compression/shannon.html for the algorithmic explanation.

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
