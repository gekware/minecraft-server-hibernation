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

	"msh/lib/confctrl"
	"msh/lib/debugctrl"
	"msh/lib/servctrl"
)

// InterruptListener listen for interrupt signals and forcefully stop the minecraft server before exiting msh.
// [goroutine]
func InterruptListener() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// wait for termination signal
	<-c

	// stop forcefully the minecraft server
	err := servctrl.StopMinecraftServer(true)
	if err != nil {
		debugctrl.Logln("InterruptListener:", err)
	}

	// exit
	fmt.Print("exiting msh")
	os.Exit(0)
}

var CheckedUpdateC chan bool = make(chan bool, 1)

// UpdateManager checks for updates and notify the user via terminal/gamechat
// [goroutine]
func UpdateManager(clientVersion string) {
	// protocol version number:		1
	// connection every:			4 hours
	// parameters passed to php:	v, clientVersion
	// response:					"latest version: $onlineVersion"

	v := "1"
	deltaT := 4 * time.Hour
	respHeader := "latest version: "

	for {
		if confctrl.Config.Msh.CheckForUpdates {
			updateAvailable, onlineVersion, err := checkUpdate(v, clientVersion, respHeader)
			if err != nil {
				debugctrl.Logln("UpdateManager:", err.Error())
				time.Sleep(deltaT)
				continue
			}

			if updateAvailable {
				notificationString := "*** msh " + onlineVersion + " is now available: visit github to update! ***"

				// notify on msh terminal
				fmt.Println(notificationString)

				// notify to game chat every 20 minutes for deltaT time
				go notifyGameChat(20*time.Minute, deltaT, notificationString)
			}
		}

		select {
		case CheckedUpdateC <- true:
		default:
		}

		time.Sleep(deltaT)
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

// notifyGameChat sends a string with the command "/say"
// every specified amount of time for a specified amount of time
// [goroutine]
func notifyGameChat(deltaNotification, deltaToEnd time.Duration, notificationString string) {
	endT := time.Now().Add(deltaToEnd)

	for time.Now().Before(endT) {
		// check if terminal is active to avoid Execute() returning an error
		if servctrl.ServTerminal.IsActive {
			_, err := servctrl.ServTerminal.Execute("/say "+notificationString, "notifyGameChat")
			if err != nil {
				debugctrl.Logln("notifyGameChat:", err.Error())
			}
		}

		time.Sleep(deltaNotification)
	}
}
