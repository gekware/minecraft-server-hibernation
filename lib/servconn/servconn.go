package servconn

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strconv"
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

	// block containing the case of serverStatus == "offline" or "starting"
	if servctrl.Stats.Status == "offline" || servctrl.Stats.Status == "starting" {
		buffer := make([]byte, 1024)

		// read first packet
		dataLen, err := clientSocket.Read(buffer)
		if err != nil {
			logger.Logln("handleClientSocket: error during clientSocket.Read() 1")
			return
		}

		// the client first packet is {data, 1, 1, 0} or {data, 1} --> the client is requesting server info and ping
		if buffer[dataLen-1] == 0 || buffer[dataLen-1] == 1 {
			if servctrl.Stats.Status == "offline" {
				log.Printf("*** player unknown requested server info from %s:%s to %s:%s\n", clientAddress, config.ConfigRuntime.Msh.Port, config.TargetHost, config.TargetPort)
				// answer to client with emulated server info
				clientSocket.Write(BuildMessage("info", config.ConfigRuntime.Msh.InfoHibernation))

			} else if servctrl.Stats.Status == "starting" {
				log.Printf("*** player unknown requested server info from %s:%s to %s:%s during server startup\n", clientAddress, config.ConfigRuntime.Msh.Port, config.TargetHost, config.TargetPort)
				// answer to client with emulated server info
				clientSocket.Write(BuildMessage("info", config.ConfigRuntime.Msh.InfoStarting))
			}

			// answer to client with ping
			err = AnswerPingReq(clientSocket)
			if err != nil {
				logger.Logln("handleClientSocket:", err)
			}
		}

		// the client first message is [data, listenPortBytes, 2] or [data, listenPortBytes, 2, playerNameData] -->
		// the client is trying to join the server
		listenPortUint64, err := strconv.ParseUint(config.ConfigRuntime.Msh.Port, 10, 16) // bitSize: 16 -> since it will be converted to Uint16
		if err != nil {
			logger.Logln("handleClientSocket: error during ListenPort conversion to uint64")
		}
		listenPortUint16 := uint16(listenPortUint64)
		listenPortBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(listenPortBytes, listenPortUint16) // 25555 ->	[99 211] / hex[63 D3]
		listenPortJoinBytes := append(listenPortBytes, byte(2))       // 			[99 211 2] / hex[63 D3 2]

		if bytes.Contains(buffer[:dataLen], listenPortJoinBytes) {
			var playerName string

			// if [99 211 2] are the last bytes then there is only the join request
			// read again the client socket to get the player name packet
			if bytes.Index(buffer[:dataLen], listenPortJoinBytes) == dataLen-3 {
				dataLen, err = clientSocket.Read(buffer)
				if err != nil {
					logger.Logln("handleClientSocket: error during clientSocket.Read() 2")
					return
				}
				playerName = string(buffer[3:dataLen])
			} else {
				// the packet contains the join request and the player name in the scheme:
				// [... 99 211 2 X X X (player name) 0 0 0 0 0...]
				//  ^-dataLen----------------------^
				//                                   ^-zerosLen-^
				//               ^-playerNameBuffer-------------^
				zerosLen := len(buffer) - dataLen
				playerNameBuffer := bytes.SplitAfter(buffer, listenPortJoinBytes)[1]
				playerName = string(playerNameBuffer[3 : len(playerNameBuffer)-zerosLen])
			}

			if servctrl.Stats.Status == "offline" {
				// client is trying to join the server and serverStatus == "offline" --> issue StartMS()
				err = servctrl.StartMS()
				if err != nil {
					// log to msh console and warn client with text in the loadscreen
					logger.Logln("HandleClientSocket:", err)
					clientSocket.Write(BuildMessage("txt", "An error occurred while starting the server: check the msh log"))
				} else {
					// log to msh console and answer to client with text in the loadscreen
					log.Printf("*** %s tried to join from %s:%s to %s:%s\n", playerName, clientAddress, config.ConfigRuntime.Msh.Port, config.TargetHost, config.TargetPort)
					clientSocket.Write(BuildMessage("txt", "Server start command issued. Please wait... "+servctrl.Stats.LoadProgress))
				}

			} else if servctrl.Stats.Status == "starting" {
				log.Printf("*** %s tried to join from %s:%s to %s:%s during server startup\n", playerName, clientAddress, config.ConfigRuntime.Msh.Port, config.TargetHost, config.TargetPort)
				// answer to client with text in the loadscreen
				clientSocket.Write(BuildMessage("txt", "Server is starting. Please wait... "+servctrl.Stats.LoadProgress))
			}
		}

		// since the server is still not online, close the client connection
		logger.Logln(fmt.Sprintf("closing connection for: %s", clientAddress))
		clientSocket.Close()
	}

	// block containing the case of serverStatus == "online"
	if servctrl.Stats.Status == "online" {
		// if the server is online, just open a connection with the server and connect it with the client
		serverSocket, err := net.Dial("tcp", config.TargetHost+":"+config.TargetPort)
		if err != nil {
			logger.Logln("handleClientSocket: error during serverSocket.Dial()")
			// report dial error to client with text in the loadscreen
			clientSocket.Write(BuildMessage("txt", "can't connect to server... check if minecraft server is running and set the correct targetPort"))
			return
		}

		// stopC is used to close serv->client and client->serv at the same time
		stopC := make(chan bool, 1)

		// launch proxy client -> server
		go Forward(clientSocket, serverSocket, false, stopC)

		// launch proxy server -> client
		go Forward(serverSocket, clientSocket, true, stopC)
	}
}
