package servctrl

import (
	"log"
	"strings"
	"time"

	"msh/lib/asyncctrl"
	"msh/lib/confctrl"
	"msh/lib/debugctrl"
)

// StartMinecraftServer starts the minecraft server
func StartMinecraftServer() {
	var err error

	// start server terminal
	command := strings.ReplaceAll(confctrl.Config.Commands.StartServer, "serverFileName", confctrl.Config.Server.FileName)
	err = CmdStart(confctrl.Config.Server.Folder, command)
	if err != nil {
		log.Printf("StartMinecraftServer: error starting minecraft server: %v\n", err)
		return
	}
}

// StopMinecraftServer stops the minecraft server. When force == true, it bypasses checks for StopInstancesa/Players and orders the server shutdown
func StopMinecraftServer(force bool) {
	var err error

	// wait for the starting server to go online
	for ServStats.Status == "starting" {
		time.Sleep(1 * time.Second)
	}
	// if server is not online return
	if ServStats.Status != "online" {
		debugctrl.Log("servctrl: StopEmptyMinecraftServer: server is not online")
		return
	}

	// execute stop command
	if force {
		// if force == true, bypass checks for StopInstances/Players and proceed with server shutdown
		_, err = ServTerminal.Execute(confctrl.Config.Commands.StopServerForce)
	} else {
		// if force == false, check that there is only one "stop server command" instance running and players <= 0,
		// if so proceed with server shutdown
		asyncctrl.WithLock(func() { ServStats.StopInstances-- })
		if asyncctrl.WithLock(func() interface{} { return ServStats.StopInstances > 0 || ServStats.Players > 0 }).(bool) {
			return
		}

		_, err = ServTerminal.Execute(confctrl.Config.Commands.StopServer)
	}
	if err != nil {
		log.Printf("stopEmptyMinecraftServer: error stopping minecraft server: %s\n", err.Error())
		return
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
}

// RequestStopMinecraftServer increases stopInstances by one and starts the timer to execute stopEmptyMinecraftServer(false)
func RequestStopMinecraftServer() {
	asyncctrl.WithLock(func() { ServStats.StopInstances++ })
	time.AfterFunc(time.Duration(confctrl.Config.Msh.TimeBeforeStoppingEmptyServer)*time.Second, func() { StopMinecraftServer(false) })
}
