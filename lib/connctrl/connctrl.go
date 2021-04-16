package connctrl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"msh/lib/confctrl"
	"msh/lib/debugctrl"
	"msh/lib/proxy"
	"msh/lib/servctrl"
	"msh/lib/servprotocol"
)

// HandleClientSocket handles a client that is connecting.
// Can handle a client that is requesting server info or trying to join.
// [goroutine]
func HandleClientSocket(clientSocket net.Conn) {
	// handling of ipv6 addresses
	li := strings.LastIndex(clientSocket.RemoteAddr().String(), ":")
	clientAddress := clientSocket.RemoteAddr().String()[:li]

	// block containing the case of serverStatus == "offline" or "starting"
	if servctrl.ServStats.Status == "offline" || servctrl.ServStats.Status == "starting" {
		buffer := make([]byte, 1024)

		// read first packet
		dataLen, err := clientSocket.Read(buffer)
		if err != nil {
			debugctrl.Log("handleClientSocket: error during clientSocket.Read() 1")
			return
		}

		// the client first packet is {data, 1, 1, 0} or {data, 1} --> the client is requesting server info and ping
		if buffer[dataLen-1] == 0 || buffer[dataLen-1] == 1 {
			if servctrl.ServStats.Status == "offline" {
				log.Printf("*** player unknown requested server info from %s:%s to %s:%s\n", clientAddress, confctrl.Config.Msh.Port, confctrl.TargetHost, confctrl.TargetPort)
				// answer to client with emulated server info
				clientSocket.Write(servprotocol.BuildMessage("info", confctrl.Config.Msh.HibernationInfo))

			} else if servctrl.ServStats.Status == "starting" {
				log.Printf("*** player unknown requested server info from %s:%s to %s:%s during server startup\n", clientAddress, confctrl.Config.Msh.Port, confctrl.TargetHost, confctrl.TargetPort)
				// answer to client with emulated server info
				clientSocket.Write(servprotocol.BuildMessage("info", confctrl.Config.Msh.StartingInfo))
			}

			// answer to client with ping
			err = servprotocol.AnswerPingReq(clientSocket)
			if err != nil {
				debugctrl.Log("handleClientSocket: %v", err)
			}
		}

		// the client first message is [data, listenPortBytes, 2] or [data, listenPortBytes, 2, playerNameData] -->
		// the client is trying to join the server
		listenPortInt, err := strconv.Atoi(confctrl.Config.Msh.Port)
		if err != nil {
			debugctrl.Log("handleClientSocket: error during ListenPort conversion to int")
		}
		listenPortBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(listenPortBytes, uint16(listenPortInt)) // 25555 ->	[99 211] / hex[63 D3]
		listenPortJoinBytes := append(listenPortBytes, byte(2))            // 			[99 211 2] / hex[63 D3 2]

		if bytes.Contains(buffer[:dataLen], listenPortJoinBytes) {
			var playerName string

			// if [99 211 2] are the last bytes then there is only the join request
			// read again the client socket to get the player name packet
			if bytes.Index(buffer[:dataLen], listenPortJoinBytes) == dataLen-3 {
				dataLen, err = clientSocket.Read(buffer)
				if err != nil {
					debugctrl.Log("handleClientSocket: error during clientSocket.Read() 2")
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

			if servctrl.ServStats.Status == "offline" {
				// client is trying to join the server and serverStatus == "offline" --> issue startMinecraftServer()
				err = servctrl.StartMinecraftServer()
				if err != nil {
					// log to msh console and warn client with text in the loadscreen
					debugctrl.Log("HandleClientSocket: %v", err)
					clientSocket.Write(servprotocol.BuildMessage("txt", "An error occurred while starting the server: check the msh log"))
				} else {
					// log to msh console and answer to client with text in the loadscreen
					log.Printf("*** %s tried to join from %s:%s to %s:%s\n", playerName, clientAddress, confctrl.Config.Msh.Port, confctrl.TargetHost, confctrl.TargetPort)
					clientSocket.Write(servprotocol.BuildMessage("txt", "Server start command issued. Please wait... "+servctrl.ServStats.LoadProgress))
				}

			} else if servctrl.ServStats.Status == "starting" {
				log.Printf("*** %s tried to join from %s:%s to %s:%s during server startup\n", playerName, clientAddress, confctrl.Config.Msh.Port, confctrl.TargetHost, confctrl.TargetPort)
				// answer to client with text in the loadscreen
				clientSocket.Write(servprotocol.BuildMessage("txt", "Server is starting. Please wait... "+servctrl.ServStats.LoadProgress))
			}
		}

		// since the server is still not online, close the client connection
		debugctrl.Log(fmt.Sprintf("closing connection for: %s", clientAddress))
		clientSocket.Close()
	}

	// block containing the case of serverStatus == "online"
	if servctrl.ServStats.Status == "online" {
		// if the server is online, just open a connection with the server and connect it with the client
		serverSocket, err := net.Dial("tcp", confctrl.TargetHost+":"+confctrl.TargetPort)
		if err != nil {
			debugctrl.Log("handleClientSocket: error during serverSocket.Dial()")
			// report dial error to client with text in the loadscreen
			clientSocket.Write(servprotocol.BuildMessage("txt", "can't connect to server... check if minecraft server is running and set the correct targetPort"))
			return
		}

		// stopSig is used to close serv->client and client->serv at the same time
		stopSig := false

		// launch clientToServer() and serverToClient()
		go clientToServer(clientSocket, serverSocket, &stopSig)
		go serverToClient(serverSocket, clientSocket, &stopSig)
	}
}

func clientToServer(source, destination net.Conn, stopSig *bool) {
	servctrl.ServStats.Players++
	log.Printf("*** A PLAYER JOINED THE SERVER! - %d players online", servctrl.ServStats.Players)

	// exchanges data from client to server (isServerToClient == false)
	proxy.Forward(source, destination, false, stopSig)

	servctrl.ServStats.Players--
	log.Printf("*** A PLAYER LEFT THE SERVER! - %d players online", servctrl.ServStats.Players)

	servctrl.RequestStopMinecraftServer()
}

func serverToClient(source, destination net.Conn, stopSig *bool) {
	// exchanges data from server to client (isServerToClient == true)
	proxy.Forward(source, destination, true, stopSig)
}
