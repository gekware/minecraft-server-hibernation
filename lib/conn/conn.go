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

	reqPacket, reqType, playerName, logMsh := getReqType(clientSocket)
	if logMsh != nil {
		errco.Log(logMsh.AddTrace())
		return
	}

	// if ms has a major error warn the client and return
	if servstats.Stats.MajorError != nil {
		// close the client connection at the end
		defer func() {
			errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "closing connection for: %s", clientAddress)
			clientSocket.Close()
		}()

		switch reqType {
		case errco.CLIENT_REQ_INFO:
			// log to msh console and answer to client with error
			errco.Logln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_MINECRAFT_SERVER, "%s requested server info from %s:%d to %s:%d but server has encountered major problems", playerName, clientAddress, config.ListenPort, config.TargetHost, config.TargetPort)
			mes := buildMessage(reqType, string(servstats.Stats.MajorError.Ori)+": "+servstats.Stats.MajorError.Mex)
			clientSocket.Write(mes)
			errco.Logln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s:%v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

			// answer to client ping
			logMsh = getPing(clientSocket)
			if logMsh != nil {
				errco.Log(logMsh.AddTrace())
			}
		case errco.CLIENT_REQ_JOIN:
			// log to msh console and answer to client with error
			errco.Logln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_MINECRAFT_SERVER, "%s requested server info from %s:%d to %s:%d but server has encountered major problems", playerName, clientAddress, config.ListenPort, config.TargetHost, config.TargetPort)
			mes := buildMessage(reqType, string(servstats.Stats.MajorError.Ori)+": "+servstats.Stats.MajorError.Mex)
			clientSocket.Write(mes)
			errco.Logln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s:%v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)
		}

		return
	}

	switch reqType {
	case errco.CLIENT_REQ_INFO:
		errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "%s requested server info from %s:%d to %s:%d", playerName, clientAddress, config.ListenPort, config.TargetHost, config.TargetPort)

		if servstats.Stats.Status != errco.SERVER_STATUS_ONLINE || servstats.Stats.Suspended {
			// ms not online or suspended

			defer func() {
				// close the client connection
				errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "closing connection for: %s", clientAddress)
				clientSocket.Close()
			}()

			// answer to client with emulated server info
			var mes []byte
			switch servstats.Stats.Status {
			case errco.SERVER_STATUS_OFFLINE:
				mes = buildMessage(reqType, config.ConfigRuntime.Msh.InfoHibernation)
			case errco.SERVER_STATUS_STARTING:
				mes = buildMessage(reqType, config.ConfigRuntime.Msh.InfoStarting)
			case errco.SERVER_STATUS_ONLINE: // ms suspended
				mes = buildMessage(reqType, config.ConfigRuntime.Msh.InfoHibernation)
			case errco.SERVER_STATUS_STOPPING:
				mes = buildMessage(reqType, "server is stopping...\nrefresh the page")
			}
			clientSocket.Write(mes)
			errco.Logln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s:%v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

			// answer to client ping
			logMsh := getPing(clientSocket)
			if logMsh != nil {
				errco.Log(logMsh.AddTrace())
				return
			}

		} else {
			// ms online and not suspended

			// open proxy between client and server
			openProxy(clientSocket, reqPacket)
		}

	case errco.CLIENT_REQ_JOIN:
		errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "%s tried to join from %s:%d to %s:%d", playerName, clientAddress, config.ListenPort, config.TargetHost, config.TargetPort)

		if servstats.Stats.Status != errco.SERVER_STATUS_ONLINE {
			// ms not online (un/suspended)

			defer func() {
				// close the client connection
				errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "closing connection for: %s", clientAddress)
				clientSocket.Close()
			}()

			// check if the client address or player name are whitelisted
			logMsh := config.ConfigRuntime.InWhitelist(playerName, clientAddress)
			if logMsh != nil {
				// warn client with text in the loadscreen
				errco.Log(logMsh.AddTrace())
				mes := buildMessage(reqType, "You don't have permission to warm this server")
				clientSocket.Write(mes)
				errco.Logln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s:%v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)
				return
			}

			// issue warm
			logMsh = servctrl.WarmMS()
			if logMsh != nil {
				// warn client with text in the loadscreen
				errco.Log(logMsh.AddTrace())
				mes := buildMessage(reqType, "An error occurred while warming the server: check the msh log")
				clientSocket.Write(mes)
				errco.Logln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s:%v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)
				return
			}

			// answer client with text in the loadscreen
			mes := buildMessage(reqType, "Server start command issued. Please wait... "+servstats.Stats.LoadProgress)
			clientSocket.Write(mes)
			errco.Logln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s:%v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

		} else {
			// ms online (un/suspended)

			// issue warm
			logMsh = servctrl.WarmMS()
			if logMsh != nil {
				// warn client with text in the loadscreen
				errco.Log(logMsh.AddTrace())
				mes := buildMessage(errco.MESSAGE_FORMAT_TXT, "An error occurred while warming the server: check the msh log")
				clientSocket.Write(mes)
				errco.Logln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s:%v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)
				return
			}

			// open proxy between client and server
			openProxy(clientSocket, reqPacket)
		}
	}
}

func openProxy(clientSocket net.Conn, serverInitPacket []byte) {
	// open a connection to ms and connect it with the client
	serverSocket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", config.TargetHost, config.TargetPort))
	if err != nil {
		errco.Logln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_DIAL, err.Error())
		// report dial error to client with text in the loadscreen
		mes := buildMessage(errco.CLIENT_REQ_JOIN, "can't connect to server... check if minecraft server is running and set the correct targetPort")
		clientSocket.Write(mes)
		errco.Logln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s:%v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)
		return
	}

	// stopC is used to close serv->client and client->serv at the same time
	stopC := make(chan bool, 1)

	// forward the request packet data
	serverSocket.Write(serverInitPacket)

	// launch proxy client -> server
	go forward(clientSocket, serverSocket, false, stopC)

	// launch proxy server -> client
	go forward(serverSocket, clientSocket, true, stopC)
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
				errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "closing %15s --> %15s because of: %s", strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0], err.Error())
			} else {
				errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "%s\n%15s --> %15s", err.Error(), strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0])
			}

			// close the source connection
			stopC <- true
			source.Close()
			return
		}

		// write data to destination
		destination.Write(data[:dataLen])

		// calculate bytes/s to client/server
		if errco.DebugLvl >= errco.LVL_3 {
			servstats.Stats.M.Lock()
			if isServerToClient {
				servstats.Stats.BytesToClients = servstats.Stats.BytesToClients + float64(dataLen)
				errco.Logln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%sserver --> client%s:%v", errco.COLOR_BLUE, errco.COLOR_RESET, data[:dataLen])
			} else {
				servstats.Stats.BytesToServer = servstats.Stats.BytesToServer + float64(dataLen)
				errco.Logln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%sclient --> server%s:%v", errco.COLOR_GREEN, errco.COLOR_RESET, data[:dataLen])
			}
			servstats.Stats.M.Unlock()
		}
	}
}
