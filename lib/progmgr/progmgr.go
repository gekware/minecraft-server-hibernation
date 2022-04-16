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
	MshVersion string = "v2.4.6"  // msh version
	MshCommit  string = "-------" // msh commit

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

	// start segment manager
	go sgmMgr()

	for {
		// msh termination signal is received
		<-msh.sigExit

		// stop the minecraft server forcefully
		errMsh := servctrl.FreezeMS(true)
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
	}
}
