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
func WarmMS() *errco.MshLog {
	// don't try to warm ms if it has encountered major errors
	if servstats.Stats.MajorError != nil {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_1, errco.ERROR_MINECRAFT_SERVER, "minecraft server has encountered major problems")
	}

	errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "warming minecraft server...")

	switch servstats.Stats.Status {

	case errco.SERVER_STATUS_OFFLINE:
		// ms is offline, log error if ms process is set to suspended

		if servstats.Stats.Suspended {
			errco.Logln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_OFFLINE_SUSPENDED, "minecraft server is suspended and offline")
			servstats.Stats.Suspended = false // if ms is offline it's process can't be suspended
		}

		logMsh := termStart(config.ConfigRuntime.Server.Folder, config.ConfigRuntime.Commands.StartServer)
		if logMsh != nil {
			servstats.Stats.SetMajorError(errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_MINECRAFT_SERVER, "error starting minecraft server (check logs)"))
			return logMsh.AddTrace()
		}

	default:
		if servstats.Stats.Suspended {
			var logMsh *errco.MshLog
			servstats.Stats.Suspended, logMsh = opsys.ProcTreeResume(uint32(ServTerm.cmd.Process.Pid))
			if logMsh != nil {
				return logMsh.AddTrace()
			}
		} else {
			errco.Logln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_SERVER_IS_WARM, "minecraft server already warm")
		}
	}

	// request a soft freeze
	FreezeMSRequest()

	return nil
}

// FreezeMS executes "stop" command on the minecraft server.
// When force == true, it does not perform player check and orders the server shutdown (according to ms status)
func FreezeMS(force bool) *errco.MshLog {
	errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "freezing minecraft server...")

	switch servstats.Stats.Status {

	case errco.SERVER_STATUS_OFFLINE:
		// ms is offline, log error if ms process is set to suspended

		if servstats.Stats.Suspended {
			errco.Logln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_SERVER_OFFLINE_SUSPENDED, "minecraft server is suspended and offline")
			servstats.Stats.Suspended = false // if ms is offline it's process can't be suspended
		}

		errco.Logln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_SERVER_IS_FROZEN, "minecraft server already frozen")

	case errco.SERVER_STATUS_STARTING:
		// ms is starting, resume the ms process, wait for status online and then freeze ms

		var logMsh *errco.MshLog

		// resume ms process if suspended
		if servstats.Stats.Suspended {
			servstats.Stats.Suspended, logMsh = opsys.ProcTreeResume(uint32(ServTerm.cmd.Process.Pid))
			if logMsh != nil {
				return logMsh.AddTrace()
			}
		}

		// wait for ms to go online
		for servstats.Stats.Status == errco.SERVER_STATUS_STARTING {
			time.Sleep(1 * time.Second)
		}

		// if ms is not online return
		if servstats.Stats.Status != errco.SERVER_STATUS_ONLINE {
			return errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_NOT_ONLINE, "server is not online")
		}

		// now it's the case of the ms status online
		fallthrough

	case errco.SERVER_STATUS_ONLINE:
		// is ms is online, resume the process and then stop it

		var logMsh *errco.MshLog

		// execute ms freeze
		if force {
			// if forceful freeze, execute ms stop
			logMsh = executeMSStop()
			if logMsh != nil {
				return logMsh.AddTrace()
			}
		} else if logMsh = readyToFreezeMS(); logMsh == nil {
			// if soft freeze and ms can be stopped, suspend/stop
			if config.ConfigRuntime.Msh.AllowSuspend {
				if !servstats.Stats.Suspended {
					servstats.Stats.Suspended, logMsh = opsys.ProcTreeSuspend(uint32(ServTerm.cmd.Process.Pid))
					if logMsh != nil {
						return logMsh.AddTrace()
					}
				}
			} else {
				logMsh = executeMSStop()
				if logMsh != nil {
					return logMsh.AddTrace()
				}
			}
		} else {
			errco.Log(logMsh.AddTrace())
		}

	case errco.SERVER_STATUS_STOPPING:
		// is ms is stopping, resume the process and let it stop

		// resume ms process if suspended
		if servstats.Stats.Suspended {
			var logMsh *errco.MshLog
			servstats.Stats.Suspended, logMsh = opsys.ProcTreeResume(uint32(ServTerm.cmd.Process.Pid))
			if logMsh != nil {
				return logMsh.AddTrace()
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
func executeMSStop() *errco.MshLog {
	var logMsh *errco.MshLog

	// resume ms process if suspended
	if servstats.Stats.Suspended {
		servstats.Stats.Suspended, logMsh = opsys.ProcTreeResume(uint32(ServTerm.cmd.Process.Pid))
		if logMsh != nil {
			return logMsh.AddTrace()
		}
	}

	// execute stop command
	_, logMsh = Execute(config.ConfigRuntime.Commands.StopServer, "executeMSStop")
	if logMsh != nil {
		return logMsh.AddTrace()
	}

	// if sigint is allowed, launch a function to check the shutdown of minecraft server
	if config.ConfigRuntime.Commands.StopServerAllowKill > 0 {
		go killMSifOnlineAfterTimeout()
	}

	return nil
}

// readyToFreezeMS returns nil if server is ready to be frozen (depending on player count)
func readyToFreezeMS() *errco.MshLog {
	// check that there is only one FreezeMSRequest running and players <= 0,
	// if so proceed with server shutdown
	atomic.AddInt32(&servstats.Stats.FreezeMSRequests, -1)

	// check how many players are on the server
	playerCount, method := countPlayerSafe()
	errco.Logln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "%d online players - method for player count: %s", playerCount, method)
	if playerCount > 0 {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_NOT_EMPTY, "server is not empty")
	}

	// check if enough time has passed since last player disconnected
	if atomic.LoadInt32(&servstats.Stats.FreezeMSRequests) > 0 {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_MUST_WAIT, fmt.Sprintf("not enough time has passed since last player disconnected (FreezeMSRequests: %d)", servstats.Stats.FreezeMSRequests))
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
			logMsh := FreezeMS(false)
			if logMsh != nil {
				errco.Log(logMsh.AddTrace())
			}
		},
	)
}

// killMSifOnlineAfterTimeout waits for the specified time and then if the server is still online, kills the server process
func killMSifOnlineAfterTimeout() {
	var logMsh *errco.MshLog

	countdown := config.ConfigRuntime.Commands.StopServerAllowKill

	// resume ms process if suspended
	if servstats.Stats.Suspended {
		servstats.Stats.Suspended, logMsh = opsys.ProcTreeResume(uint32(ServTerm.cmd.Process.Pid))
		if logMsh != nil {
			errco.Log(logMsh.AddTrace())
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
	errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "saving word before killing the minecraft server process")
	_, _ = Execute("save-all", "killMSifOnlineAfterTimeout")

	// give time to save word
	time.Sleep(10 * time.Second)

	// send kill signal to server
	errco.Logln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_SERVER_KILL, "minecraft server process won't stop normally: sending kill signal")
	err := ServTerm.cmd.Process.Kill()
	if err != nil {
		errco.Logln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_KILL, err.Error())
	}
}
