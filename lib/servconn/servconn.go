package servconn

import (
	"log"
	"net"
	"strings"

	"msh/lib/config"
	"msh/lib/logger"
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
	case servctrl.STATUS_OFFLINE:
		clientPacket, err := readClientPacket(clientSocket)
		if err != nil {
			logger.Logln("HandleClientSocket:", err)
			return
		}

		switch getReqType(clientPacket) {
		case CLIENT_REQ_INFO:
			// client requests "server info"

			log.Printf("*** player unknown requested server info from %s:%s to %s:%s\n", clientAddress, config.ConfigRuntime.Msh.Port, config.TargetHost, config.TargetPort)

			// answer to client with emulated server info
			clientSocket.Write(buildMessage("info", config.ConfigRuntime.Msh.InfoHibernation))

			// answer to client ping
			err = answerPing(clientSocket)
			if err != nil {
				logger.Logln("HandleClientSocket:", err)
			}

		case CLIENT_REQ_JOIN:
			// client requests "server join"

			playerName, err := getPlayerName(clientSocket, clientPacket)
			if err != nil {
				logger.Logln("HandleClientSocket:", err)
				// this error is non-blocking, use an error string as playerName
				playerName = "playerNameError"
			}

			// server status == OFFLINE --> issue StartMS()
			err = servctrl.StartMS()
			if err != nil {
				// log to msh console and warn client with text in the loadscreen
				logger.Logln("HandleClientSocket:", err)
				clientSocket.Write(buildMessage("txt", "An error occurred while starting the server: check the msh log"))
			} else {
				// log to msh console and answer to client with text in the loadscreen
				log.Printf("*** %s tried to join from %s:%s to %s:%s\n", playerName, clientAddress, config.ConfigRuntime.Msh.Port, config.TargetHost, config.TargetPort)
				clientSocket.Write(buildMessage("txt", "Server start command issued. Please wait... "+servctrl.Stats.LoadProgress))
			}
		}

		// close the client connection
		logger.Logln("closing connection for:", clientAddress)
		clientSocket.Close()

	case servctrl.STATUS_STARTING:
		clientPacket, err := readClientPacket(clientSocket)
		if err != nil {
			logger.Logln("HandleClientSocket:", err)
			return
		}

		switch getReqType(clientPacket) {
		case CLIENT_REQ_INFO:
			// client requests "server info"

			log.Printf("*** player unknown requested server info from %s:%s to %s:%s during server startup\n", clientAddress, config.ConfigRuntime.Msh.Port, config.TargetHost, config.TargetPort)

			// answer to client with emulated server info
			clientSocket.Write(buildMessage("info", config.ConfigRuntime.Msh.InfoStarting))

			// answer to client ping
			err = answerPing(clientSocket)
			if err != nil {
				logger.Logln("HandleClientSocket:", err)
			}

		case CLIENT_REQ_JOIN:
			// client requests "server join"

			playerName, err := getPlayerName(clientSocket, clientPacket)
			if err != nil {
				logger.Logln("HandleClientSocket:", err)
				// this error is non-blocking, use an error string as playerName
				playerName = "playerNameError"
			}

			// log to msh console and answer to client with text in the loadscreen
			log.Printf("*** %s tried to join from %s:%s to %s:%s during server startup\n", playerName, clientAddress, config.ConfigRuntime.Msh.Port, config.TargetHost, config.TargetPort)
			clientSocket.Write(buildMessage("txt", "Server is starting. Please wait... "+servctrl.Stats.LoadProgress))
		}

		// close the client connection
		logger.Logln("closing connection for:", clientAddress)
		clientSocket.Close()

	case servctrl.STATUS_ONLINE:
		// just open a connection with the server and connect it with the client
		serverSocket, err := net.Dial("tcp", config.TargetHost+":"+config.TargetPort)
		if err != nil {
			logger.Logln("HandleClientSocket: error during serverSocket.Dial()")
			// report dial error to client with text in the loadscreen
			clientSocket.Write(buildMessage("txt", "can't connect to server... check if minecraft server is running and set the correct targetPort"))
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
