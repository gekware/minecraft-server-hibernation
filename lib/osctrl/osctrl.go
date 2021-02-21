package osctrl

import (
	"log"
	"os"
	"runtime"
)

// CheckOsSupport checks if OS is supported and exit if it's not
func CheckOsSupport() {
	// check if OS is windows/linux/macos
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		log.Print("checkConfig: error: OS not supported!")
		os.Exit(1)
	}
}
