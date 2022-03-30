package progmgr

import (
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"msh/lib/errco"
	"msh/lib/servctrl"
	"msh/lib/servstats"

	"github.com/shirou/gopsutil/process"
)

// segment used for stats
var sgm *segment = &segment{
	m: &sync.Mutex{},
}

type segment struct {
	m *sync.Mutex // segment mutex (initialized with sgm and not affected by reset function)

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
		tk       *time.Ticker // time ticker to send an update notification in chat
		messages []string     // the message shown by the notification
	}
}

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

			// update segment average cpu/memory usage
			var mshTreeCpu, mshTreeMem float64 = 0, 0
			if mshProc, err := process.NewProcess(int32(os.Getpid())); err != nil {
				break
			} else {
				for _, c := range treeProc(mshProc) {
					if pCpu, err := c.CPUPercent(); err != nil {
						mshTreeCpu = sgm.stats.cpuUsage
						mshTreeMem = sgm.stats.memUsage
						break
					} else if pMem, err := c.MemoryPercent(); err != nil {
						mshTreeCpu = sgm.stats.cpuUsage
						mshTreeMem = sgm.stats.memUsage
						break
					} else {
						mshTreeCpu += float64(pCpu)
						mshTreeMem += float64(pMem)
					}
				}
			}
			sgm.stats.cpuUsage = (sgm.stats.cpuUsage*float64(sgm.stats.seconds-1) + float64(mshTreeCpu)) / float64(sgm.stats.seconds) // sgm.stats.seconds-1 because the average is relative to 1 sec ago
			sgm.stats.memUsage = (sgm.stats.memUsage*float64(sgm.stats.seconds-1) + float64(mshTreeMem)) / float64(sgm.stats.seconds)

			// update play seconds sum
			sgm.stats.playerSec = servstats.Stats.PlayerCount

			sgm.m.Unlock() // not using defer since it's an infinite loop

		// send a notification in game chat for players to see.
		// (should not send notification in console)
		case <-sgm.push.tk.C:
			if len(sgm.push.messages) != 0 && servstats.Stats.PlayerCount > 0 {
				for _, m := range sgm.push.messages {
					_, errMsh := servctrl.Execute("say "+m, "sgmMgr")
					if errMsh != nil {
						errco.LogMshErr(errMsh.AddTrace("sgmMgr"))
					}
				}
			}
		}
	}
}

// reset segment variables
// accepted parameters types: int, time.Duration, *http.Response
func (sgm *segment) reset(i interface{}) *segment {
	sgm.tk = time.NewTicker(time.Second)
	sgm.defDuration = 4 * time.Hour
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
			sgm.end = time.NewTimer(sgm.defDuration)
		}
	default:
		sgm.end = time.NewTimer(sgm.defDuration)
	}

	sgm.stats.seconds = 0
	sgm.stats.secondsHibe = 0
	sgm.stats.cpuUsage = 0
	sgm.stats.memUsage = 0
	sgm.stats.playerSec = 0
	sgm.stats.preTerm = false

	sgm.push.tk = time.NewTicker(20 * time.Minute)
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
			sgm.end.Reset(sgm.defDuration)
		}
	default:
		sgm.end.Reset(sgm.defDuration)
	}
}
