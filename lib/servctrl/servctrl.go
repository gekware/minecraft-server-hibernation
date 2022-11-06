package servctrl

import (
	"fmt"
	"sync/atomic"
	"time"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/opsys"
	"msh/lib/servstats"
)

// WarmMS warms the minecraft server
// [non-blocking]
func WarmMS() *errco.Error {
	// don't try to warm ms if it has encountered major errors
	if servstats.Stats.MajorError != nil {
		return errco.NewErr(errco.ERROR_MINECRAFT_SERVER, errco.LVL_1, "StartMS", "minecraft server has encountered major problems")
	}

	errco.Logln(errco.LVL_3, "warming minecraft server...")

	switch servstats.Stats.Status {

	case errco.SERVER_STATUS_OFFLINE:
		// ms is offline, log error if ms process is set to suspended

		if servstats.Stats.Suspended {
			errco.LogMshErr(errco.NewErr(errco.ERROR_SERVER_OFFLINE_SUSPENDED, errco.LVL_3, "WarmMS", "minecraft server is suspended and offline"))
			servstats.Stats.Suspended = false // if ms is offline it's process can't be suspended
		}

		errMsh := termStart(config.ConfigRuntime.Server.Folder, config.ConfigRuntime.Commands.StartServer)
		if errMsh != nil {
			servstats.Stats.SetMajorError(errco.NewErr(errco.ERROR_MINECRAFT_SERVER, errco.LVL_3, "StartMS", "error starting minecraft server (check logs)"))
			return errMsh.AddTrace("WarmMS")
		}

	default:
		if servstats.Stats.Suspended {
			var errMsh *errco.Error
			servstats.Stats.Suspended, errMsh = opsys.ProcTreeResume(uint32(ServTerm.cmd.Process.Pid))
			if errMsh != nil {
				return errMsh.AddTrace("WarmMS")
			}
		} else {
			errco.LogWarn(errco.NewErr(errco.ERROR_SERVER_IS_WARM, errco.LVL_3, "WarmMS", "minecraft server already warm"))
		}
	}

	// request a soft freeze
	FreezeMSRequest()

	return nil
}

// FreezeMS executes "stop" command on the minecraft server.
// When force == true, it does not perform player check and orders the server shutdown (according to ms status)
func FreezeMS(force bool) *errco.Error {
	errco.Logln(errco.LVL_3, "freezing minecraft server...")

	switch servstats.Stats.Status {

	case errco.SERVER_STATUS_OFFLINE:
		// ms is offline, log error if ms process is set to suspended

		if servstats.Stats.Suspended {
			errco.LogMshErr(errco.NewErr(errco.ERROR_SERVER_OFFLINE_SUSPENDED, errco.LVL_3, "FreezeMS", "minecraft server is suspended and offline"))
			servstats.Stats.Suspended = false // if ms is offline it's process can't be suspended
		}

		errco.LogWarn(errco.NewErr(errco.ERROR_SERVER_IS_FROZEN, errco.LVL_3, "FreezeMS", "minecraft server already frozen"))

	case errco.SERVER_STATUS_STARTING:
		// ms is starting, resume the ms process, wait for status online and then freeze ms

		var errMsh *errco.Error

		// resume ms process if suspended
		if servstats.Stats.Suspended {
			servstats.Stats.Suspended, errMsh = opsys.ProcTreeResume(uint32(ServTerm.cmd.Process.Pid))
			if errMsh != nil {
				return errMsh.AddTrace("FreezeMS")
			}
		}

		// wait for ms to go online
		for servstats.Stats.Status == errco.SERVER_STATUS_STARTING {
			time.Sleep(1 * time.Second)
		}

		// if ms is not online return
		if servstats.Stats.Status != errco.SERVER_STATUS_ONLINE {
			return errco.NewErr(errco.ERROR_SERVER_NOT_ONLINE, errco.LVL_3, "FreezeMS", "server is not online")
		}

		// now it's the case of the ms status online
		fallthrough

	case errco.SERVER_STATUS_ONLINE:
		// is ms is online, resume the process and then stop it

		var errMsh *errco.Error

		// execute ms freeze
		if force {
			// if forceful freeze, execute ms stop
			errMsh = executeMSStop()
			if errMsh != nil {
				return errMsh.AddTrace("FreezeMS")
			}
		} else if errMsh = readyToFreezeMS(); errMsh == nil {
			// if soft freeze and ms can be stopped, suspend/stop
			if config.ConfigRuntime.Msh.AllowSuspend {
				if !servstats.Stats.Suspended {
					servstats.Stats.Suspended, errMsh = opsys.ProcTreeSuspend(uint32(ServTerm.cmd.Process.Pid))
					if errMsh != nil {
						return errMsh.AddTrace("FreezeMS")
					}
				}
			} else {
				errMsh = executeMSStop()
				if errMsh != nil {
					return errMsh.AddTrace("FreezeMS")
				}
			}
		} else {
			errco.LogMshErr(errMsh.AddTrace("FreezeMS"))
		}

	case errco.SERVER_STATUS_STOPPING:
		// is ms is stopping, resume the process and let it stop

		// resume ms process if suspended
		if servstats.Stats.Suspended {
			var errMsh *errco.Error
			servstats.Stats.Suspended, errMsh = opsys.ProcTreeResume(uint32(ServTerm.cmd.Process.Pid))
			if errMsh != nil {
				return errMsh.AddTrace("FreezeMS")
			}
		}

		// wait for ms to go offline
		for servstats.Stats.Status == errco.SERVER_STATUS_STOPPING {
			time.Sleep(1 * time.Second)
		}
	}

	return nil
}

// executeMSStop resumes ms process and executes a stop command in ms terminal.
// should be called only when ms status is online
func executeMSStop() *errco.Error {
	var errMsh *errco.Error

	// resume ms process if suspended
	if servstats.Stats.Suspended {
		servstats.Stats.Suspended, errMsh = opsys.ProcTreeResume(uint32(ServTerm.cmd.Process.Pid))
		if errMsh != nil {
			return errMsh.AddTrace("executeMSStop")
		}
	}

	// execute stop command
	_, errMsh = Execute(config.ConfigRuntime.Commands.StopServer, "executeMSStop")
	if errMsh != nil {
		return errMsh.AddTrace("executeMSStop")
	}

	// if sigint is allowed, launch a function to check the shutdown of minecraft server
	if config.ConfigRuntime.Commands.StopServerAllowKill > 0 {
		go killMSifOnlineAfterTimeout()
	}

	return nil
}

// readyToFreezeMS returns nil if server is ready to be frozen (depending on player count)
func readyToFreezeMS() *errco.Error {
	// check that there is only one FreezeMSRequest running and players <= 0,
	// if so proceed with server shutdown
	atomic.AddInt32(&servstats.Stats.FreezeMSRequests, -1)

	// check how many players are on the server
	playerCount, method := countPlayerSafe()
	errco.Logln(errco.LVL_1, "%d online players - method for player count: %s", playerCount, method)
	if playerCount > 0 {
		return errco.NewErr(errco.ERROR_SERVER_NOT_EMPTY, errco.LVL_3, "readyToFreezeMS", "server is not empty")
	}

	// check if enough time has passed since last player disconnected
	if atomic.LoadInt32(&servstats.Stats.FreezeMSRequests) > 0 {
		return errco.NewErr(errco.ERROR_SERVER_MUST_WAIT, errco.LVL_3, "readyToFreezeMS", fmt.Sprintf("not enough time has passed since last player disconnected (FreezeMSRequests: %d)", servstats.Stats.FreezeMSRequests))
	}

	return nil
}

// FreezeMSRequest increases FreezeMSRequests by one and starts the timer to execute soft minecraft server shutdown
// [goroutine]
func FreezeMSRequest() {
	// add 1 freeze ms request
	atomic.AddInt32(&servstats.Stats.FreezeMSRequests, 1)

	// [goroutine]
	time.AfterFunc(
		time.Duration(config.ConfigRuntime.Msh.TimeBeforeStoppingEmptyServer)*time.Second,
		func() {
			// stop minecraft server softly
			errMsh := FreezeMS(false)
			if errMsh != nil {
				errco.LogWarn(errMsh.AddTrace("FreezeMSRequest"))
			}
		},
	)
}

// killMSifOnlineAfterTimeout waits for the specified time and then if the server is still online, kills the server process
func killMSifOnlineAfterTimeout() {
	var errMsh *errco.Error

	countdown := config.ConfigRuntime.Commands.StopServerAllowKill

	// resume ms process if suspended
	if servstats.Stats.Suspended {
		servstats.Stats.Suspended, errMsh = opsys.ProcTreeResume(uint32(ServTerm.cmd.Process.Pid))
		if errMsh != nil {
			errco.LogMshErr(errMsh.AddTrace("killMSifOnlineAfterTimeout"))
		}
	}

	for countdown > 0 {
		// if server goes offline it's the correct behaviour -> return
		if servstats.Stats.Status == errco.SERVER_STATUS_OFFLINE {
			return
		}

		countdown--
		time.Sleep(time.Second)
	}

	// save world before killing the server, do not check for errors
	errco.Logln(errco.LVL_3, "saving word before killing the minecraft server process")
	_, _ = Execute("save-all", "killMSifOnlineAfterTimeout")

	// give time to save word
	time.Sleep(10 * time.Second)

	// send kill signal to server
	errco.Logln(errco.LVL_3, "minecraft server process won't stop normally: sending kill signal")
	err := ServTerm.cmd.Process.Kill()
	if err != nil {
		errco.LogMshErr(errco.NewErr(errco.ERROR_SERVER_KILL, errco.LVL_3, "killMSifOnlineAfterTimeout", err.Error()))
	}
}
