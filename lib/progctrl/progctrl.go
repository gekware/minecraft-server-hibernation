package progctrl

import (
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
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
		err := servctrl.StopMS(false)
		if err != nil {
			debugctrl.Logln("InterruptListener:", err)
		}

		// wait 1 second to let the server go into stopping mode
		time.Sleep(time.Second)

		switch servctrl.Stats.Status {
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

// these constant represent the result status of checkUpdate()
const (
	ERROR             = -1
	UPDATED           = 0
	UPDATEAVAILABLE   = 1
	UNOFFICIALVERSION = 2
)

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
		status, onlineVersion, err := checkUpdate(v, clientVersion, respHeader)
		if err != nil {
			debugctrl.Logln("UpdateManager:", err.Error())
		}

		if confctrl.ConfigRuntime.Msh.NotifyUpdate {
			switch status {
			case UPDATED:
				fmt.Println("*** msh " + clientVersion + " is updated ***")
			case UPDATEAVAILABLE:
				notification := "*** msh " + onlineVersion + " is now available: visit github to update! ***"
				fmt.Println(notification)

				// notify to game chat every 20 minutes for deltaT time
				go notifyGameChat(20*time.Minute, deltaT, notification)
			case UNOFFICIALVERSION:
				fmt.Println("*** msh " + clientVersion + " is running an unofficial release ***")
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
func checkUpdate(v, clientVersion, respHeader string) (int, string, error) {
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
		return ERROR, "error", fmt.Errorf("checkUpdate: %v", err)
	}
	req.Header.Add("User-Agent", "msh ("+userAgentOs+") msh/"+clientVersion)

	// execute http request
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ERROR, "error", fmt.Errorf("checkUpdate: %v", err)
	}
	defer resp.Body.Close()

	// read http response
	respByte, err := ioutil.ReadAll(resp.Body)
	if err != nil || !strings.Contains(string(respByte), respHeader) {
		return ERROR, "error", fmt.Errorf("checkUpdate: %v", err)
	}

	// no error and respByte contains respHeader
	onlineVersion := strings.ReplaceAll(string(respByte), respHeader, "")

	// check which version is more recent
	res, err := compareVersion(clientVersion, onlineVersion)
	if err != nil {
		return ERROR, "error", fmt.Errorf("checkUpdate: %v", err)
	}

	switch res {
	case -1:
		// an update is available
		return UPDATEAVAILABLE, onlineVersion, nil
	case 1:
		// the runtime version has not yet been officially released
		return UNOFFICIALVERSION, clientVersion, nil
	default:
		// no update available
		return UPDATED, clientVersion, nil
	}
}

// compareVersion returns:
// -2	an error occurred.
// -1	if the first version parameter is smaller.
// 0	if the first version parameter is equale.
// +1	if the first version parameter is greater.
func compareVersion(clientVersion, onlineVersion string) (int, error) {
	// digitize transforms a string "vx.x.x" into an integer x000x000x000
	digitize := func(Version string) (int, error) {
		versionInt := 0

		// replace and split version (input: "vx.x.x") to get a list of integers
		versionSplit := strings.Split(strings.ReplaceAll(Version, "v", ""), ".")
		for n, digit := range versionSplit {
			digitInt, err := strconv.Atoi(digit)
			if err != nil {
				return -2, err
			}
			versionInt += digitInt * int(math.Pow(1000, float64(len(versionSplit)-n)))
		}
		// versionInt has this format: x000x000x000
		return versionInt, nil
	}

	clientVersionInt, err := digitize(clientVersion)
	if err != nil {
		return -2, fmt.Errorf("compareVersion: %v", err)
	}
	onlineVersionInt, err := digitize(onlineVersion)
	if err != nil {
		return -2, fmt.Errorf("compareVersion: %v", err)
	}

	switch {
	case clientVersionInt < onlineVersionInt:
		return -1, nil
	case clientVersionInt > onlineVersionInt:
		return 1, nil
	default:
		return 0, nil
	}
}

// notifyGameChat sends a string with the command "/say"
// every specified amount of time for a specified amount of time
// [goroutine]
func notifyGameChat(deltaNotification, deltaToEnd time.Duration, notificationString string) {
	endT := time.Now().Add(deltaToEnd)

	for time.Now().Before(endT) {
		// check if terminal is active to avoid Execute() returning an error
		if servctrl.ServTerm.IsActive {
			_, err := servctrl.Execute("/say "+notificationString, "notifyGameChat")
			if err != nil {
				debugctrl.Logln("notifyGameChat:", err.Error())
			}
		}

		time.Sleep(deltaNotification)
	}
}
