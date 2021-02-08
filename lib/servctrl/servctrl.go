package servctrl

import (
	"log"
	"strings"
	"time"

	"msh/lib/asyncctrl"
	"msh/lib/cmdctrl"
	"msh/lib/confctrl"
	"msh/lib/debugctrl"
)

var servTerm *cmdctrl.ServTerm

// StartMinecraftServer starts the minecraft server
func StartMinecraftServer() {
	var err error

	// start server terminal
	command := strings.ReplaceAll(confctrl.Config.Basic.StartMinecraftServer, "serverFileName", confctrl.Config.Basic.ServerFileName)
	servTerm, err = cmdctrl.Start(confctrl.Config.Basic.ServerDirPath, command)
	if err != nil {
		log.Printf("StartMinecraftServer: error starting minecraft server: %v\n", err)
		return
	}

	// initialization
	ServStats.Status = "starting"
	ServStats.TimeLeftUntilUp = confctrl.Config.Basic.MinecraftServerStartupTime
	ServStats.Players = 0

	log.Print("*** MINECRAFT SERVER IS STARTING!")

	// sets serverStatus == "online".
	// After {TimeBeforeStoppingEmptyServer} executes stopEmptyMinecraftServer(false)
	var setServerStatusOnline = func() {
		ServStats.Status = "online"
		log.Print("*** MINECRAFT SERVER IS UP!")

		asyncctrl.WithLock(func() { ServStats.StopInstances++ })
		time.AfterFunc(time.Duration(confctrl.Config.Basic.TimeBeforeStoppingEmptyServer)*time.Second, func() { StopEmptyMinecraftServer(false) })
	}
	// updates TimeLeftUntilUp each second. if TimeLeftUntilUp == 0 it executes setServerStatusOnline()
	var updateTimeleft func()
	updateTimeleft = func() {
		if ServStats.TimeLeftUntilUp > 0 {
			ServStats.TimeLeftUntilUp--
			time.AfterFunc(1*time.Second, func() { updateTimeleft() })
		} else if ServStats.TimeLeftUntilUp == 0 {
			setServerStatusOnline()
		}
	}

	time.AfterFunc(1*time.Second, func() { updateTimeleft() })
}

// StopEmptyMinecraftServer stops the minecraft server
func StopEmptyMinecraftServer(force bool) {
	if force && ServStats.Status != "offline" {
		// skip some checks to issue the stop server command forcefully
	} else {
		// check that there is only one "stop server command" instance running and players <= 0 and ServerStatus != "offline".
		// on the contrary the server won't be stopped
		asyncctrl.WithLock(func() { ServStats.StopInstances-- })

		if asyncctrl.WithLockBool(func() bool {
			return ServStats.StopInstances > 0 || ServStats.Players > 0 || ServStats.Status == "offline"
		}) {
			return
		}
	}

	// execute stop command
	if force && confctrl.Config.Basic.StopMinecraftServerForce != "" {
		err := servTerm.Execute(confctrl.Config.Basic.StopMinecraftServerForce)
		if err != nil {
			log.Printf("stopEmptyMinecraftServer: error stopping minecraft server: %v\n", err)
			return
		}
		// waits for the terminal to exit
		debugctrl.Logger("waiting for server terminal to exit")
		servTerm.Wg.Wait()
		debugctrl.Logger("server terminal exited")
	} else {
		err := servTerm.Execute(confctrl.Config.Basic.StopMinecraftServer)
		if err != nil {
			log.Printf("stopEmptyMinecraftServer: error stopping minecraft server: %v\n", err)
			return
		}
	}

	ServStats.Status = "offline"

	if force {
		log.Print("*** MINECRAFT SERVER IS FORCEFULLY SHUTTING DOWN!")
	} else {
		log.Print("*** MINECRAFT SERVER IS SHUTTING DOWN!")
	}
}
