package conn

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/servctrl"
	"msh/lib/servstats"
)

// HandleClientSocket handles a client that is connecting.
// Can handle a client that is requesting server info or trying to join.
// [goroutine]
func HandleClientSocket(clientSocket net.Conn) {
	// handling of ipv6 addresses
	li := strings.LastIndex(clientSocket.RemoteAddr().String(), ":")
	clientAddress := clientSocket.RemoteAddr().String()[:li]

	switch servstats.Stats.Status {
	case errco.SERVER_STATUS_OFFLINE:
		reqType, playerName, errMsh := getReqType(clientSocket)
		if errMsh != nil {
			errco.LogMshErr(errMsh.AddTrace("HandleClientSocket"))
			return
		}

		switch reqType {
		case errco.CLIENT_REQ_INFO:
			// client requests "server info"
			errco.Logln(errco.LVL_D, "%s requested server info from %s:%d to %s:%d", playerName, clientAddress, config.ListenPort, config.TargetHost, config.TargetPort)

			// answer to client with emulated server info
			clientSocket.Write(buildMessage(errco.MESSAGE_FORMAT_INFO, config.ConfigRuntime.Msh.InfoHibernation))

			// answer to client ping
			errMsh := getPing(clientSocket)
			if errMsh != nil {
				errco.LogMshErr(errMsh.AddTrace("HandleClientSocket"))
			}

		case errco.CLIENT_REQ_JOIN:
			// client requests "server join"

			// server is OFFLINE --> issue StartMS()
			errMsh := servctrl.StartMS()
			if errMsh != nil {
				// log to msh console and warn client with text in the loadscreen
				errco.LogMshErr(errMsh.AddTrace("HandleClientSocket"))
				clientSocket.Write(buildMessage(errco.MESSAGE_FORMAT_TXT, "An error occurred while starting the server: check the msh log"))
			} else {
				// log to msh console and answer client with text in the loadscreen
				errco.Logln(errco.LVL_D, "%s tried to join from %s:%d to %s:%d", playerName, clientAddress, config.ListenPort, config.TargetHost, config.TargetPort)
				clientSocket.Write(buildMessage(errco.MESSAGE_FORMAT_TXT, "Server start command issued. Please wait... "+servstats.Stats.LoadProgress))
			}
		}

		// close the client connection
		errco.Logln(errco.LVL_D, "closing connection for: %s", clientAddress)
		clientSocket.Close()

	case errco.SERVER_STATUS_STARTING:
		reqType, playerName, errMsh := getReqType(clientSocket)
		if errMsh != nil {
			errco.LogMshErr(errMsh.AddTrace("HandleClientSocket"))
			return
		}

		switch reqType {
		case errco.CLIENT_REQ_INFO:
			// client requests "INFO"

			errco.Logln(errco.LVL_D, "%s requested server info from %s:%d to %s:%d during server startup", playerName, clientAddress, config.ListenPort, config.TargetHost, config.TargetPort)

			// answer to client with emulated server info
			clientSocket.Write(buildMessage(errco.MESSAGE_FORMAT_INFO, config.ConfigRuntime.Msh.InfoStarting))

			// answer to client ping
			errMsh = getPing(clientSocket)
			if errMsh != nil {
				errco.LogMshErr(errMsh.AddTrace("HandleClientSocket"))
			}

		case errco.CLIENT_REQ_JOIN:
			// client requests "JOIN"

			// log to msh console and answer to client with text in the loadscreen
			errco.Logln(errco.LVL_D, "%s tried to join from %s:%d to %s:%d during server startup", playerName, clientAddress, config.ListenPort, config.TargetHost, config.TargetPort)
			clientSocket.Write(buildMessage(errco.MESSAGE_FORMAT_TXT, "Server is starting. Please wait... "+servstats.Stats.LoadProgress))
		}

		// close the client connection
		errco.Logln(errco.LVL_D, "closing connection for: %s", clientAddress)
		clientSocket.Close()

	case errco.SERVER_STATUS_ONLINE:
		// just open a connection with the server and connect it with the client
		serverSocket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", config.TargetHost, config.TargetPort))
		if err != nil {
			errco.LogMshErr(errco.NewErr(errco.ERROR_SERVER_DIAL, errco.LVL_D, "HandleClientSocket", err.Error()))
			// report dial error to client with text in the loadscreen
			clientSocket.Write(buildMessage(errco.MESSAGE_FORMAT_TXT, "can't connect to server... check if minecraft server is running and set the correct targetPort"))
			return
		}

		// stopC is used to close serv->client and client->serv at the same time
		stopC := make(chan bool, 1)

		// launch proxy client -> server
		go forward(clientSocket, serverSocket, false, stopC)

		// launch proxy server -> client
		go forward(serverSocket, clientSocket, true, stopC)
	}
}

// forward takes a source and a destination net.Conn and forwards them.
// (isServerToClient used to know the forward direction).
// [goroutine]
func forward(source, destination net.Conn, isServerToClient bool, stopC chan bool) {
	data := make([]byte, 1024)

	for {
		// if stopC receives true, close the source connection, otherwise continue
		select {
		case <-stopC:
			source.Close()
			return
		default:
		}

		// update read and write timeout
		source.SetReadDeadline(time.Now().Add(time.Duration(config.ConfigRuntime.Msh.TimeBeforeStoppingEmptyServer) * time.Second))
		destination.SetWriteDeadline(time.Now().Add(time.Duration(config.ConfigRuntime.Msh.TimeBeforeStoppingEmptyServer) * time.Second))

		// read data from source
		dataLen, err := source.Read(data)
		if err != nil {
			// case in which the connection is closed by the source or closed by target
			if err == io.EOF {
				errco.Logln(errco.LVL_D, "forward: closing %15s --> %15s because of: %s", strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0], err.Error())
			} else {
				errco.Logln(errco.LVL_D, "forward: %v\n%15s --> %15s", err, strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0])
			}

			// close the source connection
			stopC <- true
			source.Close()
			return
		}

		// write data to destination
		destination.Write(data[:dataLen])

		// calculate bytes/s to client/server
		if errco.DebugLvl >= errco.LVL_D {
			servstats.Stats.M.Lock()
			if isServerToClient {
				servstats.Stats.BytesToClients = servstats.Stats.BytesToClients + float64(dataLen)
				errco.Logln(errco.LVL_E, "server --> client:%v", data[:dataLen])
			} else {
				servstats.Stats.BytesToServer = servstats.Stats.BytesToServer + float64(dataLen)
				errco.Logln(errco.LVL_E, "client --> server:%v", data[:dataLen])
			}
			servstats.Stats.M.Unlock()
		}
	}
}
