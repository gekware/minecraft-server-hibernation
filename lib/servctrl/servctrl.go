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
		return fmt.Errorf("StopEmptyMinecraftServer: server is not online")
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
			return fmt.Errorf("StopEmptyMinecraftServer: server is not empty")
		}

		// check if enough time has passed since last player disconnected
		if asyncctrl.WithLock(func() interface{} { return ServStats.StopInstances > 0 }).(bool) {
			return fmt.Errorf("StopEmptyMinecraftServer: not enough time has passed since last player disconnected (StopInstances: %d)", ServStats.StopInstances)
		}
	}

	// execute stop command
	_, errExec = ServTerminal.Execute(confctrl.Config.Commands.StopServer, "StopMinecraftServer")
	if errExec != nil {
		return fmt.Errorf("stopEmptyMinecraftServer: error executing minecraft server stop command: %v", errExec)
	}

	if !playersCheck {
		if ServStats.Status == "stopping" {
			// wait for the terminal to exit
			debugctrl.Logln("waiting for server terminal to exit")
			ServTerminal.Wg.Wait()
		} else {
			debugctrl.Logln("server was not stopped by stop command, world save might be compromised")
		}
	}

	return nil
}

// RequestStopMinecraftServer increases stopInstances by one and starts the timer to execute stopEmptyMinecraftServer(false)
func RequestStopMinecraftServer() {
	asyncctrl.WithLock(func() { ServStats.StopInstances++ })

	// [goroutine]
	time.AfterFunc(time.Duration(confctrl.Config.Msh.TimeBeforeStoppingEmptyServer)*time.Second, func() {
		err := StopMinecraftServer(true)
		if err != nil {
			// avoid printing "server is not online" error since it can be very frequent
			// when updating the logging system this could be managed by logging it only at certain log levels
			if err.Error() != "StopEmptyMinecraftServer: server is not online" {
				debugctrl.Logln("RequestStopMinecraftServer:", err)
			}
		}
	})
}
