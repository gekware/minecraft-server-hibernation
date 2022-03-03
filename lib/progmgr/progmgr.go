package progmgr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/model"
	"msh/lib/servctrl"
	"msh/lib/servstats"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
)

var (
	// msh version
	MshVersion string = "v2.4.5"

	// CheckedUpdateC communicates to main func that the first update check
	// has been done and msh can continue
	CheckedUpdateC chan bool = make(chan bool, 1)

	// api protocol version
	protv int = 2

	// msh program
	msh *program

	// segment used for stats
	sgm *segment
)

type program struct {
	startTime time.Time      // msh program start time
	sigExit   chan os.Signal // channel through which OS termination signals are notified
}

type segment struct {
	m *sync.Mutex // segment mutex

	tk          *time.Ticker  // segment ticker (every second)
	defDuration time.Duration // segment default duration
	startTime   time.Time     // segment start time
	end         *time.Timer   // segment end timer

	// stats are reset when segment reset is invoked
	stats struct {
		seconds     int
		secondsHibe int
		cpuUsage    float64
		memUsage    float64
		playerSec   int
		preTerm     bool
	}

	// push contains data for user notification
	push struct {
		tk      *time.Ticker // time ticker to send an update notification in chat
		message string       // the message shown by the notification
	}
}

// MshMgr handles exit signal and updates for msh
// [goroutine]
func MshMgr() {
	msh = &program{
		startTime: time.Now(),
		sigExit:   make(chan os.Signal, 1),
	}

	// set sigExit to relay termination signals
	signal.Notify(msh.sigExit, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT)

	// segment initialized to 0 so that update check can be executed immediately
	// must be reset to initialize all variables
	sgm = sgmReset(0)

	// start segment manager
	go sgm.sgmMgr()

	for {
		select {
		// msh termination signal is received
		case <-msh.sigExit:
			// stop the minecraft server with no player check
			errMsh := servctrl.StopMS(false)
			if errMsh != nil {
				errco.LogMshErr(errMsh.AddTrace("InterruptListener"))
			}

			// send last statistics before exiting
			go checkUpdReq(true)

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

		// check for update when segment ends
		case <-sgm.end.C:
			// send check update request
			errco.Logln(errco.LVL_D, "sending check update request...")
			res, errMsh := checkUpdReq(false)
			if errMsh != nil {
				errco.LogMshErr(errMsh.AddTrace("UpdateManager"))
				sgm.prolong(10 * time.Minute) // retry in 10 min
				continue
			}

			// data successfully received by server, reset segment
			errco.Logln(errco.LVL_D, "resetting segment...")
			sgm = sgmReset(sgm.defDuration)

			// analyze check update response
			errco.Logln(errco.LVL_D, "analyzing check update response...")
			errMsh = checkUpdAnalyze(res)
			if errMsh != nil {
				errco.LogMshErr(errMsh.AddTrace("UpdateManager"))
				continue
			}
		}

		// reminder: if you add here replace `continue` with `break` in select block
	}
}

// checkUpdReq logs segment stats and checks for updates.
func checkUpdReq(preTerm bool) (*http.Response, *errco.Error) {
	// before returning, communicate that update check is done
	defer func() {
		select {
		case CheckedUpdateC <- true:
		default:
		}
	}()

	// build request struct
	reqJson := buildReq(preTerm)

	// marshal request struct
	reqByte, err := json.Marshal(reqJson)
	if err != nil {
		return nil, errco.NewErr(errco.ERROR_VERSION, errco.LVL_D, "checkUpdReq", err.Error())
	}

	// build http request
	url := fmt.Sprintf("http://msh.gekware.net/api/v%d", protv)
	req, err := http.NewRequest("GET", url, bytes.NewReader(reqByte))
	if err != nil {
		return nil, errco.NewErr(errco.ERROR_VERSION, errco.LVL_D, "checkUpdReq", err.Error())
	}

	// add header User-Agent: msh/<msh-version> (<system-information>) <platform> (<platform-details>) <extensions>
	req.Header.Add("User-Agent", fmt.Sprintf("msh/%s (%s) %s", MshVersion, runtime.GOOS, runtime.GOARCH))

	errco.Logln(errco.LVL_E, "%smsh --> mshc%s:%v", errco.COLOR_PURPLE, errco.COLOR_RESET, reqByte)

	// execute http request
	client := &http.Client{Timeout: 4 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, errco.NewErr(errco.ERROR_VERSION, errco.LVL_D, "checkUpdReq", err.Error())
	}

	if res.StatusCode != 200 {
		return nil, errco.NewErr(errco.ERROR_VERSION, errco.LVL_D, "checkUpdReq", "response status code is "+res.Status)
	}

	return res, nil
}

// checkUpdAnalyze analyzes server response
func checkUpdAnalyze(res *http.Response) *errco.Error {
	defer res.Body.Close()

	// read http response
	resByte, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errco.NewErr(errco.ERROR_VERSION, errco.LVL_D, "checkUpdAnalyze", err.Error())
	}

	errco.Logln(errco.LVL_E, "%smshc --> msh%s:%v", errco.COLOR_PURPLE, errco.COLOR_RESET, resByte)

	// load res data into resJson
	var resJson *model.Api2Res
	err = json.Unmarshal(resByte, &resJson)
	if err != nil {
		return errco.NewErr(errco.ERROR_VERSION, errco.LVL_D, "checkUpdAnalyze", err.Error())
	}

	// compare MshVersion to versions from mshc server
	vStatus, errMsh := compareVersion(resJson, MshVersion)
	if errMsh != nil {
		return errMsh.AddTrace("checkUpdAnalyze")
	}

	// check version status
	switch vStatus {
	case errco.VERSION_DEP:
		// don't check if NotifyUpdate is set to true
		// override ConfigRuntime variables to display deprecated error message
		config.ConfigRuntime.Msh.InfoHibernation = "                   §fserver status:\n                   §b§lHIBERNATING\n                   §b§cmsh version DEPRECATED"
		config.ConfigRuntime.Msh.InfoStarting = "                   §fserver status:\n                    §6§lWARMING UP\n                   §b§cmsh version DEPRECATED"
		config.ConfigRuntime.Msh.NotifyUpdate = true

		notification := fmt.Sprintf("msh (%s) is deprecated: please update msh to %s!", MshVersion, resJson.Official.Version)

		// write in console log
		errco.Logln(errco.LVL_A, notification)

		// set push notification message
		sgm.push.message = notification

	case errco.VERSION_UPD:
		if config.ConfigRuntime.Msh.NotifyUpdate {
			notification := fmt.Sprintf("msh (%s) is now available: visit github to update!", resJson.Official.Version)

			// write in console log
			errco.Logln(errco.LVL_A, notification)

			// set push notification message
			sgm.push.message = notification
		}

	case errco.VERSION_OK:
		if config.ConfigRuntime.Msh.NotifyUpdate {
			// write in console log
			errco.Logln(errco.LVL_A, "msh (%s) is updated", MshVersion)
		}

	case errco.VERSION_DEV:
		if config.ConfigRuntime.Msh.NotifyUpdate {
			// write in console log
			errco.Logln(errco.LVL_A, "msh (%s) is running a dev release", MshVersion)
		}

	case errco.VERSION_UNO:
		if config.ConfigRuntime.Msh.NotifyUpdate {
			// write in console log
			errco.Logln(errco.LVL_A, "msh (%s) is running an unofficial release", MshVersion)
		}
	}

	return nil
}

// buildReq builds Api2Req
func buildReq(preTerm bool) *model.Api2Req {
	reqJson := &model.Api2Req{}

	reqJson.Protv = protv

	reqJson.Msh.Mshv = MshVersion
	reqJson.Msh.ID = config.ConfigRuntime.Msh.ID
	reqJson.Msh.Uptime = int(time.Since(msh.startTime).Seconds())
	reqJson.Msh.AllowSuspend = config.ConfigRuntime.Msh.AllowSuspend
	reqJson.Msh.Sgm.Seconds = sgm.stats.seconds
	reqJson.Msh.Sgm.SecondsHibe = sgm.stats.secondsHibe
	reqJson.Msh.Sgm.CpuUsage = sgm.stats.cpuUsage
	reqJson.Msh.Sgm.MemUsage = sgm.stats.memUsage
	reqJson.Msh.Sgm.PlayerSec = sgm.stats.playerSec
	reqJson.Msh.Sgm.PreTerm = preTerm

	reqJson.Machine.Os = runtime.GOOS
	reqJson.Machine.Platform = runtime.GOARCH
	reqJson.Machine.Javav = config.Javav
	reqJson.Machine.Stats.CoresMsh = runtime.NumCPU()
	if cores, err := cpu.Counts(true); err != nil {
		errco.LogMshErr(errco.NewErr(errco.ERROR_GET_CORES, errco.LVL_D, "buildReq", err.Error())) // non blocking error
		reqJson.Machine.Stats.Cores = -1
	} else {
		reqJson.Machine.Stats.Cores = cores
	}
	if memInfo, err := mem.VirtualMemory(); err != nil {
		errco.LogMshErr(errco.NewErr(errco.ERROR_GET_MEMORY, errco.LVL_D, "buildReq", err.Error())) // non blocking error
		reqJson.Machine.Stats.Mem = -1
	} else {
		reqJson.Machine.Stats.Mem = int(memInfo.Total)
	}

	reqJson.Server.Uptime = servctrl.TermUpTime()
	reqJson.Server.Minev = config.ConfigRuntime.Server.Version
	reqJson.Server.MineProt = config.ConfigRuntime.Server.Protocol

	return reqJson
}

// compareVersion compares version struct received from server with local version
func compareVersion(resJson *model.Api2Res, v string) (int, *errco.Error) {
	// check if there is a result
	if resJson.Result == "" {
		return 0, errco.NewErr(errco.ERROR_VERSION_INVALID, errco.LVL_D, "compareVersion", "result is invalid")
	}

	// digitize transforms a string "vx.x.x" into an integer x000x000x
	// returns errco.ERROR_VERSION_INVALID in case of error
	digitize := func(v string) int {
		vInt := 0

		// split version ("vx.x.x") to get a list of 3 integers
		vSplit := strings.Split(strings.ReplaceAll(v, "v", ""), ".")

		// vSplit should be 3 numbers
		if len(vSplit) != 3 {
			return errco.ERROR_VERSION_INVALID
		}

		// convert version to a single integer
		for i := 0; i < 3; i++ {
			digit, err := strconv.Atoi(vSplit[i])
			if err != nil {
				return errco.ERROR_VERSION_INVALID
			}
			vInt += digit * int(math.Pow(1000, float64(2-i)))
		}

		// format: x000x000x
		return vInt
	}

	// check that all versions have valid format
	switch errco.ERROR_VERSION_INVALID {
	case digitize(v):
		return 0, errco.NewErr(errco.ERROR_VERSION_INVALID, errco.LVL_D, "compareVersion", "msh local version is invalid")
	case digitize(resJson.Deprecated.Version):
		return 0, errco.NewErr(errco.ERROR_VERSION_INVALID, errco.LVL_D, "compareVersion", "msh deprecated version is invalid")
	case digitize(resJson.Official.Version):
		return 0, errco.NewErr(errco.ERROR_VERSION_INVALID, errco.LVL_D, "compareVersion", "msh official version is invalid")
	case digitize(resJson.Dev.Version):
		return 0, errco.NewErr(errco.ERROR_VERSION_INVALID, errco.LVL_D, "compareVersion", "msh dev version is invalid")
	}

	// compare versions
	switch {
	case digitize(v) <= digitize(resJson.Deprecated.Version):
		return errco.VERSION_DEP, nil
	case digitize(v) < digitize(resJson.Official.Version):
		return errco.VERSION_UPD, nil
	case digitize(v) == digitize(resJson.Official.Version):
		return errco.VERSION_OK, nil
	case digitize(v) <= digitize(resJson.Dev.Version):
		return errco.VERSION_DEV, nil
	default:
		// v is greater than dev
		return errco.VERSION_UNO, nil
	}
}

// ------------------------ segment ------------------------ //

// sgmMgr handles segment and all variables related
// [goroutine]
func (sgm *segment) sgmMgr() {
	for {
		select {

		// segment 1 second tick
		case <-sgm.tk.C:
			sgm.m.Lock()

			// increment segment second counter
			sgm.stats.seconds += 1

			// increment work/hibernation second counter
			if !servctrl.ServTerm.IsActive {
				sgm.stats.secondsHibe += 1
			}

			// treeProc returns the list of tree pids (also original ppid)
			var treeProc func(pid *process.Process) []*process.Process
			treeProc = func(proc *process.Process) []*process.Process {
				children, err := proc.Children()
				if err != nil {
					// set pid to -1 to indicate that an error occurred
					proc.Pid = -1
					return []*process.Process{proc}
				}

				tree := []*process.Process{proc}
				for _, child := range children {
					tree = append(tree, treeProc(child)...)
				}
				return tree
			}

			// update segment average cpu/memory usage
			var mshTreeCpu, mshTreeMem float64 = 0, 0
			mshProc, _ := process.NewProcess(int32(os.Getpid())) // don't check for error, if mshProc *process.Process is invalid it will be caught in treeProc()
			for _, c := range treeProc(mshProc) {
				if pCpu, err := c.CPUPercent(); err != nil {
					mshTreeCpu = -1
					mshTreeMem = -1
					break
				} else if pMem, err := c.MemoryPercent(); err != nil {
					mshTreeCpu = -1
					mshTreeMem = -1
					break
				} else {
					mshTreeCpu += float64(pCpu)
					mshTreeMem += float64(pMem)
				}
			}

			sgm.stats.cpuUsage = (sgm.stats.cpuUsage*float64(sgm.stats.seconds-1) + float64(mshTreeCpu)) / float64(sgm.stats.seconds) // sgm.stats.seconds-1 because the average is updated to 1 sec ago
			sgm.stats.memUsage = (sgm.stats.memUsage*float64(sgm.stats.seconds-1) + float64(mshTreeMem)) / float64(sgm.stats.seconds)

			// update play seconds sum
			sgm.stats.playerSec = servstats.Stats.PlayerCount

			sgm.m.Unlock() // not using defer since it's an infinite loop

		// send a notification in game chat for players to see.
		// (should not send notification in console)
		case <-sgm.push.tk.C:
			if config.ConfigRuntime.Msh.NotifyUpdate && sgm.push.message != "" && servctrl.ServTerm.IsActive {
				_, errMsh := servctrl.Execute("say "+sgm.push.message, "sgmMgr")
				if errMsh != nil {
					errco.LogMshErr(errMsh.AddTrace("sgmMgr"))
				}
			}
		}
	}
}

// reset resets segment variables
func sgmReset(sgmDur time.Duration) *segment {
	sgm = &segment{}

	sgm.m = &sync.Mutex{}

	sgm.tk = time.NewTicker(time.Second)
	sgm.defDuration = 4 * time.Hour
	sgm.startTime = time.Now()
	sgm.end = time.NewTimer(sgmDur)

	sgm.stats.seconds = 0
	sgm.stats.secondsHibe = 0
	sgm.stats.cpuUsage = 0
	sgm.stats.memUsage = 0
	sgm.stats.playerSec = 0
	sgm.stats.preTerm = false

	sgm.push.tk = time.NewTicker(20 * time.Minute)
	sgm.push.message = ""

	return sgm
}

// prolong prolongs segment end timer. Should be called only when sgm.(*time.Timer).C has been drained
func (sgm *segment) prolong(sgmDur time.Duration) {
	sgm.m.Lock()
	defer sgm.m.Unlock()

	sgm.end.Reset(sgmDur)
}
