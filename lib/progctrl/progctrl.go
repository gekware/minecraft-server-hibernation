package progctrl

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"msh/lib/servctrl"
)

// CheckOsSupport checks if OS is supported and exit if it's not
func CheckOsSupport() {
	// check if OS is windows/linux/macos
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		log.Print("checkConfig: error: OS not supported!")
		os.Exit(1)
	}
}

// UpdateChecker checks for updates and notify the user is case there is a new version
func UpdateChecker(version string) {
	v := "1"
	// latest-version.php protocol version number: 1
	// connection every 4 hours
	// parameters passed to php: v, version
	// response: "latest version: $latestVersion"

	var latestVersion string

	resp, err := http.Get("http://minecraft-server-hibernation.heliohost.us/latest-version.php?v=" + v + "&version=" + version)
	if err != nil {
		time.AfterFunc(1*time.Minute, func() { UpdateChecker(version) })
		return
	}
	defer resp.Body.Close()

	respByte, err := ioutil.ReadAll(resp.Body)
	if err == nil && strings.Contains(string(respByte), "latest version: ") {
		// no error and contains "latest version: "
		latestVersion = strings.ReplaceAll(string(respByte), "latest version: ", "")
	} else {
		// error happened, suppose that the actual version is updated
		latestVersion = version
	}

	if version != latestVersion {
		fmt.Println("***", latestVersion, "is now available: visit github to update!", "***")
	}

	time.AfterFunc(4*time.Hour, func() { UpdateChecker(version) })
}

// InterruptListener listen for interrupt signals and forcefully stop the minecraft server before exiting msh
func InterruptListener() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		select {
		case <-c:
			servctrl.StopMinecraftServer(true)
			os.Exit(0)
		}
	}()
}
