package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"msh/lib/asyncctrl"
	"msh/lib/confctrl"
	"msh/lib/data"
	"msh/lib/debugctrl"
	"msh/lib/progctrl"
	"msh/lib/servctrl"
)

// script version
var version string = "v2.1.2"

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

	// load configuration from config file
	// load server-icon-frozen.png if present
	confctrl.LoadConfig()

	// check for updates
	if confctrl.Config.Basic.CheckForUpdates {
		progctrl.UpdateChecker(version)
	}

	// listen for interrupt signals
	progctrl.InterruptListener()

	// launch printDataUsage()
	go debugctrl.PrintDataUsage()

	// open a listener on {config.Advanced.ListenHost}+":"+{config.Advanced.ListenPort}
	listener, err := net.Listen("tcp", confctrl.Config.Advanced.ListenHost+":"+confctrl.Config.Advanced.ListenPort)
	if err != nil {
		log.Printf("main: Fatal error: %s", err.Error())
		time.Sleep(time.Duration(5) * time.Second)
		os.Exit(1)
	}

	defer func() {
		debugctrl.Logger("Closing connection for: listener")
		listener.Close()
	}()

	log.Println("*** listening for new clients to connect...")

	// infinite cycle to accept clients. when a clients connects it is passed to handleClientSocket()
	for {
		clientSocket, err := listener.Accept()
		if err != nil {
			debugctrl.Logger("main:", err.Error())
			continue
		}
		handleClientSocket(clientSocket)
	}
}

//---------------------connection management------------------//

// to handle a client that is connecting.
// can handle a client that is requesting server info or trying to join.
func handleClientSocket(clientSocket net.Conn) {
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
				clientSocket.Write(buildMessage("info", confctrl.Config.Basic.HibernationInfo))

			} else if servctrl.ServerStatus == "starting" {
				log.Printf("*** player unknown requested server info from %s:%s to %s:%s during server startup\n", clientAddress, confctrl.Config.Advanced.ListenPort, confctrl.Config.Advanced.TargetHost, confctrl.Config.Advanced.TargetPort)
				// answer to client with emulated server info
				clientSocket.Write(buildMessage("info", confctrl.Config.Basic.StartingInfo))
			}

			// answer to client with ping
			answerPingReq(clientSocket)
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
				clientSocket.Write(buildMessage("txt", fmt.Sprintf("Server start command issued. Please wait... Time left: %d seconds", confctrl.TimeLeftUntilUp)))

			} else if servctrl.ServerStatus == "starting" {
				log.Printf("*** %s tried to join from %s:%s to %s:%s during server startup\n", playerName, clientAddress, confctrl.Config.Advanced.ListenPort, confctrl.Config.Advanced.TargetHost, confctrl.Config.Advanced.TargetPort)
				// answer to client with text in the loadscreen
				clientSocket.Write(buildMessage("txt", fmt.Sprintf("Server is starting. Please wait... Time left: %d seconds", confctrl.TimeLeftUntilUp)))
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
			clientSocket.Write(buildMessage("txt", fmt.Sprintf("can't connect to server... check if minecraft server is running and set the correct targetPort")))
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

		// if debug == true --> calculate bytes/s to client/server
		if confctrl.Config.Advanced.Debug {
			asyncctrl.Mutex.Lock()
			if isServerToClient {
				debugctrl.DataCountBytesToClients = debugctrl.DataCountBytesToClients + float64(dataLen)
			} else {
				debugctrl.DataCountBytesToServer = debugctrl.DataCountBytesToServer + float64(dataLen)
			}
			asyncctrl.Mutex.Unlock()
		}

		// this block is used to find the serverVersion and serverProtocol.
		// these parameters are only found in serverToClient connection in the first buffer that is read
		// if the above specified buffer contains "\"version\":{\"name\":\"" and ",\"protocol\":" --> extract the serverVersion and serverProtocol
		if isServerToClient && firstBuffer && bytes.Contains(data[:dataLen], []byte("\"version\":{\"name\":\"")) && bytes.Contains(data[:dataLen], []byte(",\"protocol\":")) {
			newServerVersion := string(bytes.Split(bytes.Split(data[:dataLen], []byte("\"version\":{\"name\":\""))[1], []byte("\","))[0])
			newServerProtocol := string(bytes.Split(bytes.Split(data[:dataLen], []byte(",\"protocol\":"))[1], []byte("}"))[0])

			// if serverVersion or serverProtocol are different from the ones specified in config.json --> update them
			if newServerVersion != confctrl.Config.Advanced.ServerVersion || newServerProtocol != confctrl.Config.Advanced.ServerProtocol {
				confctrl.Config.Advanced.ServerVersion = newServerVersion
				confctrl.Config.Advanced.ServerProtocol = newServerProtocol

				debugctrl.Logger(
					"server version found!",
					"serverVersion:", confctrl.Config.Advanced.ServerVersion,
					"serverProtocol:", confctrl.Config.Advanced.ServerProtocol,
				)

				// write the struct config to json data
				configData, err := json.MarshalIndent(confctrl.Config, "", "  ")
				if err != nil {
					debugctrl.Logger("forwardSync: could not marshal configuration")
					continue
				}
				// write json data to config.json
				err = ioutil.WriteFile("config.json", configData, 0644)
				if err != nil {
					debugctrl.Logger("forwardSync: could not update config.json")
					continue
				}
				debugctrl.Logger("saved to config.json")
			}
		}

		// first cycle is finished, set firstBuffer = false
		firstBuffer = false
	}
}

//-----------------server connection protocol-----------------//

// takes the format ("txt", "info") and a message to write to the client
func buildMessage(format, message string) []byte {
	var mountHeader = func(messageStr string) []byte {
		// mountHeader: mounts the complete header to a specified message
		//					┌--------------------complete header--------------------┐
		// scheme: 			[sub-header1		|sub-header2 	|sub-header3		|message	]
		// bytes used:		[2					|1				|2					|0 ... 16381]
		// value range:		[131 0 - 255 127	|0				|128 0 - 252 127	|---		]

		var addSubHeader = func(message []byte) []byte {
			// addSubHeader: mounts 1 sub-header to a specified message
			//				┌sub-header1/sub-header3┐
			// scheme:		[firstByte	|secondByte	|data	]
			// value range:	[128-255	|0-127		|---	]
			// it's a number composed of 2 digits in base-128 (firstByte is least significant byte)
			// sub-header represents the lenght of the following data

			firstByte := len(message)%128 + 128
			secondByte := math.Floor(float64(len(message) / 128))
			return append([]byte{byte(firstByte), byte(secondByte)}, message...)
		}

		messageByte := []byte(messageStr)

		// sub-header3 calculation
		messageByte = addSubHeader(messageByte)

		// sub-header2 calculation
		messageByte = append([]byte{0}, messageByte...)

		// sub-header1 calculation
		messageByte = addSubHeader(messageByte)

		return messageByte
	}

	var messageHeader []byte

	if format == "txt" {
		// to display text in the loadscreen

		messageJSON := fmt.Sprint(
			"{",
			"\"text\":\"", message, "\"",
			"}",
		)

		messageHeader = mountHeader(messageJSON)

	} else if format == "info" {
		// to send server info

		// in message: "\n" -> "&r\\n" then "&" -> "\xc2\xa7"
		messageAdapted := strings.ReplaceAll(strings.ReplaceAll(message, "\n", "&r\\n"), "&", "\xc2\xa7")

		messageJSON := fmt.Sprint("{",
			"\"description\":{\"text\":\"", messageAdapted, "\"},",
			"\"players\":{\"max\":0,\"online\":0},",
			"\"version\":{\"name\":\"", confctrl.Config.Advanced.ServerVersion, "\",\"protocol\":", fmt.Sprint(confctrl.Config.Advanced.ServerProtocol), "},",
			"\"favicon\":\"data:image/png;base64,", data.ServerIcon, "\"",
			"}",
		)

		messageHeader = mountHeader(messageJSON)
	}

	return messageHeader
}

// responds to the ping request
func answerPingReq(clientSocket net.Conn) {
	req := make([]byte, 1024)

	// read the first packet
	dataLen, err := clientSocket.Read(req)
	if err != nil {
		debugctrl.Logger("answerPingReq: error while reading [1] ping request:", err.Error())
		return
	}

	// if req == [1, 0] --> read again (the correct ping byte array has still to arrive)
	if bytes.Equal(req[:dataLen], []byte{1, 0}) {
		dataLen, err = clientSocket.Read(req)
		if err != nil {
			debugctrl.Logger("answerPingReq: error while reading [2] ping request:", err.Error())
			return
		}
	} else if bytes.Equal(req[:2], []byte{1, 0}) {
		// sometimes the [1 0] is at the beginning and needs to be removed.
		// Example: [1 0 9 1 0 0 0 0 0 89 73 114] -> [9 1 0 0 0 0 0 89 73 114]
		req = req[2:dataLen]
		dataLen = dataLen - 2
	}

	// answer the ping request
	clientSocket.Write(req[:dataLen])
}
