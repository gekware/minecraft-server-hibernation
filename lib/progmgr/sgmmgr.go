package progmgr

import (
	"os"
	"sync"
	"time"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/servctrl"
	"msh/lib/servstats"

	"github.com/shirou/gopsutil/process"
)

// segment used for stats
var sgm *segment = &segment{}

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
func (sgm *segment) reset(sgmDur time.Duration) *segment {
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
