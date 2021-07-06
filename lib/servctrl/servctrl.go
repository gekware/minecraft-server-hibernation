package servctrl

import (
	"fmt"
	"sync/atomic"
	"time"

	"msh/lib/confctrl"
	"msh/lib/debugctrl"
)

// StartMS starts the minecraft server
func StartMS() error {
	// start server terminal
	err := CmdStart(confctrl.ConfigRuntime.Server.Folder, confctrl.ConfigRuntime.Commands.StartServer)
	if err != nil {
		return fmt.Errorf("StartMS: error starting minecraft server: %v", err)
	}

	return nil
}

// StopMS stops the minecraft server.
// When playersCheck == true, it checks for StopInstancesa/Players and orders the server shutdown
func StopMS(playersCheck bool) error {
	// error that returns from Execute() when executing the stop command
	var errExec error

	// wait for the starting server to go online
	for Stats.Status == "starting" {
		time.Sleep(1 * time.Second)
	}
	// if server is not online return
	if Stats.Status != "online" {
		return fmt.Errorf("StopMS: server is not online")
	}

	// player/stopInstances check
	if playersCheck {
		// check that there is only one "stop server command" instance running and players <= 0,
		// if so proceed with server shutdown
		atomic.AddInt32(&Stats.StopMSRequests, -1)

		// check how many players are on the server
		playerCount, isFromServer := CountPlayerSafe()
		debugctrl.Logln(playerCount, "online players - number got from server:", isFromServer)
		if playerCount > 0 {
			return fmt.Errorf("StopMS: server is not empty")
		}

		// check if enough time has passed since last player disconnected

		if atomic.LoadInt32(&Stats.StopMSRequests) > 0 {
			return fmt.Errorf("StopMS: not enough time has passed since last player disconnected (StopInstances: %d)", Stats.StopMSRequests)
		}
	}

	// execute stop command
	_, errExec = Execute(confctrl.ConfigRuntime.Commands.StopServer, "StopMS")
	if errExec != nil {
		return fmt.Errorf("StopMS: error executing minecraft server stop command: %v", errExec)
	}

	// if sigint is allowed, launch a function to check the shutdown of minecraft server
	if confctrl.ConfigRuntime.Commands.StopServerAllowKill > 0 {
		go killMSifOnlineAfterTimeout()
	}

	return nil
}

// StopMSRequest increases stopInstances by one and starts the timer to execute StopMS(false)
// [goroutine]
func StopMSRequest() { // !!! + cambiare tutti i minecraft server a MS
	atomic.AddInt32(&Stats.StopMSRequests, 1)

	// [goroutine]
	time.AfterFunc(time.Duration(confctrl.ConfigRuntime.Msh.TimeBeforeStoppingEmptyServer)*time.Second, func() {
		err := StopMS(true)
		if err != nil {
			// avoid printing "server is not online" error since it can be very frequent
			// when updating the logging system this could be managed by logging it only at certain log levels
			if err.Error() != "StopMS: server is not online" {
				debugctrl.Logln("StopMSRequest:", err)
			}
		}
	})
}

// killMSifOnlineAfterTimeout waits for the specified time and then if the server is still online, kills the server process
func killMSifOnlineAfterTimeout() {
	countdown := confctrl.ConfigRuntime.Commands.StopServerAllowKill

	for countdown > 0 {
		// if server goes offline it's the correct behaviour -> return
		if Stats.Status == "offline" {
			return
		}

		countdown--
		time.Sleep(time.Second)
	}

	// save world before killing the server, do not check for errors
	debugctrl.Logln("saving word before killing the minecraft server process")
	_, _ = Execute("/save-all", "killMSifOnlineAfterTimeout")

	// give time to save word
	time.Sleep(time.Duration(10) * time.Second)

	// send kill signal to server
	debugctrl.Logln("send kill signal to minecraft server process since it won't stop normally")
	err := ServTerm.cmd.Process.Kill()
	if err != nil {
		debugctrl.Logln("killMSifOnlineAfterTimeout: %v", err)
	}
}
