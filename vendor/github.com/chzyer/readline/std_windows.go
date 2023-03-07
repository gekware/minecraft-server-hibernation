//go:build windows
// +build windows

package readline

func init() {
	Stdin = NewRawReader()
}
