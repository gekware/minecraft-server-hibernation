package progmgr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/model"
	"msh/lib/servctrl"
	"msh/lib/servstats"
)

var (
	MshVersion string = "v2.4.5"  // msh version
	MshCommit  string = "-------" // msh commit

	// CheckedUpdateC communicates to main func that the first update check
	// has been done and msh can continue
	CheckedUpdateC chan bool = make(chan bool, 1)

	protv   int    = 2                                                                 // api protocol version
	updAddr string = fmt.Sprintf("https://mshdev.gekware.net/api/v%d/versions", protv) // server address to check update

	// msh program
	msh *program = &program{
		startTime: time.Now(),
		sigExit:   make(chan os.Signal, 1),
	}
)

type program struct {
	startTime time.Time      // msh program start time
	sigExit   chan os.Signal // channel through which OS termination signals are notified
}

// MshMgr handles exit signal and updates for msh
// [goroutine]
func MshMgr() {
	// set sigExit to relay termination signals
	signal.Notify(msh.sigExit, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT)

	// initialize sgm variables
	sgm.reset(0) // segment duration initialized to 0 so that update check can be executed immediately

	// start segment manager
	go sgm.sgmMgr()

	for {
	mainselect:
		select {
		// msh termination signal is received
		case <-msh.sigExit:
			// stop the minecraft server with no player check
			errMsh := servctrl.StopMS(false)
			if errMsh != nil {
				errco.LogMshErr(errMsh.AddTrace("MshMgr"))
			}

			// send last statistics before exiting
			go sendApi2Req(updAddr, buildApi2Req(true))

			// wait 1 second to let the server go into stopping mode
			time.Sleep(time.Second)

			switch servstats.Stats.Status {
			case errco.SERVER_STATUS_STOPPING:
				// if server is correctly stopping, wait for minecraft server to exit
				errco.Logln(errco.LVL_D, "MshMgr: waiting for minecraft server terminal to exit (server is stopping)")
				servctrl.ServTerm.Wg.Wait()

			case errco.SERVER_STATUS_OFFLINE:
				// if server is offline, then it's safe to continue
				errco.Logln(errco.LVL_D, "MshMgr: minecraft server terminal already exited (server is offline)")

			default:
				errco.Logln(errco.LVL_D, "MshMgr: stop command does not seem to be stopping server during forceful shutdown")
			}

			// exit
			errco.Logln(errco.LVL_A, "exiting msh")
			os.Exit(0)

		// check for update when segment ends
		case <-sgm.end.C:
			// send statistics
			res, errMsh := sendApi2Req(updAddr, buildApi2Req(false))
			if errMsh != nil {
				errco.LogMshErr(errMsh.AddTrace("UpdateManager"))
				sgm.prolong(10 * time.Minute)
				break mainselect
			}

			// check update response status code
			switch res.StatusCode {
			case 200:
				errco.Logln(errco.LVL_D, "resetting segment...")
				sgm.reset(res)
			case 403:
				errco.LogMshErr(errco.NewErr(errco.ERROR_VERSION, errco.LVL_D, "MshMgr", "client is unauthorized"))
				os.Exit(1)
			default:
				errco.LogMshErr(errco.NewErr(errco.ERROR_VERSION, errco.LVL_D, "MshMgr", "response status code is "+res.Status))
				errco.Logln(errco.LVL_D, "prolonging segment...")
				sgm.prolong(res)
				break mainselect
			}

			// get server response into struct
			resJson, errMsh := readApi2Res(res)
			if errMsh != nil {
				errco.LogMshErr(errMsh.AddTrace("UpdateManager"))
				break mainselect
			}

			// log res message
			for _, m := range resJson.Messages {
				errco.Logln(errco.LVL_B, m)
			}

			// check version result
			switch resJson.Result {
			case "dep": // local version deprecated
				// don't check if NotifyUpdate is set to true
				// override ConfigRuntime variables to display deprecated error message
				config.ConfigRuntime.Msh.InfoHibernation = "                   §fserver status:\n                   §b§lHIBERNATING\n                   §b§cmsh version DEPRECATED"
				config.ConfigRuntime.Msh.InfoStarting = "                   §fserver status:\n                    §6§lWARMING UP\n                   §b§cmsh version DEPRECATED"
				config.ConfigRuntime.Msh.NotifyUpdate = true

				notification := fmt.Sprintf("msh (%s) is deprecated: please update msh to %s!", MshVersion, resJson.Official.Version)
				errco.Logln(errco.LVL_A, notification)
				sgm.push.message = notification

			case "upd": // local version to update
				if config.ConfigRuntime.Msh.NotifyUpdate {
					notification := fmt.Sprintf("msh (%s) is now available: visit github to update!", resJson.Official.Version)
					errco.Logln(errco.LVL_A, notification)
					sgm.push.message = notification
				}

			case "off": // local version is official
				if config.ConfigRuntime.Msh.NotifyUpdate {
					errco.Logln(errco.LVL_A, "msh (%s) is updated", MshVersion)
				}

			case "dev": // local version is a developement version
				if config.ConfigRuntime.Msh.NotifyUpdate {
					errco.Logln(errco.LVL_A, "msh (%s) is running a dev release", MshVersion)
				}

			case "uno": // local version is unofficial
				if config.ConfigRuntime.Msh.NotifyUpdate {
					errco.Logln(errco.LVL_A, "msh (%s) is running an unofficial release", MshVersion)
				}

			default: // an error occurred
				errco.LogMshErr(errco.NewErr(errco.ERROR_VERSION, errco.LVL_D, "MshMgr", "invalid result from server"))
				break mainselect
			}
		}
	}
}

// sendApi2Req sends api2 request
func sendApi2Req(url string, api2req *model.Api2Req) (*http.Response, *errco.Error) {
	// before returning, communicate that update check is done
	defer func() {
		select {
		case CheckedUpdateC <- true:
		default:
		}
	}()

	errco.Logln(errco.LVL_D, "sendApi2Req: sending request...")

	// marshal request struct
	reqByte, err := json.Marshal(api2req)
	if err != nil {
		return nil, errco.NewErr(errco.ERROR_VERSION, errco.LVL_D, "sendApi2Req", err.Error())
	}

	// create http request
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqByte))
	if err != nil {
		return nil, errco.NewErr(errco.ERROR_VERSION, errco.LVL_D, "sendApi2Req", err.Error())
	}

	// add header User-Agent, Content-Type
	req.Header.Add("User-Agent", fmt.Sprintf("msh/%s (%s) %s", MshVersion, runtime.GOOS, runtime.GOARCH)) // format: msh/vx.x.x (linux) i386
	req.Header.Set("Content-Type", "application/json")                                                    // necessary for post request

	// execute http request
	errco.Logln(errco.LVL_E, "%smsh --> mshc%s:%v", errco.COLOR_PURPLE, errco.COLOR_RESET, string(reqByte))
	client := &http.Client{Timeout: 4 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, errco.NewErr(errco.ERROR_VERSION, errco.LVL_D, "sendApi2Req", err.Error())
	}

	return res, nil
}

// readApi2Res returns response in api2 struct
func readApi2Res(res *http.Response) (*model.Api2Res, *errco.Error) {
	defer res.Body.Close()

	errco.Logln(errco.LVL_D, "readApi2Res: reading response...")

	// read http response
	resByte, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errco.NewErr(errco.ERROR_VERSION, errco.LVL_D, "readApi2Res", err.Error())
	}
	errco.Logln(errco.LVL_E, "%smshc --> msh%s:%v", errco.COLOR_PURPLE, errco.COLOR_RESET, resByte)

	// load res data into resJson
	var resJson *model.Api2Res
	err = json.Unmarshal(resByte, &resJson)
	if err != nil {
		return nil, errco.NewErr(errco.ERROR_VERSION, errco.LVL_D, "readApi2Res", err.Error())
	}

	return resJson, nil
}
