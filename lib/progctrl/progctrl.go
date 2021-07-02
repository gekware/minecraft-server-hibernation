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

	for {
		// wait for termination signal
		<-c

		// stop the minecraft server with no player check
		err := servctrl.StopMinecraftServer(false)
		if err != nil {
			debugctrl.Logln("InterruptListener:", err)
		}

		// wait 1 second to let the server go into stopping mode
		time.Sleep(time.Second)

		switch servctrl.ServStats.Status {
		case "stopping":
			// if server is correctly stopping, wait for minecraft server to exit
			debugctrl.Logln("InterruptListener: waiting for minecraft server terminal to exit (server is stopping)")
			servctrl.ServTerm.Wg.Wait()
		case "offline":
			// if server is offline, then it's safe to continue
			debugctrl.Logln("InterruptListener: minecraft server terminal already exited (server is offline)")
		default:
			debugctrl.Logln("InterruptListener: stop command does not seem to be stopping server during forceful shutdown")
		}

		// exit
		fmt.Print("exiting msh")
		os.Exit(0)
	}
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
		if confctrl.ConfigRuntime.Msh.CheckForUpdates {
			updateAvailable, onlineVersion, err := checkUpdate(v, clientVersion, respHeader)
			if err != nil {
				debugctrl.Logln("UpdateManager:", err.Error())
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
// if error occurred, online version will be "error"
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
		return false, "error", fmt.Errorf("checkUpdate: %v", err)
	}
	req.Header.Add("User-Agent", "msh ("+userAgentOs+") msh/"+clientVersion)

	// execute http request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, "error", fmt.Errorf("checkUpdate: %v", err)
	}
	defer resp.Body.Close()

	// read http response
	respByte, err := ioutil.ReadAll(resp.Body)
	if err != nil || !strings.Contains(string(respByte), respHeader) {
		return false, "error", fmt.Errorf("checkUpdate: (error reading http response) %v", err)
	}

	// no error and respByte contains respHeader
	onlineVersion := strings.ReplaceAll(string(respByte), respHeader, "")
	if clientVersion == onlineVersion {
		// no update available, return updateAvailable == false
		return false, onlineVersion, nil
	}

	// an update is available, return updateAvailable == true
	return true, onlineVersion, nil
}

// notifyGameChat sends a string with the command "/say"
// every specified amount of time for a specified amount of time
// [goroutine]
func notifyGameChat(deltaNotification, deltaToEnd time.Duration, notificationString string) {
	endT := time.Now().Add(deltaToEnd)

	for time.Now().Before(endT) {
		// check if terminal is active to avoid Execute() returning an error
		if servctrl.ServTerm.IsActive {
			_, err := servctrl.ServTerm.Execute("/say "+notificationString, "notifyGameChat")
			if err != nil {
				debugctrl.Logln("notifyGameChat:", err.Error())
			}
		}

		time.Sleep(deltaNotification)
	}
}
