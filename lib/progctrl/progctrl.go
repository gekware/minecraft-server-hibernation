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

func UpdateManager(clientVersion string) {
	// protocol version number:		1
	// connection every:			4 hours
	// parameters passed to php:	v, clientVersion
	// response:					"latest version: $onlineVersion"

	v := "1"
	deltaT := 4 * time.Hour
	respHeader := "latest version: "

	// after UpdateChecker has returned (for error or completion), launch an other UpdateChecker instance
	defer time.AfterFunc(deltaT, func() { UpdateManager(clientVersion) })

	updateAvailable, onlineVersion, err := checkUpdate(v, clientVersion, respHeader)
	if err != nil {
		debugctrl.Logger("progctrl: UpdateManager:", err.Error())
		return
	}

	if updateAvailable {
		notificationString := "*** msh " + onlineVersion + " is now available: visit github to update! ***"

		// notify on msh terminal
		fmt.Println(notificationString)

		// notify to game chat
		go notifyEveryFor(20*time.Minute, deltaT, notificationString)
	}
}

// checkUpdate checks for updates. Returns (update available, online version, error)
func checkUpdate(v, clientVersion, respHeader string) (bool, string, error) {
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
		return false, "", fmt.Errorf("checkUpdate: %v", err)
	}
	req.Header.Add("User-Agent", "msh ("+userAgentOs+") msh/"+clientVersion)

	// execute http request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("checkUpdate: %v", err)
	}
	defer resp.Body.Close()

	// read http response
	respByte, err := ioutil.ReadAll(resp.Body)
	if err != nil || !strings.Contains(string(respByte), respHeader) {
		return false, "", fmt.Errorf("checkUpdate: (error reading http response) %v", err)
	}

	// no error and respByte contains respHeader
	onlineVersion := strings.ReplaceAll(string(respByte), respHeader, "")
	if clientVersion == onlineVersion {
		// no update available, return updateAvailable == false
		return false, onlineVersion, nil
	}

	// an update is available, return updateAvailable == false
	return true, onlineVersion, nil
}

// notifyEveryFor sends a string with the command "/say"
// every specified amount of time for a specified amount of time
func notifyEveryFor(deltaNotification, deltaToEnd time.Duration, notificationString string) {
	endT := time.Now().Add(deltaToEnd)

	for time.Now().Before(endT) {
		if servctrl.ServTerminal.IsActive {
			_, err := servctrl.ServTerminal.Execute("/say " + notificationString)
			if err != nil {
				debugctrl.Logger("progctrl: notifyEveryFor:", err.Error())
			}
		}

		time.Sleep(deltaNotification)
	}
}
