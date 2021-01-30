package servctrl

import (
	"log"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"msh/lib/asyncctrl"
	"msh/lib/cmdctrl"
	"msh/lib/confctrl"
)

// ServerStatus represent the status of the minecraft server ("offline", "starting", "online")
var ServerStatus = "offline"

// Players keeps track of players connected to the server
var Players int = 0

// StopInstances keeps track of how many times stopEmptyMinecraftServer() has been called in the last {TimeBeforeStoppingEmptyServer} seconds
var StopInstances int = 0

// TimeLeftUntilUp keeps track of how many seconds are still needed to reach serverStatus == "online"
var TimeLeftUntilUp int

// StartMinecraftServer starts the minecraft server
func StartMinecraftServer() {
	ServerStatus = "starting"

	TimeLeftUntilUp = confctrl.Config.Basic.MinecraftServerStartupTime

	// block that execute the correct start command depending on the OS
	var err error
	if runtime.GOOS == "linux" {
		command := strings.ReplaceAll(confctrl.Config.Basic.StartMinecraftServerLin, "serverFileName", confctrl.Config.Basic.ServerFileName)
		cmd := exec.Command("/bin/bash", "-c", command)
		cmd.Dir = confctrl.Config.Basic.ServerDirPath
		err = cmd.Run()
	} else if runtime.GOOS == "darwin" {
		command := strings.ReplaceAll(confctrl.Config.Basic.StartMinecraftServerMac, "serverFileName", confctrl.Config.Basic.ServerFileName)
		cmd := exec.Command("/bin/bash", "-c", command)
		cmd.Dir = confctrl.Config.Basic.ServerDirPath
		err = cmd.Run()
	} else if runtime.GOOS == "windows" {
		command := strings.ReplaceAll(confctrl.Config.Basic.StartMinecraftServerWin, "serverFileName", confctrl.Config.Basic.ServerFileName)
		commandSplit := strings.Split(command, " ")
		cmd := exec.Command(commandSplit[0], commandSplit[1:]...)
		cmd.Dir = confctrl.Config.Basic.ServerDirPath
		cmdctrl.In, _ = cmd.StdinPipe()
		err = cmd.Start()
	}

	if err != nil {
		log.Printf("startMinecraftServer: error starting minecraft server: %v\n", err)
		return
	}

	log.Print("*** MINECRAFT SERVER IS STARTING!")

	// initialization of players
	Players = 0

	// sets serverStatus == "online".
	// After {TimeBeforeStoppingEmptyServer} executes stopEmptyMinecraftServer(false)
	var setServerStatusOnline = func() {
		ServerStatus = "online"
		log.Print("*** MINECRAFT SERVER IS UP!")

		asyncctrl.WithLock(func() { StopInstances++ })
		time.AfterFunc(time.Duration(confctrl.Config.Basic.TimeBeforeStoppingEmptyServer)*time.Second, func() { StopEmptyMinecraftServer(false) })
	}
	// updates TimeLeftUntilUp each second. if TimeLeftUntilUp == 0 it executes setServerStatusOnline()
	var updateTimeleft func()
	updateTimeleft = func() {
		if TimeLeftUntilUp > 0 {
			TimeLeftUntilUp--
			time.AfterFunc(1*time.Second, func() { updateTimeleft() })
		} else if TimeLeftUntilUp == 0 {
			setServerStatusOnline()
		}
	}

	time.AfterFunc(1*time.Second, func() { updateTimeleft() })
}

// StopEmptyMinecraftServer stops the minecraft server
func StopEmptyMinecraftServer(forceExec bool) {
	if forceExec && ServerStatus != "offline" {
		// skip some checks to issue the stop server command forcefully
	} else {
		// check that there is only one "stop server command" instance running and players <= 0 and ServerStatus != "offline".
		// on the contrary the server won't be stopped
		asyncctrl.WithLock(func() { StopInstances-- })

		if StopInstances > 0 || Players > 0 || ServerStatus == "offline" {
			return
		}
	}

	ServerStatus = "offline"

	// block that execute the correct stop command depending on the OS
	var err error
	if runtime.GOOS == "linux" {
		if forceExec {
			err = exec.Command("/bin/bash", "-c", confctrl.Config.Basic.ForceStopMinecraftServerLin).Run()
		} else {
			err = exec.Command("/bin/bash", "-c", confctrl.Config.Basic.StopMinecraftServerLin).Run()
		}
	} else if runtime.GOOS == "darwin" {
		if forceExec {
			err = exec.Command("/bin/bash", "-c", confctrl.Config.Basic.ForceStopMinecraftServerMac).Run()
		} else {
			err = exec.Command("/bin/bash", "-c", confctrl.Config.Basic.StopMinecraftServerMac).Run()
		}
	} else if runtime.GOOS == "windows" {
		if forceExec {
			_, err = cmdctrl.In.Write([]byte(confctrl.Config.Basic.ForceStopMinecraftServerWin))
		} else {
			_, err = cmdctrl.In.Write([]byte(confctrl.Config.Basic.StopMinecraftServerWin))
		}
		cmdctrl.In.Close()
	}

	if err != nil {
		log.Printf("stopEmptyMinecraftServer: error stopping minecraft server: %v\n", err)
	}

	if forceExec {
		log.Print("*** MINECRAFT SERVER IS FORCEFULLY SHUTTING DOWN!")
	} else {
		log.Print("*** MINECRAFT SERVER IS SHUTTING DOWN!")
	}
}
