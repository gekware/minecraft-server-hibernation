package servctrl

import (
	"fmt"
	"strings"
	"time"

	"msh/lib/asyncctrl"
	"msh/lib/confctrl"
	"msh/lib/debugctrl"
)

// StartMinecraftServer starts the minecraft server
func StartMinecraftServer() error {
	var err error

	// start server terminal
	command := strings.ReplaceAll(confctrl.Config.Commands.StartServer, "serverFileName", confctrl.Config.Server.FileName)
	err = CmdStart(confctrl.Config.Server.Folder, command)
	if err != nil {
		return fmt.Errorf("StartMinecraftServer: error starting minecraft server: %v", err)
	}

	return nil
}

// StopMinecraftServer stops the minecraft server.
// When playersCheck == true, it checks for StopInstancesa/Players and orders the server shutdown
func StopMinecraftServer(playersCheck bool) error {
	// error that returns from Execute() when executing the stop command
	var errExec error

	// wait for the starting server to go online
	for ServStats.Status == "starting" {
		time.Sleep(1 * time.Second)
	}
	// if server is not online return
	if ServStats.Status != "online" {
		return fmt.Errorf("StopMinecraftServer: server is not online")
	}

	// player/stopInstances check
	if playersCheck {
		// check that there is only one "stop server command" instance running and players <= 0,
		// if so proceed with server shutdown
		asyncctrl.WithLock(func() { ServStats.StopInstances-- })

		// check how many players are on the server
		playerCount, isFromServer := CountPlayerSafe()
		debugctrl.Logln(playerCount, "online players - number got from server:", isFromServer)
		if playerCount > 0 {
			return fmt.Errorf("StopMinecraftServer: server is not empty")
		}

		// check if enough time has passed since last player disconnected
		if asyncctrl.WithLock(func() interface{} { return ServStats.StopInstances > 0 }).(bool) {
			return fmt.Errorf("StopMinecraftServer: not enough time has passed since last player disconnected (StopInstances: %d)", ServStats.StopInstances)
		}
	}

	// execute stop command
	_, errExec = ServTerminal.Execute(confctrl.Config.Commands.StopServer, "StopMinecraftServer")
	if errExec != nil {
		return fmt.Errorf("StopMinecraftServer: error executing minecraft server stop command: %v", errExec)
	}

	// if sigint is allowed, launch a function to check the shutdown of minecraft server
	if confctrl.Config.Commands.StopServerAllowKill > 0 {
		go sigintMinecraftServerIfOnlineAfterTimeout()
	}

	return nil
}

// RequestStopMinecraftServer increases stopInstances by one and starts the timer to execute StopMinecraftServer(false)
// [goroutine]
func RequestStopMinecraftServer() {
	asyncctrl.WithLock(func() { ServStats.StopInstances++ })

	// [goroutine]
	time.AfterFunc(time.Duration(confctrl.Config.Msh.TimeBeforeStoppingEmptyServer)*time.Second, func() {
		err := StopMinecraftServer(true)
		if err != nil {
			// avoid printing "server is not online" error since it can be very frequent
			// when updating the logging system this could be managed by logging it only at certain log levels
			if err.Error() != "StopMinecraftServer: server is not online" {
				debugctrl.Logln("RequestStopMinecraftServer:", err)
			}
		}
	})
}

// sigintMinecraftServerIfOnlineAfterTimeout waits for the specified time and then if the server is still online sends SIGINT to the process
func sigintMinecraftServerIfOnlineAfterTimeout() {
	countdown := confctrl.Config.Commands.StopServerAllowKill

	for countdown > 0 {
		// if server goes offline it's the correct behaviour -> return
		if ServStats.Status == "offline" {
			return
		}

		countdown--
		time.Sleep(time.Second)
	}

	// save world before killing the server, do not check for errors
	debugctrl.Logln("saving word before killing the minecraft server process")
	_, _ = ServTerminal.Execute("/save-all", "sigintMinecraftServerIfOnlineAfterTimeout")

	// give time to save word
	time.Sleep(time.Duration(10) * time.Second)

	// send kill signal to server
	debugctrl.Logln("send kill signal to minecraft server process since it won't stop normally")
	err := ServTerminal.cmd.Process.Kill()
	if err != nil {
		debugctrl.Logln("sigintMinecraftServerIfOnlineAfterTimeout: %v", err)
	}
}
