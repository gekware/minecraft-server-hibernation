package servctrl

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"msh/lib/asyncctrl"
	"msh/lib/confctrl"
	"msh/lib/debugctrl"
	"msh/lib/utility"
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
// When force == true, it bypasses checks for StopInstancesa/Players and orders the server shutdown
func StopMinecraftServer(force bool) error {
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

	// execute stop command
	if force {
		// if force == true, bypass checks for StopInstances/Players and proceed with server shutdown
		_, errExec = ServTerminal.Execute(confctrl.Config.Commands.StopServerForce)
	} else {
		// if force == false, check that there is only one "stop server command" instance running and players <= 0,
		// if so proceed with server shutdown
		asyncctrl.WithLock(func() { ServStats.StopInstances-- })

		// check if there are players online
		if asyncctrl.WithLock(func() interface{} { return ServStats.Players > 0 }).(bool) {
			return fmt.Errorf("StopEmptyMinecraftServer: server is not empty (player count > 0)")
		}

		// check if there are players online using the /list command
		// (for safety, in case the first player check does not work)
		outStr, err := ServTerminal.Execute("/list")
		if err != nil {
			return fmt.Errorf("stopEmptyMinecraftServer: error executing minecraft server /list command: %v", err)
		}
		playersStr, err := utility.StrBetween(outStr, "There are ", " of a max")
		if err != nil {
			return fmt.Errorf("stopEmptyMinecraftServer: error finding player number from /list command: %v", err)
		}
		players, err := strconv.Atoi(playersStr)
		if err != nil {
			return fmt.Errorf("stopEmptyMinecraftServer: error converting player number from /list command: %v", err)
		}
		if players > 0 {
			return fmt.Errorf("StopEmptyMinecraftServer: server is not empty (with /list command)")
		}

		// check if enough time has passed since last player disconnected
		if asyncctrl.WithLock(func() interface{} { return ServStats.StopInstances > 0 }).(bool) {
			return fmt.Errorf("StopEmptyMinecraftServer: not enough time has passed since last player disconnected (StopInstances: %d)", ServStats.StopInstances)
		}

		_, errExec = ServTerminal.Execute(confctrl.Config.Commands.StopServer)
	}
	if errExec != nil {
		return fmt.Errorf("stopEmptyMinecraftServer: error executing minecraft server stop command: %v", errExec)
	}

	if force {
		if ServStats.Status == "stopping" {
			// wait for the terminal to exit
			debugctrl.Log("waiting for server terminal to exit")
			ServTerminal.Wg.Wait()
		} else {
			debugctrl.Log("server was not stopped by StopMinecraftServerForce command, world save might be compromised")
		}
	}

	return nil
}

// RequestStopMinecraftServer increases stopInstances by one and starts the timer to execute stopEmptyMinecraftServer(false)
func RequestStopMinecraftServer() {
	asyncctrl.WithLock(func() { ServStats.StopInstances++ })

	// [goroutine]
	time.AfterFunc(time.Duration(confctrl.Config.Msh.TimeBeforeStoppingEmptyServer)*time.Second, func() {
		err := StopMinecraftServer(false)
		if err != nil {
			// avoid printing "server is not online" error since it can be very frequent
			// when updating the logging system this could be managed by logging it only at certain log levels
			if err.Error() != "StopEmptyMinecraftServer: server is not online" {
				debugctrl.Log("RequestStopMinecraftServer:", err)
			}
		}
	})
}
