package main

import (
	"fmt"
	"net"

	"msh/lib/config"
	"msh/lib/conn"
	"msh/lib/errco"
	"msh/lib/input"
	"msh/lib/progmgr"
	"msh/lib/servctrl"
	"msh/lib/utility"
)

// contains intro to script and program
var intro []string = []string{
	" _ __ ___  ___| |__  ",
	"| '_ ` _ \\/ __| '_ \\ ",
	"| | | | | \\__ \\ | | | " + progmgr.MshVersion,
	"|_| |_| |_|___/_| |_| " + progmgr.MshCommit,
	"Copyright (C) 2019-2022 gekigek99",
	"github: https://github.com/gekigek99",
	"remember to give a star to this repository!",
}

func main() {
	// print program intro
	// not using errco.NewLogln since log time is not needed
	fmt.Println(utility.Boxify(intro))

	// load configuration from msh config file
	logMsh := config.LoadConfig()
	if logMsh != nil {
		logMsh.Log(true)
		progmgr.AutoTerminate()
	}

	// launch msh manager
	go progmgr.MshMgr()
	// wait for the initial update check
	<-progmgr.ReqSent

	// if ms suspension is allowed, pre-warm the server
	if config.ConfigRuntime.Msh.SuspendAllow {
		errco.NewLogln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "minecraft server will now pre-warm (SuspendAllow is enabled)...")
		logMsh = servctrl.WarmMS()
		if logMsh != nil {
			logMsh.Log(true)
		}
	}

	// open a listener
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.ListenHost, config.ListenPort))
	if err != nil {
		errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CLIENT_LISTEN, err.Error())
		progmgr.AutoTerminate()
	}

	errco.NewLogln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "listening for new clients to connect on %s:%d ...", config.ListenHost, config.ListenPort)

	// launch GetInput()
	go input.GetInput()

	// infinite cycle to accept clients. when a clients connects it is passed to handleClientSocket()
	for {
		clientSocket, err := listener.Accept()
		if err != nil {
			errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CLIENT_ACCEPT, err.Error())
			continue
		}

		go conn.HandleClientSocket(clientSocket)
	}
}
