package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"msh/lib/confctrl"
	"msh/lib/connctrl"
	"msh/lib/debugctrl"
	"msh/lib/osctrl"
	"msh/lib/progctrl"
)

// script version
var version string = "v2.3.4"

// contains intro to script and program
var info []string = []string{
	"               _     ",
	" _ __ ___  ___| |__  ",
	"| '_ ` _ \\/ __| '_ \\ ",
	"| | | | | \\__ \\ | | |",
	"|_| |_| |_|___/_| |_|",
	"Copyright (C) 2019-2021 gekigek99",
	version,
	"visit my github page: https://github.com/gekigek99",
	"remember to give a star to this repository!",
}

//--------------------------PROGRAM---------------------------//

func main() {
	// print program intro
	fmt.Println(strings.Join(info, "\n"))

	// check is os is supported
	osctrl.CheckOsSupport()

	// load configuration from config file
	// load server-icon-frozen.png if present
	confctrl.LoadConfig()

	// check for updates
	if confctrl.Config.Msh.CheckForUpdates {
		progctrl.UpdateManager(version)
	}

	// listen for interrupt signals
	progctrl.InterruptListener()

	// launch printDataUsage()
	go debugctrl.PrintDataUsage()

	// open a listener on {confctrl.ListenHost}+":"+{Config.Msh.Port}
	listener, err := net.Listen("tcp", confctrl.ListenHost+":"+confctrl.Config.Msh.Port)
	if err != nil {
		log.Printf("main: Fatal error: %s", err.Error())
		time.Sleep(time.Duration(5) * time.Second)
		os.Exit(1)
	}

	defer func() {
		debugctrl.Logger("Closing connection for: listener")
		listener.Close()
	}()

	log.Println("*** listening for new clients to connect on " + confctrl.ListenHost + ":" + confctrl.Config.Msh.Port + " ...")

	// infinite cycle to accept clients. when a clients connects it is passed to handleClientSocket()
	for {
		clientSocket, err := listener.Accept()
		if err != nil {
			debugctrl.Logger("main:", err.Error())
			continue
		}
		connctrl.HandleClientSocket(clientSocket)
	}
}
