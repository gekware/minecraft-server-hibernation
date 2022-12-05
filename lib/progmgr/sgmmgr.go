package progmgr

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/servctrl"
	"msh/lib/servstats"
)

var (
	// ReqSent communicates to main func that the first request is completed and msh can continue
	ReqSent chan bool = make(chan bool, 1)

	protv   int    = 2                                                              // api protocol version
	updAddr string = fmt.Sprintf("https://msh.gekware.net/api/v%d/versions", protv) // server address to get version info

	// segment used for stats
	sgm *segment = &segment{
		m:      &sync.Mutex{},
		tk:     time.NewTicker(time.Second),
		defDur: 4 * time.Hour,
	}
)

type segment struct {
	m *sync.Mutex // segment mutex (initialized with sgm and not affected by reset function)

	tk        *time.Ticker  // segment ticker (every second)
	defDur    time.Duration // segment default duration
	startTime time.Time     // segment start time
	end       *time.Timer   // segment end timer

	// stats are reset when segment reset is invoked
	stats struct {
		preTerm  bool
		dur      int
		hibeDur  int
		usageCpu float64
		usageMem float64
		playSec  int
	}

	// push contains data for user notification
	push struct {
		tk       *time.Ticker // time ticker to send an update notification in chat
		verCheck string       // version check result
		messages []string     // the message shown by the notification
	}
}

// sgmMgr handles segment and all variables related
// [goroutine]
func sgmMgr() {
	// initialize sgm variables
	sgm.reset(0) // segment duration initialized to 0 so that the first request can be executed immediately

	for {
	mainselect:
		select {

		// segment 1 second tick
		case <-sgm.tk.C:
			sgm.m.Lock()

			// increment segment duration counter
			sgm.stats.dur += 1

			// increment hibernation duration counter if ms is not warm/interactable
			logMsh := servctrl.CheckMSWarm()
			if logMsh != nil {
				sgm.stats.hibeDur += 1
			}

			// increment play seconds sum
			sgm.stats.playSec += servstats.Stats.ConnCount

			// update segment average cpu/memory usage
			mshTreeCpu, mshTreeMem := getMshTreeStats()
			sgm.stats.usageCpu = (sgm.stats.usageCpu*float64(sgm.stats.dur-1) + float64(mshTreeCpu)) / float64(sgm.stats.dur) // sgm.stats.seconds-1 because the average is relative to 1 sec ago
			sgm.stats.usageMem = (sgm.stats.usageMem*float64(sgm.stats.dur-1) + float64(mshTreeMem)) / float64(sgm.stats.dur)

			sgm.m.Unlock() // not using defer since it's an infinite loop

		// send a notification in game chat for players to see.
		// (should not send notification in console)
		case <-sgm.push.tk.C:
			if sgm.push.verCheck != "" && servstats.Stats.ConnCount > 0 {
				logMsh := servctrl.TellRaw("manager", sgm.push.verCheck, "sgmMgr")
				if logMsh != nil {
					logMsh.Log(true)
				}
			}

			if len(sgm.push.messages) != 0 && servstats.Stats.ConnCount > 0 {
				for _, m := range sgm.push.messages {
					logMsh := servctrl.TellRaw("message", m, "sgmMgr")
					if logMsh != nil {
						logMsh.Log(true)
					}
				}
			}

		// send request when segment ends
		case <-sgm.end.C:
			// send request
			res, logMsh := sendApi2Req(updAddr, buildApi2Req(false))
			if logMsh != nil {
				logMsh.Log(true)
				sgm.prolong(10 * time.Minute)
				break mainselect
			}

			// check response status code
			switch res.StatusCode {
			case 200:
				errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "segment reset")
				sgm.reset(res)
			case 403:
				errco.NewLogln(errco.TYPE_WAR, errco.LVL_0, errco.ERROR_VERSION, "client is unauthorized, issuing msh termination")
				AutoTerminate()
			default:
				errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_VERSION, "response status code is %s -> prolonging segment...", res.Status)
				sgm.prolong(res)
				break mainselect
			}

			// get server response into struct
			resJson, logMsh := readApi2Res(res)
			if logMsh != nil {
				logMsh.Log(true)
				break mainselect
			}

			// check version result
			switch resJson.Result {
			case "dep": // local version deprecated
				// don't check NotifyUpdate
				verCheck := fmt.Sprintf("msh (%s) is deprecated: visit github to update to %s!", MshVersion, resJson.Official.V)
				errco.NewLogln(errco.TYPE_WAR, errco.LVL_0, errco.ERROR_VERSION, verCheck)
				sgm.push.verCheck = verCheck

				// override ConfigRuntime variables to display deprecated error message in motd
				config.ConfigRuntime.Msh.InfoHibernation = "                   §fserver status:\n                   §b§lHIBERNATING\n                   §b§cmsh version DEPRECATED"
				config.ConfigRuntime.Msh.InfoStarting = "                   §fserver status:\n                    §6§lWARMING UP\n                   §b§cmsh version DEPRECATED"

			case "upd": // local version to update
				if config.ConfigRuntime.Msh.NotifyUpdate {
					verCheck := fmt.Sprintf("msh (%s) can be updated: visit github to update to %s!", MshVersion, resJson.Official.V)
					errco.NewLogln(errco.TYPE_WAR, errco.LVL_0, errco.ERROR_VERSION, verCheck)
					sgm.push.verCheck = verCheck
				}

			case "off": // local version is official
				if config.ConfigRuntime.Msh.NotifyUpdate {
					verCheck := fmt.Sprintf("msh (%s) is updated", MshVersion)
					errco.NewLogln(errco.TYPE_INF, errco.LVL_0, errco.ERROR_NIL, verCheck)
					sgm.push.verCheck = verCheck
				}

			case "dev": // local version is a developement version
				if config.ConfigRuntime.Msh.NotifyUpdate {
					verCheck := fmt.Sprintf("msh (%s) is running a dev release", MshVersion)
					errco.NewLogln(errco.TYPE_WAR, errco.LVL_0, errco.ERROR_VERSION, verCheck)
					sgm.push.verCheck = verCheck
				}

			case "uno": // local version is unofficial
				if config.ConfigRuntime.Msh.NotifyUpdate {
					verCheck := fmt.Sprintf("msh (%s) is running an unofficial release", MshVersion)
					errco.NewLogln(errco.TYPE_WAR, errco.LVL_0, errco.ERROR_VERSION, verCheck)
					sgm.push.verCheck = verCheck
				}

			default: // an error occurred
				if config.ConfigRuntime.Msh.NotifyUpdate {
					errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_VERSION, "invalid version result from server")
				}
			}

			// log response messages
			if config.ConfigRuntime.Msh.NotifyMessage {
				for _, m := range resJson.Messages {
					errco.NewLogln(errco.TYPE_INF, errco.LVL_0, errco.ERROR_NIL, "message from the moon: %s", m)
				}
				sgm.push.messages = append(sgm.push.messages, resJson.Messages...)
			}
		}
	}
}

// reset segment variables
// accepted parameters types: int, time.Duration, *http.Response
func (sgm *segment) reset(i interface{}) *segment {
	sgm.startTime = time.Now()
	switch v := i.(type) {
	case int:
		sgm.end = time.NewTimer(time.Duration(v) * time.Second)
	case time.Duration:
		sgm.end = time.NewTimer(v)
	case *http.Response:
		if xrr, err := strconv.Atoi(v.Header.Get("x-ratelimit-reset")); err == nil {
			sgm.end = time.NewTimer(time.Duration(xrr) * time.Second)
		} else {
			sgm.end = time.NewTimer(sgm.defDur)
		}
	default:
		sgm.end = time.NewTimer(sgm.defDur)
	}

	sgm.stats.dur = 0
	sgm.stats.hibeDur = 0
	sgm.stats.usageCpu, sgm.stats.usageMem = getMshTreeStats()
	sgm.stats.playSec = 0
	sgm.stats.preTerm = false

	sgm.push.tk = time.NewTicker(20 * time.Minute)
	sgm.push.verCheck = ""
	sgm.push.messages = []string{}

	return sgm
}

// prolong prolongs segment end timer. Should be called only when sgm.(*time.Timer).C has been drained
// accepted parameters types: int, time.Duration, *http.Response
func (sgm *segment) prolong(i interface{}) {
	sgm.m.Lock()
	defer sgm.m.Unlock()

	switch v := i.(type) {
	case int:
		sgm.end.Reset(time.Duration(v) * time.Second)
	case time.Duration:
		sgm.end = time.NewTimer(v)
	case *http.Response:
		if xrr, err := strconv.Atoi(v.Header.Get("x-ratelimit-reset")); err == nil {
			sgm.end.Reset(time.Duration(xrr) * time.Second)
		} else {
			sgm.end.Reset(sgm.defDur)
		}
	default:
		sgm.end.Reset(sgm.defDur)
	}
}
