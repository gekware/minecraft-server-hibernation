package progmgr

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

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/servctrl"
	"msh/lib/servstats"
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

		switch servstats.Stats.Status {
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
func UpdateManager(versClient string) {
	// protocol version number:		1
	// connection every:			4 hours
	// parameters passed to php:	v (prot), version (client)
	// request headers:				HTTP_USER_AGENT
	// response:					"latest version: $officialVersion"

	protv := 1
	deltaT := 4 * time.Hour
	respHeader := "latest version: "

	for {
		errco.Logln(errco.LVL_D, "checking version...")

		status, versOnline, errMsh := checkUpdate(protv, versClient, respHeader)
		if errMsh != nil {
			// since UpdateManager is a goroutine, don't return and just log the error
			errco.LogMshErr(errMsh.AddTrace("UpdateManager"))
		}

		if config.ConfigRuntime.Msh.NotifyUpdate {
			switch status {
			case errco.VERSION_UPDATED:
				errco.Logln(errco.LVL_A, "msh (%s) is updated", versClient)

			case errco.VERSION_UPDATEAVAILABLE:
				notification := fmt.Sprintf("msh (%s) is now available: visit github to update!", versOnline)
				errco.Logln(errco.LVL_A, notification)
				// notify to game chat every 20 minutes for deltaT time
				go notifyGameChat(20*time.Minute, deltaT, notification)

			case errco.VERSION_UNOFFICIALVERSION:
				errco.Logln(errco.LVL_A, "msh (%s) is running an unofficial release", versClient)
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
func checkUpdate(protv int, versClient, respHeader string) (int, string, *errco.Error) {
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
	url := "http://minecraft-server-hibernation.heliohost.us/latest-version.php?v=" + fmt.Sprint(protv) + "&version=" + versClient
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return errco.ERROR_VERSION, "error", errco.NewErr(errco.ERROR_VERSION, errco.LVL_D, "checkUpdate", err.Error())
	}
	req.Header.Add("User-Agent", "msh ("+userAgentOs+") msh/"+versClient)

	// execute http request
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return errco.ERROR_VERSION, "error", errco.NewErr(errco.ERROR_VERSION, errco.LVL_D, "checkUpdate", err.Error())
	}
	defer resp.Body.Close()

	// read http response
	respByte, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errco.ERROR_VERSION, "error", errco.NewErr(errco.ERROR_VERSION, errco.LVL_D, "checkUpdate", err.Error())
	}
	if !strings.Contains(string(respByte), respHeader) {
		return errco.ERROR_VERSION, "error", errco.NewErr(errco.ERROR_VERSION, errco.LVL_D, "checkUpdate", "missing response header")
	}

	// no error and respByte contains respHeader
	versOnline := strings.ReplaceAll(string(respByte), respHeader, "")

	// check which version is more recent
	delta, errMsh := deltaVersion(versOnline, versClient)
	if errMsh != nil {
		return errco.ERROR_VERSION, "error", errMsh.AddTrace("checkUpdate")
	}

	switch {
	case delta > 0:
		// an update is available
		return errco.VERSION_UPDATEAVAILABLE, versOnline, nil
	case delta < 0:
		// the runtime version has not yet been officially released
		return errco.VERSION_UNOFFICIALVERSION, versOnline, nil
	default:
		// no update available
		return errco.VERSION_UPDATED, versOnline, nil
	}
}

// deltaVersion returns the difference between versOnline and versClient:
// =0	versions are equal or an error occurred.
// >0	if official version is more recent.
// <0	if official version is less recent.
func deltaVersion(versOnline, versClient string) (int, *errco.Error) {
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

	versClientInt, err := digitize(versClient)
	if err != nil {
		return 0, errco.NewErr(errco.ERROR_VERSION_COMPARISON, errco.LVL_D, "deltaVersion", err.Error())
	}
	versOnlineInt, err := digitize(versOnline)
	if err != nil {
		return 0, errco.NewErr(errco.ERROR_VERSION_COMPARISON, errco.LVL_D, "deltaVersion", err.Error())
	}

	return versOnlineInt - versClientInt, nil
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
