package servctrl

import (
	"fmt"
	"sync/atomic"
	"time"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/servstats"
)

// StartMS starts the minecraft server
func StartMS() *errco.Error {
	// start server terminal
	errMsh := termStart(config.ConfigRuntime.Server.Folder, config.ConfigRuntime.Commands.StartServer)
	if errMsh != nil {
		return errMsh.AddTrace("StartMS")
	}

	return nil
}

// StopMS executes "stop" command on the minecraft server.
// When playersCheck == true, it checks for StopMSRequests/Players and orders the server shutdown
func StopMS(playersCheck bool) *errco.Error {
	// wait for the starting server to go online
	for servstats.Stats.Status == errco.SERVER_STATUS_STARTING {
		time.Sleep(1 * time.Second)
	}
	// if server is not online return
	if servstats.Stats.Status != errco.SERVER_STATUS_ONLINE {
		return errco.NewErr(errco.ERROR_SERVER_NOT_ONLINE, errco.LVL_D, "StopMS", "server is not online")
	}

	// player/StopMSRequests check
	if playersCheck {
		// check that there is only one StopMSRequest running and players <= 0,
		// if so proceed with server shutdown
		atomic.AddInt32(&servstats.Stats.StopMSRequests, -1)

		// check how many players are on the server
		playerCount, method := countPlayerSafe()
		errco.Logln(errco.LVL_B, "%d online players - method for player count: %s", playerCount, method)
		if playerCount > 0 {
			return errco.NewErr(errco.ERROR_SERVER_NOT_EMPTY, errco.LVL_D, "StopMS", "server is not empty")
		}

		// check if enough time has passed since last player disconnected

		if atomic.LoadInt32(&servstats.Stats.StopMSRequests) > 0 {
			return errco.NewErr(errco.ERROR_SERVER_MUST_WAIT, errco.LVL_D, "StopMS", fmt.Sprintf("not enough time has passed since last player disconnected (StopMSRequests: %d )", servstats.Stats.StopMSRequests))
		}
	}

	// execute stop command
	_, errMsh := Execute(config.ConfigRuntime.Commands.StopServer, "StopMS")
	if errMsh != nil {
		return errMsh.AddTrace("StopMS")
	}

	// if sigint is allowed, launch a function to check the shutdown of minecraft server
	if config.ConfigRuntime.Commands.StopServerAllowKill > 0 {
		go killMSifOnlineAfterTimeout()
	}

	return nil
}

// StopMSRequest increases StopMSRequests by one and starts the timer to execute StopMS(true) (with playersCheck)
// [goroutine]
func StopMSRequest() {
	atomic.AddInt32(&servstats.Stats.StopMSRequests, 1)

	// [goroutine]
	time.AfterFunc(
		time.Duration(config.ConfigRuntime.Msh.TimeBeforeStoppingEmptyServer)*time.Second,
		func() {
			errMsh := StopMS(true)
			if errMsh != nil {
				// avoid logging "server is not online" error since it can be very frequent
				if errMsh.Cod != errco.ERROR_SERVER_NOT_ONLINE {
					errco.LogMshErr(errMsh.AddTrace("StopMSRequest"))
				}
			}
		})
}

// killMSifOnlineAfterTimeout waits for the specified time and then if the server is still online, kills the server process
func killMSifOnlineAfterTimeout() {
	countdown := config.ConfigRuntime.Commands.StopServerAllowKill

	for countdown > 0 {
		// if server goes offline it's the correct behaviour -> return
		if servstats.Stats.Status == errco.SERVER_STATUS_OFFLINE {
			return
		}

		countdown--
		time.Sleep(time.Second)
	}

	// save world before killing the server, do not check for errors
	errco.Logln(errco.LVL_D, "saving word before killing the minecraft server process")
	_, _ = Execute("save-all", "killMSifOnlineAfterTimeout")

	// give time to save word
	time.Sleep(10 * time.Second)

	// send kill signal to server
	errco.Logln(errco.LVL_D, "minecraft server process won't stop normally: sending kill signal")
	err := ServTerm.cmd.Process.Kill()
	if err != nil {
		errco.LogMshErr(errco.NewErr(errco.ERROR_SERVER_KILL, errco.LVL_D, "killMSifOnlineAfterTimeout", err.Error()))
	}
}
