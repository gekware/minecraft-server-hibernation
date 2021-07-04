package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"msh/lib/confctrl"
	"msh/lib/connctrl"
	"msh/lib/debugctrl"
	"msh/lib/inputctrl"
	"msh/lib/osctrl"
	"msh/lib/progctrl"
	"msh/lib/utility"
)

// script version
var version string = "v2.4.2"

// contains intro to script and program
var intro []string = []string{
	" _ __ ___  ___| |__  ",
	"| '_ ` _ \\/ __| '_ \\ ",
	"| | | | | \\__ \\ | | |",
	"|_| |_| |_|___/_| |_| " + version,
	"Copyright (C) 2019-2021 gekigek99",
	"github: https://github.com/gekigek99",
	"remember to give a star to this repository!",
}

func main() {
	// print program intro
	fmt.Println(utility.Boxify(intro))

	// check is os is supported.
	// OsSupported is the first function to be called
	err := osctrl.OsSupported()
	if err != nil {
		log.Println("main:", err.Error())
		os.Exit(1)
	}

	// load configuration from config file
	// load server-icon-frozen.png if present
	// LoadConfig is the second function to be called
	err = confctrl.LoadConfig()
	if err != nil {
		log.Println("main:", err.Error())
		os.Exit(1)
	}

	// launch update manager to check for updates
	go progctrl.UpdateManager(version)
	// wait for the initial update check
	<-progctrl.CheckedUpdateC

	// listen for interrupt signals
	go progctrl.InterruptListener()

	// launch GetInput()
	go inputctrl.GetInput()

	// open a listener on {confctrl.ListenHost}+":"+{Config.Msh.Port}
	listener, err := net.Listen("tcp", confctrl.ListenHost+":"+confctrl.ConfigRuntime.Msh.Port)
	if err != nil {
		log.Println("main:", err.Error())
		os.Exit(1)
	}

	defer func() {
		listener.Close()
	}()

	log.Println("*** listening for new clients to connect on " + confctrl.ListenHost + ":" + confctrl.ConfigRuntime.Msh.Port + " ...")

	// infinite cycle to accept clients. when a clients connects it is passed to handleClientSocket()
	for {
		clientSocket, err := listener.Accept()
		if err != nil {
			debugctrl.Logln("main:", err.Error())
			continue
		}

		go connctrl.HandleClientSocket(clientSocket)
	}
}
