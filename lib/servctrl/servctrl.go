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

	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "issued minecraft server warm...")

	// don't try to warm ms if it has encountered major errors
	if servstats.Stats.MajorError != nil {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_1, errco.ERROR_MINECRAFT_SERVER, "minecraft server has encountered major problems")
	}

	switch servstats.Stats.Status {

	case errco.SERVER_STATUS_OFFLINE:
		// ms is offline, log error if ms process is set to suspended

		if servstats.Stats.Suspended {
			errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_OFFLINE_SUSPENDED, "minecraft server is suspended and offline")
			servstats.Stats.Suspended = false // if ms is offline it's process can't be suspended
		}

		logMsh = termStart()
		if logMsh != nil {
			servstats.Stats.SetMajorError(errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_MINECRAFT_SERVER, "error starting minecraft server (check logs)"))
			return logMsh.AddTrace()
		}

	default:
		if config.ConfigRuntime.Msh.SuspendAllow {
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
//
// If force freeze is issued while ms is starting, this func waits for ms to reach online state and then force freeze it.
func FreezeMS(force bool) *errco.MshLog {
	var logMsh *errco.MshLog

	if force {
		errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "executing ms force freeze...")
	} else {
		errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "executing ms soft freeze...")
	}

	switch servstats.Stats.Status {

	case errco.SERVER_STATUS_STARTING:
		// ms is starting, resume the ms process and freeze ms

		// resume ms process (un/suspended)
		// to be sure that ms process is running to allow ms start
		if config.ConfigRuntime.Msh.SuspendAllow {
			servstats.Stats.Suspended, logMsh = opsys.ProcTreeResume(uint32(ServTerm.cmd.Process.Pid))
			if logMsh != nil {
				return logMsh.AddTrace()
			}
		}

		if force {
			// wait ms to go online
			errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "waiting for minecraft server to go online... (msh will stop it after)")
			for servstats.Stats.Status == errco.SERVER_STATUS_STARTING {
				time.Sleep(1 * time.Second)
			}

			// if ms not online return error
			if servstats.Stats.Status != errco.SERVER_STATUS_ONLINE {
				return errco.NewLog(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_SERVER_NOT_ONLINE, "minecraft server did not reach online status after starting")
			}

			// ms is now online
			errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "minecraft server is now online! (msh proceeds to stop it)")

			// proceed to fallthrough

		} else {
			// schedule soft freeze of ms
			// (give ms more time to start)
			FreezeMSSchedule()
			return nil
		}

		// ms is now online
		fallthrough

	case errco.SERVER_STATUS_ONLINE:
		// ms is online, resume the process and then stop ms

		// if force freeze, resume and stop ms
		if force {
			logMsh = resumeStopMS()
			if logMsh != nil {
				return logMsh.AddTrace()
			}
			return nil
		}

		// check how many players are on the server
		if countPlayerSafe() > 0 {
			return errco.NewLog(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_SERVER_NOT_EMPTY, "server is not empty")
		}

		// suspend/stop ms
		if config.ConfigRuntime.Msh.SuspendAllow {
			servstats.Stats.Suspended, logMsh = opsys.ProcTreeSuspend(uint32(ServTerm.cmd.Process.Pid))
			if logMsh != nil {
				return logMsh.AddTrace()
			}
		} else {
			// resume and stop ms
			logMsh = resumeStopMS()
			if logMsh != nil {
				return logMsh.AddTrace()
			}
		}

		return nil

	case errco.SERVER_STATUS_STOPPING:
		// is ms is stopping, resume the process and let it stop

		// resume ms process (un/suspended)
		if config.ConfigRuntime.Msh.SuspendAllow {
			servstats.Stats.Suspended, logMsh = opsys.ProcTreeResume(uint32(ServTerm.cmd.Process.Pid))
			if logMsh != nil {
				return logMsh.AddTrace()
			}
		}

		errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_SERVER_STOPPING, "waiting for minecraft server to go offline...")

		// wait for ms to go offline
		for servstats.Stats.Status == errco.SERVER_STATUS_STOPPING {
			time.Sleep(1 * time.Second)
		}

		fallthrough

	case errco.SERVER_STATUS_OFFLINE:
		// ms is offline

		// log error if ms process is set to suspended
		if servstats.Stats.Suspended {
			errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_SERVER_OFFLINE_SUSPENDED, "minecraft server is suspended and offline")
			servstats.Stats.Suspended = false // if ms is offline it's process can't be suspended
		}

		errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_SERVER_OFFLINE, "minecraft server is offline")

		return nil

	default:
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_STATUS_UNKNOWN, "server status unknown")
	}
}

// FreezeMSSchedule stops freeze timer and schedules a soft freeze of ms
func FreezeMSSchedule() {
	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "scheduling ms soft freeze in %d seconds", config.ConfigRuntime.Msh.TimeBeforeStoppingEmptyServer)

	// stop freeze timer so that it can be reset
	// don't use drain channel procedure described in Stop() as it might happen
	// that at this point a signal has already been received from t.C
	// (calling a <-channel might be blocking)
	_ = servstats.Stats.FreezeTimer.Stop()

	// schedule soft freeze of ms in TimeBeforeStoppingEmptyServer seconds
	// [goroutine]
	servstats.Stats.FreezeTimer = time.AfterFunc(
		time.Duration(config.ConfigRuntime.Msh.TimeBeforeStoppingEmptyServer)*time.Second,
		func() {
			// perform soft freeze of ms
			errco.NewLogln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "performing scheduled ms soft freeze")
			logMsh := FreezeMS(false)
			if logMsh != nil {
				logMsh.Log(true)
			}
		},
	)
}

// resumeStopMS resumes ms process and executes a stop command in ms terminal.
//
// Should be called only when servstats.Stats.Status == ONLINE
func resumeStopMS() *errco.MshLog {
	var logMsh *errco.MshLog

	// resume ms process (un/suspended)
	if config.ConfigRuntime.Msh.SuspendAllow {
		servstats.Stats.Suspended, logMsh = opsys.ProcTreeResume(uint32(ServTerm.cmd.Process.Pid))
		if logMsh != nil {
			return logMsh.AddTrace()
		}
	}

	// execute stop command
	_, logMsh = Execute(config.ConfigRuntime.Commands.StopServer)
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
	if config.ConfigRuntime.Msh.SuspendAllow {
		servstats.Stats.Suspended, logMsh = opsys.ProcTreeResume(uint32(ServTerm.cmd.Process.Pid))
		if logMsh != nil {
			logMsh.Log(true)
		}
	}

	for countdown > 0 {
		// if server goes offline it's the correct behaviour -> return
		if servstats.Stats.Status == errco.SERVER_STATUS_OFFLINE {
			return
		}

		countdown--
		time.Sleep(1 * time.Second)
	}

	// save world before killing the server, do not check for errors
	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "saving word before killing the minecraft server process")
	_, _ = Execute("save-all")

	// give time to save word
	time.Sleep(10 * time.Second)

	// send kill signal to server
	errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_SERVER_KILL, "minecraft server process won't stop normally: sending kill signal")
	LogMsh := opsys.ProcTreeKill(uint32(ServTerm.cmd.Process.Pid))
	if LogMsh != nil {
		LogMsh.Log(true)
	}
}
