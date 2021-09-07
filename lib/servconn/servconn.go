package servconn

import (
	"bytes"
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
	case "offline":
		buffer := make([]byte, 1024)

		// read first packet
		dataLen, err := clientSocket.Read(buffer)
		if err != nil {
			logger.Logln("HandleClientSocket: error during clientSocket.Read()")
			return
		}

		bufferData := buffer[:dataLen]

		// client is requesting server info and ping
		// client first packet:	[... x x x 1 1 0]	or	[... x x x 1])

		if bufferData[dataLen-1] == 0 || bufferData[dataLen-1] == 1 {
			log.Printf("*** player unknown requested server info from %s:%s to %s:%s\n", clientAddress, config.ConfigRuntime.Msh.Port, config.TargetHost, config.TargetPort)

			// answer to client with emulated server info
			clientSocket.Write(buildMessage("info", config.ConfigRuntime.Msh.InfoHibernation))

			// answer to client with ping
			err = answerPingReq(clientSocket)
			if err != nil {
				logger.Logln("HandleClientSocket:", err)
			}
		}

		// client is trying to join the server
		// client first packet:	[ ... x x x (listenPortBytes) 2] or [ ... x x x (listenPortBytes) 2 (player name)]

		if bytes.Contains(bufferData, append(buildListenPortBytes(), byte(2))) {
			playerName := getPlayerName(clientSocket, bufferData)

			// server status == "offline" --> issue StartMS()
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

	case "starting":
		buffer := make([]byte, 1024)

		// read first packet
		dataLen, err := clientSocket.Read(buffer)
		if err != nil {
			logger.Logln("HandleClientSocket: error during clientSocket.Read()")
			return
		}

		bufferData := buffer[:dataLen]

		// client is requesting server info and ping
		// client first packet:	[... x x x 1 1 0]	or	[... x x x 1])

		if buffer[dataLen-1] == 0 || buffer[dataLen-1] == 1 {
			log.Printf("*** player unknown requested server info from %s:%s to %s:%s during server startup\n", clientAddress, config.ConfigRuntime.Msh.Port, config.TargetHost, config.TargetPort)

			// answer to client with emulated server info
			clientSocket.Write(buildMessage("info", config.ConfigRuntime.Msh.InfoStarting))

			// answer to client with ping
			err = answerPingReq(clientSocket)
			if err != nil {
				logger.Logln("HandleClientSocket:", err)
			}
		}

		// client is trying to join the server
		// client first packet:	[ ... x x x (listenPortBytes) 2] or [ ... x x x (listenPortBytes) 2 (player name)]

		if bytes.Contains(bufferData, append(buildListenPortBytes(), byte(2))) {
			playerName := getPlayerName(clientSocket, bufferData)

			// log to msh console and answer to client with text in the loadscreen
			log.Printf("*** %s tried to join from %s:%s to %s:%s during server startup\n", playerName, clientAddress, config.ConfigRuntime.Msh.Port, config.TargetHost, config.TargetPort)
			clientSocket.Write(buildMessage("txt", "Server is starting. Please wait... "+servctrl.Stats.LoadProgress))
		}

		// close the client connection
		logger.Logln("closing connection for:", clientAddress)
		clientSocket.Close()

	case "online":
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
