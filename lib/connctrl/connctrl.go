package connctrl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"msh/lib/asyncctrl"
	"msh/lib/confctrl"
	"msh/lib/debugctrl"
	"msh/lib/servctrl"
	"msh/lib/servprotocol"
)

// HandleClientSocket handles a client that is connecting.
// Can handle a client that is requesting server info or trying to join.
func HandleClientSocket(clientSocket net.Conn) {
	// handling of ipv6 addresses
	var lastIndex int = strings.LastIndex(clientSocket.RemoteAddr().String(), ":")
	clientAddress := clientSocket.RemoteAddr().String()[:lastIndex]

	debugctrl.Logger(fmt.Sprintf("*** start proxy from %s:%s to %s:%s", clientAddress, confctrl.Config.Advanced.ListenPort, confctrl.Config.Advanced.TargetHost, confctrl.Config.Advanced.TargetPort))

	// block containing the case of serverStatus == "offline" or "starting"
	if servctrl.ServerStatus == "offline" || servctrl.ServerStatus == "starting" {
		buffer := make([]byte, 1024)

		// read first packet
		dataLen, err := clientSocket.Read(buffer)
		if err != nil {
			debugctrl.Logger("handleClientSocket: error during clientSocket.Read() 1")
			return
		}

		// the client first packet is {data, 1, 1, 0} or {data, 1} --> the client is requesting server info and ping
		if buffer[dataLen-1] == 0 || buffer[dataLen-1] == 1 {
			if servctrl.ServerStatus == "offline" {
				log.Printf("*** player unknown requested server info from %s:%s to %s:%s\n", clientAddress, confctrl.Config.Advanced.ListenPort, confctrl.Config.Advanced.TargetHost, confctrl.Config.Advanced.TargetPort)
				// answer to client with emulated server info
				clientSocket.Write(servprotocol.BuildMessage("info", confctrl.Config.Basic.HibernationInfo))

			} else if servctrl.ServerStatus == "starting" {
				log.Printf("*** player unknown requested server info from %s:%s to %s:%s during server startup\n", clientAddress, confctrl.Config.Advanced.ListenPort, confctrl.Config.Advanced.TargetHost, confctrl.Config.Advanced.TargetPort)
				// answer to client with emulated server info
				clientSocket.Write(servprotocol.BuildMessage("info", confctrl.Config.Basic.StartingInfo))
			}

			// answer to client with ping
			servprotocol.AnswerPingReq(clientSocket)
		}

		// the client first message is [data, listenPortBytes, 2] or [data, listenPortBytes, 2, playerNameData] -->
		// the client is trying to join the server
		listenPortInt, err := strconv.Atoi(confctrl.Config.Advanced.ListenPort)
		if err != nil {
			debugctrl.Logger("handleClientSocket: error during ListenPort conversion to int")
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
					debugctrl.Logger("handleClientSocket: error during clientSocket.Read() 2")
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

			if servctrl.ServerStatus == "offline" {
				// client is trying to join the server and serverStatus == "offline" --> issue startMinecraftServer()
				servctrl.StartMinecraftServer()
				log.Printf("*** %s tried to join from %s:%s to %s:%s\n", playerName, clientAddress, confctrl.Config.Advanced.ListenPort, confctrl.Config.Advanced.TargetHost, confctrl.Config.Advanced.TargetPort)
				// answer to client with text in the loadscreen
				clientSocket.Write(servprotocol.BuildMessage("txt", fmt.Sprintf("Server start command issued. Please wait... Time left: %d seconds", confctrl.TimeLeftUntilUp)))

			} else if servctrl.ServerStatus == "starting" {
				log.Printf("*** %s tried to join from %s:%s to %s:%s during server startup\n", playerName, clientAddress, confctrl.Config.Advanced.ListenPort, confctrl.Config.Advanced.TargetHost, confctrl.Config.Advanced.TargetPort)
				// answer to client with text in the loadscreen
				clientSocket.Write(servprotocol.BuildMessage("txt", fmt.Sprintf("Server is starting. Please wait... Time left: %d seconds", confctrl.TimeLeftUntilUp)))
			}
		}

		// since the server is still not online, close the client connection
		debugctrl.Logger(fmt.Sprintf("closing connection for: %s", clientAddress))
		clientSocket.Close()
	}

	// block containing the case of serverStatus == "online"
	if servctrl.ServerStatus == "online" {
		// if the server is online, just open a connection with the server and connect it with the client
		serverSocket, err := net.Dial("tcp", confctrl.Config.Advanced.TargetHost+":"+confctrl.Config.Advanced.TargetPort)
		if err != nil {
			debugctrl.Logger("handleClientSocket: error during serverSocket.Dial()")
			// report dial error to client with text in the loadscreen
			clientSocket.Write(servprotocol.BuildMessage("txt", fmt.Sprintf("can't connect to server... check if minecraft server is running and set the correct targetPort")))
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
	servctrl.Players++
	log.Printf("*** A PLAYER JOINED THE SERVER! - %d players online", servctrl.Players)

	// exchanges data from client to server (isServerToClient == false)
	forwardSync(source, destination, false, stopSig)

	servctrl.Players--
	log.Printf("*** A PLAYER LEFT THE SERVER! - %d players online", servctrl.Players)

	// this block increases stopInstances by one and starts the timer to execute stopEmptyMinecraftServer(false)
	// (that will do nothing in case there are players online)
	asyncctrl.Mutex.Lock()
	servctrl.StopInstances++
	asyncctrl.Mutex.Unlock()
	time.AfterFunc(time.Duration(confctrl.Config.Basic.TimeBeforeStoppingEmptyServer)*time.Second, func() { servctrl.StopEmptyMinecraftServer(false) })
}

func serverToClient(source, destination net.Conn, stopSig *bool) {
	// exchanges data from server to client (isServerToClient == true)
	forwardSync(source, destination, true, stopSig)
}

// forwardSync takes a source and a destination net.Conn and forwards them.
// (isServerToClient used to know the forward direction)
func forwardSync(source, destination net.Conn, isServerToClient bool, stopSig *bool) {
	data := make([]byte, 1024)

	// set to false after the first for cycle
	firstBuffer := true

	for {
		if *stopSig {
			// if stopSig is true, close the source connection
			source.Close()
			break
		}

		// update read and write timeout
		source.SetReadDeadline(time.Now().Add(time.Duration(confctrl.Config.Basic.TimeBeforeStoppingEmptyServer) * time.Second))
		destination.SetWriteDeadline(time.Now().Add(time.Duration(confctrl.Config.Basic.TimeBeforeStoppingEmptyServer) * time.Second))

		// read data from source
		dataLen, err := source.Read(data)
		if err != nil {
			// case in which the connection is closed by the source or closed by target
			if err == io.EOF {
				debugctrl.Logger(fmt.Sprintf("closing %s --> %s because of: %s", strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0], err.Error()))
			} else {
				debugctrl.Logger(fmt.Sprintf("forwardSync: error in forward(): %v\n%s --> %s", err, strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0]))
			}

			// close the source connection
			asyncctrl.Mutex.Lock()
			*stopSig = true
			asyncctrl.Mutex.Unlock()
			source.Close()
			break
		}

		// write data to destination
		destination.Write(data[:dataLen])

		// calculate bytes/s to client/server
		if confctrl.Config.Advanced.Debug {
			asyncctrl.Mutex.Lock()
			if isServerToClient {
				debugctrl.DataCountBytesToClients = debugctrl.DataCountBytesToClients + float64(dataLen)
			} else {
				debugctrl.DataCountBytesToServer = debugctrl.DataCountBytesToServer + float64(dataLen)
			}
			asyncctrl.Mutex.Unlock()
		}

		// version/protocol are only found in serverToClient connection in the first buffer that is read
		if firstBuffer && isServerToClient {
			searchVersionProtocol(data[:dataLen])

			// first cycle is finished, set firstBuffer = false
			firstBuffer = false
		}
	}
}

// searchVersionProtocol finds the serverVersion and serverProtocol and writes them in the config file
func searchVersionProtocol(data []byte) {
	// if the above specified buffer contains "\"version\":{\"name\":\"" and ",\"protocol\":" --> extract the serverVersion and serverProtocol
	if bytes.Contains(data, []byte("\"version\":{\"name\":\"")) && bytes.Contains(data, []byte(",\"protocol\":")) {
		newServerVersion := string(bytes.Split(bytes.Split(data, []byte("\"version\":{\"name\":\""))[1], []byte("\","))[0])
		newServerProtocol := string(bytes.Split(bytes.Split(data, []byte(",\"protocol\":"))[1], []byte("}"))[0])

		// if serverVersion or serverProtocol are different from the ones specified in config.json --> update them
		if newServerVersion != confctrl.Config.Advanced.ServerVersion || newServerProtocol != confctrl.Config.Advanced.ServerProtocol {
			debugctrl.Logger(
				"server version found!",
				"serverVersion:", newServerVersion,
				"serverProtocol:", newServerProtocol,
			)

			confctrl.Config.Advanced.ServerVersion = newServerVersion
			confctrl.Config.Advanced.ServerProtocol = newServerProtocol

			confctrl.SaveConfig()
		}
	}
}
