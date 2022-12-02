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
// Can handle a client that is requesting server INFO or server JOIN.
// If there is a ms major error, it is reported to client then func returns.
// [goroutine]
func HandleClientSocket(clientSocket net.Conn) {
	// handling of ipv6 addresses
	li := strings.LastIndex(clientSocket.RemoteAddr().String(), ":")
	clientAddress := clientSocket.RemoteAddr().String()[:li]

	// get request type from client
	reqPacket, reqType, logMsh := getReqType(clientSocket)
	if logMsh != nil {
		logMsh.Log(true)
		return
	}

	// if there is a major error warn the client and return
	if servstats.Stats.MajorError != nil {
		errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_MINECRAFT_SERVER, "a client connected to msh (%s:%d to %s:%d) but minecraft server has encountered major problems", clientAddress, config.ListenPort, config.TargetHost, config.TargetPort)

		// close the client connection before returning
		defer func() {
			errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "closing connection for: %s", clientAddress)
			clientSocket.Close()
		}()

		// msh INFO/JOIN response (warn client with error description)
		mes := buildMessage(reqType, fmt.Sprintf(servstats.Stats.MajorError.Mex, servstats.Stats.MajorError.Arg...))
		clientSocket.Write(mes)
		errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

		// msh PING response if it was a client INFO request
		if reqType == errco.CLIENT_REQ_INFO {
			logMsh = getPing(clientSocket)
			if logMsh != nil {
				logMsh.Log(true)
			}
		}

		return
	}

	// handle the request depending on request type
	switch reqType {
	case errco.CLIENT_REQ_INFO:
		errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "a client requested server info from %s:%d to %s:%d", clientAddress, config.ListenPort, config.TargetHost, config.TargetPort)

		if servstats.Stats.Status != errco.SERVER_STATUS_ONLINE || servstats.Stats.Suspended {
			// ms not online or suspended

			defer func() {
				// close the client connection before returning
				errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "closing connection for: %s", clientAddress)
				clientSocket.Close()
			}()

			// msh INFO response
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
			errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

			// msh PING response
			logMsh := getPing(clientSocket)
			if logMsh != nil {
				logMsh.Log(true)
				return
			}

		} else {
			// ms online and not suspended

			// open proxy between client and server
			openProxy(clientSocket, reqPacket, errco.CLIENT_REQ_INFO)
		}

	case errco.CLIENT_REQ_JOIN:
		errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "a client tried to join from %s:%d to %s:%d", clientAddress, config.ListenPort, config.TargetHost, config.TargetPort)

		if servstats.Stats.Status != errco.SERVER_STATUS_ONLINE {
			// ms not online (un/suspended)

			defer func() {
				// close the client connection before returning
				errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "closing connection for: %s", clientAddress)
				clientSocket.Close()
			}()

			// check if the request packet contains element of whitelist or the address is in whitelist
			logMsh := config.ConfigRuntime.IsWhitelist(reqPacket, clientAddress)
			if logMsh != nil {
				logMsh.Log(true)

				// msh JOIN response (warn client with text in the loadscreen)
				mes := buildMessage(reqType, "You don't have permission to warm this server")
				clientSocket.Write(mes)
				errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

				return
			}

			// issue warm
			logMsh = servctrl.WarmMS()
			if logMsh != nil {
				// msh JOIN response (warn client with text in the loadscreen)
				logMsh.Log(true)
				mes := buildMessage(reqType, "An error occurred while warming the server: check the msh log")
				clientSocket.Write(mes)
				errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

				return
			}

			// msh JOIN response (answer client with text in the loadscreen)
			mes := buildMessage(reqType, "Server start command issued. Please wait... "+servstats.Stats.LoadProgress)
			clientSocket.Write(mes)
			errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

		} else {
			// ms online (un/suspended)

			// issue warm
			logMsh = servctrl.WarmMS()
			if logMsh != nil {
				// msh JOIN response (warn client with text in the loadscreen)
				logMsh.Log(true)
				mes := buildMessage(errco.MESSAGE_FORMAT_TXT, "An error occurred while warming the server: check the msh log")
				clientSocket.Write(mes)
				errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

				return
			}

			// open proxy between client and server
			openProxy(clientSocket, reqPacket, errco.CLIENT_REQ_JOIN)
		}
	}
}

// openProxy opens a proxy connections between mincraft server and mincraft client.
//
// It forwards the server init packet for ms to interpret.
//
// The req parameter indicates what request type (INFO os JOIN) the proxy will be used for.
func openProxy(clientSocket net.Conn, serverInitPacket []byte, req int) {
	// open a connection to ms and connect it with the client
	serverSocket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", config.TargetHost, config.TargetPort))
	if err != nil {
		errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_DIAL, err.Error())

		// msh JOIN response (warn client with text in the loadscreen)
		mes := buildMessage(errco.CLIENT_REQ_JOIN, "can't connect to server... check if minecraft server is running and set the correct targetPort")
		clientSocket.Write(mes)
		errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

		return
	}

	// stopC is used to close serv->client and client->serv at the same time
	stopC := make(chan bool, 1)

	// forward the request packet data
	serverSocket.Write(serverInitPacket)

	// launch proxy client -> server
	go forward(clientSocket, serverSocket, false, stopC, req)

	// launch proxy server -> client
	go forward(serverSocket, clientSocket, true, stopC, req)
}

// forward takes a source and a destination net.Conn and forwards them.
//
// isServerToClient used to know the forward direction
//
// req indicates if connection should be counted in servstats.Stats.ConnCount
//
// [goroutine]
func forward(source, destination net.Conn, isServerToClient bool, stopC chan bool, req int) {
	data := make([]byte, 1024)

	// if client has requested ms join, change connection count
	if isServerToClient && req == errco.CLIENT_REQ_JOIN { // isServerToClient used to count in only one of the 2 forward func
		servstats.Stats.ConnCount++
		errco.NewLogln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "A CLIENT CONNECTED TO THE SERVER! (join req) - %d active connections", servstats.Stats.ConnCount)

		defer func() {
			servstats.Stats.ConnCount--
			errco.NewLogln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "A CLIENT DISCONNECTED FROM THE SERVER! (join req) - %d active connections", servstats.Stats.ConnCount)

			servctrl.FreezeMSSchedule()
		}()
	}

	for {
		// if client or server disconnect, msh should close the connection with server or client.
		// otherwise client/server (being connected to msh) thinks the connection is still alive and reaches a timeout.
		select {
		case <-stopC:
			errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_CONN_EOF, "closing %15s --> %15s (cause: %s, server to client: %t)", strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0], "stop channel", isServerToClient)
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
				errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_CONN_EOF, "closing %15s --> %15s (cause: %s, server to client: %t)", strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0], err.Error(), isServerToClient)
			} else {
				errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_READ, "closing %15s --> %15s (cause: %s, server to client: %t)", strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0], err.Error(), isServerToClient)
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
				servstats.Stats.BytesToClients += float64(dataLen)
				errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%sserver --> client%s: %v", errco.COLOR_BLUE, errco.COLOR_RESET, data[:dataLen])
			} else {
				servstats.Stats.BytesToServer += float64(dataLen)
				errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%sclient --> server%s: %v", errco.COLOR_GREEN, errco.COLOR_RESET, data[:dataLen])
			}
			servstats.Stats.M.Unlock()
		}
	}
}
