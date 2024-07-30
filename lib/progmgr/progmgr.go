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

/*
COMMIT COLLECTION
- this is commit  700!
- this is commit  800!
- this is commit  900!
- this is commit 1000!
*/

var (
	MshVersion string = "v2.5.0"  // msh version
	MshCommit  string = "-------" // msh commit

	// msh program
	msh *program = &program{
		startTime: time.Now(),
		sigExit:   make(chan os.Signal, 1),
		sigUser:   make(chan os.Signal, 1),
		mgrActive: false,
	}
)

type program struct {
	startTime time.Time      // msh program start time
	sigExit   chan os.Signal // channel through which OS termination signals are notified
	sigUser   chan os.Signal // channel through which User-defined signals are notified
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

	SetupUserSignalHandling(msh)

	msh.mgrActive = true

	for {
		// msh termination signal is received
		sig := <-msh.sigExit
		errco.NewLogln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "received signal: %s", sig.String())

		// stop the minecraft server forcefully
		logMsh := servctrl.FreezeMS(true)
		if logMsh != nil {
			logMsh.Log(true)
		}

		// send last statistics before exiting
		go sendApi2Req(updAddr, buildApi2Req(true))

		// wait 1 second to let the server go into stopping mode
		time.Sleep(1 * time.Second)

		switch servstats.Stats.Status {
		case errco.SERVER_STATUS_STOPPING:
			// if server is correctly stopping, wait for minecraft server to exit
			errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "waiting for minecraft server terminal to exit (minecraft server is stopping)")
			servctrl.ServTerm.Wg.Wait()

		case errco.SERVER_STATUS_OFFLINE:
			// if server is offline, then it's safe to continue
			errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "minecraft server terminal already exited (minecraft server is offline)")

		default:
			errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "stop command does not seem to be stopping minecraft server during forceful shutdown")
		}

		// exit
		errco.NewLogln(errco.TYPE_INF, errco.LVL_0, errco.ERROR_NIL, "exiting msh")
		os.Exit(0)
	}
}

// AutoTerminate induces correct msh termination via msh manager
func AutoTerminate() {
	errco.NewLogln(errco.TYPE_INF, errco.LVL_0, errco.ERROR_NIL, "issuing msh termination")
	if msh.mgrActive {
		// send signal to msh.sigExit so that msh manager handles msh termination
		msh.sigExit <- syscall.SIGINT
	} else {
		// msh manager still not running, just exit with non 0 value
		os.Exit(1)
	}
}
