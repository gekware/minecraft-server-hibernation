package main

import (
	"fmt"
	"net"
	"os"

	"msh/lib/config"
	"msh/lib/conn"
	"msh/lib/errco"
	"msh/lib/input"
	"msh/lib/progmgr"
	"msh/lib/utility"
)

// script version
var version string = "v2.4.4"

// contains intro to script and program
var intro []string = []string{
	" _ __ ___  ___| |__  ",
	"| '_ ` _ \\/ __| '_ \\ ",
	"| | | | | \\__ \\ | | |",
	"|_| |_| |_|___/_| |_| " + version,
	"Copyright (C) 2019-2022 gekigek99",
	"github: https://github.com/gekigek99",
	"remember to give a star to this repository!",
}

func main() {
	// print program intro
	// not using errco.Logln since log time is not needed
	fmt.Println(utility.Boxify(intro))

	// load configuration from config file
	// load server-icon-frozen.png if present
	// LoadConfig is the second function to be called
	errMsh := config.LoadConfig()
	if errMsh != nil {
		errco.LogMshErr(errMsh.AddTrace("main"))
		os.Exit(1)
	}

	// launch update manager to check for updates
	go progmgr.UpdateManager(version)
	// wait for the initial update check
	<-progmgr.CheckedUpdateC

	// listen for interrupt signals
	go progmgr.InterruptListener()

	// launch GetInput()
	go input.GetInput()

	// open a listener
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.ListenHost, config.ListenPort))
	if err != nil {
		errco.LogMshErr(errco.NewErr(errco.ERROR_CLIENT_LISTEN, errco.LVL_D, "main", err.Error()))
		os.Exit(1)
	}

	errco.Logln(errco.LVL_D, "listening for new clients to connect on %s:%d...", config.ListenHost, config.ListenPort)

	// infinite cycle to accept clients. when a clients connects it is passed to handleClientSocket()
	for {
		clientSocket, err := listener.Accept()
		if err != nil {
			errco.LogMshErr(errco.NewErr(errco.ERROR_CLIENT_ACCEPT, errco.LVL_D, "main", err.Error()))
			continue
		}

		go conn.HandleClientSocket(clientSocket)
	}
}
