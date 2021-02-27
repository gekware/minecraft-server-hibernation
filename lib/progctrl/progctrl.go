package progctrl

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"msh/lib/debugctrl"
	"msh/lib/servctrl"
)

// InterruptListener listen for interrupt signals and forcefully stop the minecraft server before exiting msh
func InterruptListener() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-c
		servctrl.StopMinecraftServer(true)
		os.Exit(0)
	}()
}

// UpdateChecker checks for updates and notify the user is case there is a new version
func UpdateChecker(clientVersion string) {
	v := "1"
	// protocol version number:		1
	// connection every:			4 hours
	// parameters passed to php:	v, clientVersion
	// response:					"latest version: $onlineVersion"

	// after UpdateChecker has returned (for error or completion), launch an other UpdateChecker instance
	defer time.AfterFunc(4*time.Hour, func() { UpdateChecker(clientVersion) })

	userAgentOs := "osNotSupported"
	switch runtime.GOOS {
	case "windows":
		userAgentOs = "windows"
	case "linux":
		userAgentOs = "linux"
	case "darwin":
		userAgentOs = "macintosh"
	}

	// build http request
	url := "http://minecraft-server-hibernation.heliohost.us/latest-version.php?v=" + v + "&version=" + clientVersion
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		debugctrl.Logger("progctrl: UpdateChecker:", err.Error())
		return
	}
	req.Header.Add("User-Agent", "msh ("+userAgentOs+") msh/"+clientVersion)

	// execute http request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		debugctrl.Logger("progctrl: UpdateChecker:", err.Error())
		return
	}
	defer resp.Body.Close()

	// read http response
	respByte, err := ioutil.ReadAll(resp.Body)
	if err != nil || !strings.Contains(string(respByte), "latest version: ") {
		debugctrl.Logger("progctrl: UpdateChecker: error reading http response")
		return
	}

	// no error and respByte contains "latest version: "
	onlineVersion := strings.ReplaceAll(string(respByte), "latest version: ", "")
	if clientVersion != onlineVersion {
		fmt.Println("***", onlineVersion, "is now available: visit github to update!", "***")
	}
}
