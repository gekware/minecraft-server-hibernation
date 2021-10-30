package progmgr

import (
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

	"msh/lib/config"
	"msh/lib/errco"
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
		errMsh := servctrl.StopMS(false)
		if errMsh != nil {
			errco.LogMshErr(errMsh.AddTrace("InterruptListener"))
		}

		// wait 1 second to let the server go into stopping mode
		time.Sleep(time.Second)

		switch servctrl.Stats.Status {
		case errco.SERVER_STATUS_STOPPING:
			// if server is correctly stopping, wait for minecraft server to exit
			errco.Logln(errco.LVL_D, "InterruptListener: waiting for minecraft server terminal to exit (server is stopping)")
			servctrl.ServTerm.Wg.Wait()

		case errco.SERVER_STATUS_OFFLINE:
			// if server is offline, then it's safe to continue
			errco.Logln(errco.LVL_D, "InterruptListener: minecraft server terminal already exited (server is offline)")

		default:
			errco.Logln(errco.LVL_D, "InterruptListener: stop command does not seem to be stopping server during forceful shutdown")
		}

		// exit
		errco.Logln(errco.LVL_A, "exiting msh")
		os.Exit(0)
	}
}

var CheckedUpdateC chan bool = make(chan bool, 1)

// UpdateManager checks for updates and notify the user via terminal/gamechat
// [goroutine]
func UpdateManager(clientVersion string) {
	// protocol version number:		1
	// connection every:			4 hours
	// parameters passed to php:	clientProtV, clientVersion
	// request headers:				HTTP_USER_AGENT
	// response:					"latest version: $officialVersion"

	clientProtV := "1"
	deltaT := 4 * time.Hour
	respHeader := "latest version: "

	for {
		errco.Logln(errco.LVL_D, "checking version...")

		status, onlineVersion, errMsh := checkUpdate(clientProtV, clientVersion, respHeader)
		if errMsh != nil {
			// since UpdateManager is a goroutine, don't return and just log the error
			errco.LogMshErr(errMsh.AddTrace("UpdateManager"))
		}

		if config.ConfigRuntime.Msh.NotifyUpdate {
			switch status {
			case errco.VERSION_UPDATED:
				errco.Logln(errco.LVL_A, "msh ("+clientVersion+") is updated")

			case errco.VERSION_UPDATEAVAILABLE:
				notification := "msh (" + onlineVersion + ") is now available: visit github to update!"
				errco.Logln(errco.LVL_A, notification)
				// notify to game chat every 20 minutes for deltaT time
				go notifyGameChat(20*time.Minute, deltaT, notification)

			case errco.VERSION_UNOFFICIALVERSION:
				errco.Logln(errco.LVL_A, "msh ("+clientVersion+") is running an unofficial release")
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
func checkUpdate(clientProtV, clientVersion, respHeader string) (int, string, *errco.Error) {
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
	url := "http://minecraft-server-hibernation.heliohost.us/latest-version.php?v=" + clientProtV + "&version=" + clientVersion
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return errco.VERSION_ERROR, "error", errco.NewErr(errco.VERSION_ERROR, errco.LVL_D, "checkUpdate", err.Error())
	}
	req.Header.Add("User-Agent", "msh ("+userAgentOs+") msh/"+clientVersion)

	// execute http request
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return errco.VERSION_ERROR, "error", errco.NewErr(errco.VERSION_ERROR, errco.LVL_D, "checkUpdate", err.Error())
	}
	defer resp.Body.Close()

	// read http response
	respByte, err := ioutil.ReadAll(resp.Body)
	if err != nil || !strings.Contains(string(respByte), respHeader) {
		return errco.VERSION_ERROR, "error", errco.NewErr(errco.VERSION_ERROR, errco.LVL_D, "checkUpdate", err.Error())
	}

	// no error and respByte contains respHeader
	onlineVersion := strings.ReplaceAll(string(respByte), respHeader, "")

	// check which version is more recent
	delta, errMsh := deltaVersion(onlineVersion, clientVersion)
	if errMsh != nil {
		return errco.VERSION_ERROR, "error", errMsh.AddTrace("checkUpdate")
	}

	switch {
	case delta > 0:
		// an update is available
		return errco.VERSION_UPDATEAVAILABLE, onlineVersion, nil
	case delta < 0:
		// the runtime version has not yet been officially released
		return errco.VERSION_UNOFFICIALVERSION, onlineVersion, nil
	default:
		// no update available
		return errco.VERSION_UPDATED, onlineVersion, nil
	}
}

// deltaVersion returns the difference between onlineVersion and clientVersion:
// =0	versions are equal or an error occurred.
// >0	if onlineVersion is more recent.
// <0	if onlineVersion is less recent.
func deltaVersion(onlineVersion, clientVersion string) (int, *errco.Error) {
	// digitize transforms a string "vx.x.x" into an integer x000x000x000
	digitize := func(Version string) (int, error) {
		versionInt := 0

		// replace and split version (input: "vx.x.x") to get a list of integers
		versionSplit := strings.Split(strings.ReplaceAll(Version, "v", ""), ".")
		for n, digit := range versionSplit {
			digitInt, err := strconv.Atoi(digit)
			if err != nil {
				return 0, err
			}
			versionInt += digitInt * int(math.Pow(1000, float64(len(versionSplit)-n)))
		}
		// versionInt has this format: x000x000x000
		return versionInt, nil
	}

	clientVersionInt, err := digitize(clientVersion)
	if err != nil {
		return 0, errco.NewErr(errco.VERSION_COMPARISON_ERROR, errco.LVL_D, "deltaVersion", err.Error())
	}
	onlineVersionInt, err := digitize(onlineVersion)
	if err != nil {
		return 0, errco.NewErr(errco.VERSION_COMPARISON_ERROR, errco.LVL_D, "deltaVersion", err.Error())
	}

	return onlineVersionInt - clientVersionInt, nil
}

// notifyGameChat sends a string with the command "say"
// every specified amount of time for a specified amount of time
// [goroutine]
func notifyGameChat(deltaNotification, deltaToEnd time.Duration, notificationString string) {
	endT := time.Now().Add(deltaToEnd)

	for time.Now().Before(endT) {
		// check if terminal is active to avoid Execute() returning an error
		if servctrl.ServTerm.IsActive {
			_, errMsh := servctrl.Execute("say "+notificationString, "notifyGameChat")
			if errMsh != nil {
				errco.LogMshErr(errMsh.AddTrace("notifyGameChat"))
			}
		}

		time.Sleep(deltaNotification)
	}
}
