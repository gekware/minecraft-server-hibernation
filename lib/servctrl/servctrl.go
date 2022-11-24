package servctrl

import (
	"time"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/opsys"
	"msh/lib/servstats"
)

// WarmMS warms the minecraft server
// [non-blocking]
func WarmMS() *errco.MshLog {
	var logMsh *errco.MshLog

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

		logMsh = termStart(config.ConfigRuntime.Server.Folder, config.ConfigRuntime.Commands.StartServer)
		if logMsh != nil {
			servstats.Stats.SetMajorError(errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_MINECRAFT_SERVER, "error starting minecraft server (check logs)"))
			return logMsh.AddTrace()
		}

	default:
		if config.ConfigRuntime.Msh.AllowSuspend {
			servstats.Stats.Suspended, logMsh = opsys.ProcTreeResume(uint32(ServTerm.cmd.Process.Pid))
			if logMsh != nil {
				return logMsh.AddTrace()
			}
		}
	}

	// schedule soft freeze of ms
	FreezeMSSchedule()

	return nil
}

// FreezeMS executes "stop" command on the minecraft server.
// When force == true, it does not perform player check and orders the server shutdown (according to ms status)
func FreezeMS(force bool) *errco.MshLog {
	var logMsh *errco.MshLog

	if force {
		errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "executing minecraft server FORCE freeze...")
	} else {
		errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "executing minecraft server SOFT freeze...")
	}

	switch servstats.Stats.Status {

	case errco.SERVER_STATUS_STARTING:
		// ms is starting, resume the ms process, wait for status online and then freeze ms

		// resume ms process (un/suspended)
		if config.ConfigRuntime.Msh.AllowSuspend {
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

		// if forceful, execute ms stop then return
		if force {
			// if forceful freeze, execute ms stop
			logMsh = executeMSStop()
			if logMsh != nil {
				return logMsh.AddTrace()
			}
			return nil
		}

		// check how many players are on the server
		playerCount, method := countPlayerSafe()
		errco.Logln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "%d online players - method for player count: %s", playerCount, method)
		if playerCount > 0 {
			return errco.NewLog(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_SERVER_NOT_EMPTY, "server is not empty")
		}

		// suspend/stop ms
		if config.ConfigRuntime.Msh.AllowSuspend {
			servstats.Stats.Suspended, logMsh = opsys.ProcTreeSuspend(uint32(ServTerm.cmd.Process.Pid))
			if logMsh != nil {
				return logMsh.AddTrace()
			}
		} else {
			logMsh = executeMSStop()
			if logMsh != nil {
				return logMsh.AddTrace()
			}
		}

		return nil

	case errco.SERVER_STATUS_STOPPING:
		// is ms is stopping, resume the process and let it stop

		// resume ms process (un/suspended)
		if config.ConfigRuntime.Msh.AllowSuspend {
			servstats.Stats.Suspended, logMsh = opsys.ProcTreeResume(uint32(ServTerm.cmd.Process.Pid))
			if logMsh != nil {
				return logMsh.AddTrace()
			}
		}

		errco.Logln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_SERVER_STOPPING, "waiting for minecraft server to go offline...")

		// wait for ms to go offline
		for servstats.Stats.Status == errco.SERVER_STATUS_STOPPING {
			time.Sleep(1 * time.Second)
		}

		fallthrough

	case errco.SERVER_STATUS_OFFLINE:
		// ms is offline

		// log error if ms process is set to suspended
		if servstats.Stats.Suspended {
			errco.Logln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_SERVER_OFFLINE_SUSPENDED, "minecraft server is suspended and offline")
			servstats.Stats.Suspended = false // if ms is offline it's process can't be suspended
		}

		errco.Logln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_SERVER_OFFLINE, "minecraft server is offline")

		return nil

	default:
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_STATUS_UNKNOWN, "server status unknown")
	}
}

// FreezeMSSchedule stops freeze timer and schedules a soft freeze of ms
func FreezeMSSchedule() {
	errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "rescheduling ms soft freeze in %d seconds", config.ConfigRuntime.Msh.TimeBeforeStoppingEmptyServer)

	// stop freeze timer so that it can be reset
	if !servstats.Stats.FreezeTimer.Stop() {
		<-servstats.Stats.FreezeTimer.C
	}

	// schedule soft freeze of ms in TimeBeforeStoppingEmptyServer seconds
	// [goroutine]
	servstats.Stats.FreezeTimer = time.AfterFunc(
		time.Duration(config.ConfigRuntime.Msh.TimeBeforeStoppingEmptyServer)*time.Second,
		func() {
			// perform soft freeze of ms
			errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "performing scheduled ms soft freeze")
			logMsh := FreezeMS(false)
			if logMsh != nil {
				errco.Log(logMsh.AddTrace())
			}
		},
	)
}

// executeMSStop resumes ms process and executes a stop command in ms terminal.
// should be called only when ms status is online
func executeMSStop() *errco.MshLog {
	var logMsh *errco.MshLog

	// resume ms process (un/suspended)
	if config.ConfigRuntime.Msh.AllowSuspend {
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

	// launch a function to check the shutdown of minecraft server
	go killMSifOnlineAfterTimeout()

	return nil
}

// killMSifOnlineAfterTimeout waits for the specified time and then
// if the server is still online, kills the server process.
//
// if StopServerAllowKill is disabled this function does nothing.
func killMSifOnlineAfterTimeout() {
	var logMsh *errco.MshLog

	// if StopServerAllowKill is disabled in config, do nothing
	if config.ConfigRuntime.Commands.StopServerAllowKill <= 0 {
		return
	}

	countdown := config.ConfigRuntime.Commands.StopServerAllowKill

	// resume ms process (un/suspended)
	// to be sure that ms is running to stop itself
	if config.ConfigRuntime.Msh.AllowSuspend {
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
