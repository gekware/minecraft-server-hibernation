package osctrl

import (
	"log"
	"os"
	"runtime"
)

// CheckOsSupport checks if OS is supported and exit if it's not
func CheckOsSupport() {
	// check if OS is windows/linux/macos
	ros := runtime.GOOS
	if ros != "linux" && ros != "windows" && ros != "darwin" {
		log.Print("osctrl: CheckOsSupport: OS not supported!")
		os.Exit(1)
	}
}
