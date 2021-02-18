package servctrl

import (
	"log"
	"strings"
	"time"

	"msh/lib/asyncctrl"
	"msh/lib/confctrl"
	"msh/lib/debugctrl"
)

var servTerm *ServTerm

// StartMinecraftServer starts the minecraft server
func StartMinecraftServer() {
	var err error

	// start server terminal
	command := strings.ReplaceAll(confctrl.Config.Basic.StartMinecraftServer, "serverFileName", confctrl.Config.Basic.ServerFileName)
	servTerm, err = CmdStart(confctrl.Config.Basic.ServerDirPath, command)
	if err != nil {
		log.Printf("StartMinecraftServer: error starting minecraft server: %v\n", err)
		return
	}
}

// StopEmptyMinecraftServer stops the minecraft server
func StopEmptyMinecraftServer(force bool) {
	// wait for the starting server to become online
	for ServStats.Status != "starting" {
		time.Sleep(1 * time.Second)
	}
	// if server is not online return
	if ServStats.Status != "online" {
		debugctrl.Logger("servctrl: StopEmptyMinecraftServer: server is not online")
		return
	}

	if force {
		// skip checks to issue the stop server command forcefully
	} else {
		// check that there is only one "stop server command" instance running and players <= 0.
		// on the contrary the server won't be stopped
		asyncctrl.WithLock(func() { ServStats.StopInstances-- })

		if asyncctrl.WithLock(func() interface{} { return ServStats.StopInstances > 0 || ServStats.Players > 0 }).(bool) {
			return
		}
	}

	// execute stop command
	var stopCom string
	stopCom = confctrl.Config.Basic.StopMinecraftServer
	if force {
		if confctrl.Config.Basic.StopMinecraftServerForce != "" {
			stopCom = confctrl.Config.Basic.StopMinecraftServerForce
		}
	}

	_, err := servTerm.Execute(stopCom)
	if err != nil {
		log.Printf("stopEmptyMinecraftServer: error stopping minecraft server: %s\n", err.Error())
		return
	}

	if force {
		if ServStats.Status == "stopping" {
			// wait for the terminal to exit
			debugctrl.Logger("waiting for server terminal to exit")
			servTerm.Wg.Wait()
		} else {
			log.Println()
			debugctrl.Logger("server does not seem to be stopping, is the StopMinecraftServerForce command correct?")
		}
	}
}
