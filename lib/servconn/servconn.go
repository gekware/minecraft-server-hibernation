package servconn

import (
	"fmt"
	"net"
	"strings"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/servctrl"
)

// HandleClientSocket handles a client that is connecting.
// Can handle a client that is requesting server info or trying to join.
// [goroutine]
func HandleClientSocket(clientSocket net.Conn) {
	// handling of ipv6 addresses
	li := strings.LastIndex(clientSocket.RemoteAddr().String(), ":")
	clientAddress := clientSocket.RemoteAddr().String()[:li]

	switch servctrl.Stats.Status {
	case errco.SERVER_STATUS_OFFLINE:
		reqType, playerName, errMsh := getReqType(clientSocket)
		if errMsh != nil {
			errco.LogMshErr(errMsh.AddTrace("HandleClientSocket"))
			return
		}

		switch reqType {
		case errco.CLIENT_REQ_INFO:
			// client requests "server info"
			errco.Logln(errco.LVL_D, fmt.Sprintf("%s requested server info from %s:%s to %s:%s\n", playerName, clientAddress, config.ListenPort, config.TargetHost, config.TargetPort))

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
				errco.Logln(errco.LVL_D, fmt.Sprintf("%s tried to join from %s:%s to %s:%s\n", playerName, clientAddress, config.ListenPort, config.TargetHost, config.TargetPort))
				clientSocket.Write(buildMessage(errco.MESSAGE_FORMAT_TXT, "Server start command issued. Please wait... "+servctrl.Stats.LoadProgress))
			}
		}

		// close the client connection
		errco.Logln(errco.LVL_D, "closing connection for:", clientAddress)
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

			errco.Logln(errco.LVL_D, fmt.Sprintf("%s requested server info from %s:%s to %s:%s during server startup\n", playerName, clientAddress, config.ListenPort, config.TargetHost, config.TargetPort))

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
			errco.Logln(errco.LVL_D, fmt.Sprintf("%s tried to join from %s:%s to %s:%s during server startup\n", playerName, clientAddress, config.ListenPort, config.TargetHost, config.TargetPort))
			clientSocket.Write(buildMessage(errco.MESSAGE_FORMAT_TXT, "Server is starting. Please wait... "+servctrl.Stats.LoadProgress))
		}

		// close the client connection
		errco.Logln(errco.LVL_D, "closing connection for:", clientAddress)
		clientSocket.Close()

	case errco.SERVER_STATUS_ONLINE:
		// just open a connection with the server and connect it with the client
		serverSocket, err := net.Dial("tcp", config.TargetHost+":"+config.TargetPort)
		if err != nil {
			errco.LogMshErr(errco.NewErr(errco.SERVER_DIAL_ERROR, errco.LVL_D, "HandleClientSocket", "error while dialing local minecraft server"))
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
