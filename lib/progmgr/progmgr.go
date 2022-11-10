package progmgr

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"msh/lib/errco"
	"msh/lib/servctrl"
	"msh/lib/servstats"
)

var (
	MshVersion string = "v2.4.8"  // msh version
	MshCommit  string = "-------" // msh commit

	// msh program
	msh *program = &program{
		startTime: time.Now(),
		sigExit:   make(chan os.Signal, 1),
		mgrActive: false,
	}
)

type program struct {
	startTime time.Time      // msh program start time
	sigExit   chan os.Signal // channel through which OS termination signals are notified
	mgrActive bool           // indicates if msh manager is running
}

// MshMgr handles exit signal and updates for msh.
// After this function is called, msh should exit by sending itself a termination signal.
// [goroutine]
func MshMgr() {
	// start segment manager
	go sgmMgr()

	// set msh.sigExit to relay termination signals
	signal.Notify(msh.sigExit, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT)

	msh.mgrActive = true

	for {
		// msh termination signal is received
		<-msh.sigExit

		// stop the minecraft server forcefully
		logMsh := servctrl.FreezeMS(true)
		if logMsh != nil {
			errco.Log(logMsh.AddTrace())
		}

		// send last statistics before exiting
		go sendApi2Req(updAddr, buildApi2Req(true))

		// wait 1 second to let the server go into stopping mode
		time.Sleep(time.Second)

		switch servstats.Stats.Status {
		case errco.SERVER_STATUS_STOPPING:
			// if server is correctly stopping, wait for minecraft server to exit
			errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "waiting for minecraft server terminal to exit (server is stopping)")
			servctrl.ServTerm.Wg.Wait()

		case errco.SERVER_STATUS_OFFLINE:
			// if server is offline, then it's safe to continue
			errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "minecraft server terminal already exited (server is offline)")

		default:
			errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "stop command does not seem to be stopping server during forceful shutdown")
		}

		// exit
		errco.Logln(errco.TYPE_INF, errco.LVL_0, errco.ERROR_NIL, "exiting msh")
		os.Exit(0)
	}
}

// AutoTerminate induces correct msh termination via msh manager
func AutoTerminate() {
	if msh.mgrActive {
		// send signal to msh.sigExit so that msh manager handles msh termination
		msh.sigExit <- syscall.SIGINT
	} else {
		// msh manager still not running, just exit with non 0 value
		os.Exit(1)
	}
}
