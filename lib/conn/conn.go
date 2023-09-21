package conn

import (
	"fmt"
	"net"
	"strings"
	"time"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/servctrl"
	"msh/lib/servstats"
)

func init() {
	go printDataUsage()
}

// HandlerClientConn handles a client that is connecting.
// Can handle a client that is requesting server INFO or server JOIN.
// If there is a ms major error, it is reported to client then func returns.
// [goroutine]
func HandlerClientConn(clientConn net.Conn) {
	// handling of ipv6 addresses
	li := strings.LastIndex(clientConn.RemoteAddr().String(), ":")
	clientAddress := clientConn.RemoteAddr().String()[:li]

	// get request type from client
	reqPacket, reqType, logMsh := getReqType(clientConn)
	if logMsh != nil {
		logMsh.Log(true)
		return
	}

	// if there is a major error warn the client and return
	if servstats.Stats.MajorError != nil {
		errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_MINECRAFT_SERVER, "a client connected to msh (%s:%d to %s:%d) but minecraft server has encountered major problems", clientAddress, config.MshPort, config.ServHost, config.ServPort)

		// close the client connection before returning
		defer func() {
			errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "closing connection for: %s", clientAddress)
			clientConn.Close()
		}()

		// msh INFO/JOIN response (warn client with error description)
		mes := buildMessage(reqType, fmt.Sprintf(servstats.Stats.MajorError.Mex, servstats.Stats.MajorError.Arg...))
		clientConn.Write(mes)
		errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

		// msh PING response if it was a client INFO request
		if reqType == errco.CLIENT_REQ_INFO {
			logMsh = getPing(clientConn)
			if logMsh != nil {
				logMsh.Log(true)
			}
		}

		return
	}

	// handle the request depending on request type
	switch reqType {
	case errco.CLIENT_REQ_INFO:
		errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "a client requested server info from %s:%d to %s:%d", clientAddress, config.MshPort, config.ServHost, config.ServPort)

		if servstats.Stats.Status != errco.SERVER_STATUS_ONLINE || servstats.Stats.Suspended {
			// ms not online or suspended

			defer func() {
				// close the client connection before returning
				errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "closing connection for: %s", clientAddress)
				clientConn.Close()
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
			clientConn.Write(mes)
			errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

			// msh PING response
			logMsh := getPing(clientConn)
			if logMsh != nil {
				logMsh.Log(true)
				return
			}

		} else {
			// ms online and not suspended

			// open proxy between client and server
			openProxy(clientConn, reqPacket, errco.CLIENT_REQ_INFO)
		}

	case errco.CLIENT_REQ_JOIN:
		errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "a client tried to join from %s:%d to %s:%d", clientAddress, config.MshPort, config.ServHost, config.ServPort)

		if servstats.Stats.Status != errco.SERVER_STATUS_ONLINE {
			// ms not online (un/suspended)

			defer func() {
				// close the client connection before returning
				errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "closing connection for: %s", clientAddress)
				clientConn.Close()
			}()

			// check if the request packet contains element of whitelist or the address is in whitelist
			logMsh := config.ConfigRuntime.IsWhitelist(reqPacket, clientAddress)
			if logMsh != nil {
				logMsh.Log(true)

				// msh JOIN response (warn client with text in the loadscreen)
				mes := buildMessage(reqType, "You don't have permission to warm this server")
				clientConn.Write(mes)
				errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

				return
			}

			// issue warm
			logMsh = servctrl.WarmMS()
			if logMsh != nil {
				// msh JOIN response (warn client with text in the loadscreen)
				logMsh.Log(true)
				mes := buildMessage(reqType, "An error occurred while warming the server: check the msh log")
				clientConn.Write(mes)
				errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

				return
			}

			// msh JOIN response (answer client with text in the loadscreen)
			mes := buildMessage(reqType, "Server start command issued. Please wait... "+servstats.Stats.LoadProgress)
			clientConn.Write(mes)
			errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

		} else {
			// ms online (un/suspended)

			// issue warm
			logMsh = servctrl.WarmMS()
			if logMsh != nil {
				// msh JOIN response (warn client with text in the loadscreen)
				logMsh.Log(true)
				mes := buildMessage(reqType, "An error occurred while warming the server: check the msh log")
				clientConn.Write(mes)
				errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

				return
			}

			// open proxy between client and server
			openProxy(clientConn, reqPacket, errco.CLIENT_REQ_JOIN)
		}

	default:
		mes := buildMessage(reqType, "Client request unknown")
		clientConn.Write(mes)
		errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)
	}
}

// openProxy opens a proxy connections between mincraft server and mincraft client.
//
// It sends the request packet for ms to interpret.
//
// The req parameter indicates what request type (INFO os JOIN) the proxy will be used for.
func openProxy(clientConn net.Conn, serverInitPacket []byte, req int) {
	// open a connection to ms and connect it with the client
	serverSocket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", config.ServHost, config.ServPort))
	if err != nil {
		errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_DIAL, err.Error())

		// msh JOIN response (warn client with text in the loadscreen)
		mes := buildMessage(errco.CLIENT_REQ_JOIN, "can't connect to server... check if minecraft server is running and set the correct ServPort")
		clientConn.Write(mes)
		errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

		return
	}

	// sends the request packet
	serverSocket.Write(serverInitPacket)

	// launch proxy client -> server
	go forwardTCP(clientConn, serverSocket, false, req)

	// launch proxy server -> client
	go forwardTCP(serverSocket, clientConn, true, req)
}

// forwardTCP takes a source and a destination net.Conn and forwards them.
//
// isServerToClient used to know the forwardTCP direction
//
// req is used to decide if connection should be counted in servstats.Stats.ConnCount
//
// [goroutine]
func forwardTCP(source, destination net.Conn, isServerToClient bool, req int) {
	var data []byte = make([]byte, 1024)
	var direction string

	if isServerToClient {
		direction = "server --> client"
	} else {
		direction = "client --> server"
	}

	// if client has requested ms join, change connection count
	if isServerToClient && req == errco.CLIENT_REQ_JOIN { // isServerToClient used to count in only one of the 2 forwardTCP()
		servstats.Stats.ConnCount++
		errco.NewLogln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "A CLIENT CONNECTED TO THE SERVER! (join req) - %d active connections", servstats.Stats.ConnCount)

		defer func() {
			servstats.Stats.ConnCount--
			errco.NewLogln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "A CLIENT DISCONNECTED FROM THE SERVER! (join req) - %d active connections", servstats.Stats.ConnCount)

			servctrl.FreezeMSSchedule()
		}()
	}

	for {
		// update read and write timeout
		source.SetReadDeadline(time.Now().Add(60 * time.Second))
		destination.SetWriteDeadline(time.Now().Add(60 * time.Second))

		// read data from source
		dataLen, err := source.Read(data)
		if err != nil {
			errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_CONN_EOF, "closing %15s --> %15s | %s (cause: %s)", strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0], direction, err.Error())

			// close the source/destination connections
			_ = destination.Close()
			_ = source.Close()
			return
		}

		// write data to destination
		_, err = destination.Write(data[:dataLen])
		if err != nil {
			errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_CONN_WRITE, "closing %15s --> %15s | %s (cause: %s)", strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0], direction, err.Error())

			// close the source/destination connections
			_ = destination.Close()
			_ = source.Close()
			return
		}

		// calculate bytes/s to client/server
		if config.ConfigRuntime.Msh.ShowInternetUsage && errco.DebugLvl >= errco.LVL_3 {
			errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%s%s%s: %v", errco.COLOR_PURPLE, direction, errco.COLOR_RESET, data[:dataLen])

			servstats.Stats.M.Lock()
			if isServerToClient {
				servstats.Stats.BytesToClients += float64(dataLen)
			} else {
				servstats.Stats.BytesToServer += float64(dataLen)
			}
			servstats.Stats.M.Unlock()
		}
	}
}

// printDataUsage prints connection data (KB/s) to clients and to minecraft server.
//
// Prints data exchanged only when clients are connected to ms.
//
// Logging is disabled when ShowInternetUsage is false.
//
// [goroutine]
func printDataUsage() {
	ticker := time.NewTicker(time.Second)
	for {
		<-ticker.C

		if !config.ConfigRuntime.Msh.ShowInternetUsage {
			continue
		}

		if servstats.Stats.BytesToClients != 0 || servstats.Stats.BytesToServer != 0 {
			errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "data/s: %8.3f KB/s to clients | %8.3f KB/s to server", servstats.Stats.BytesToClients/1024, servstats.Stats.BytesToServer/1024)
			servstats.Stats.M.Lock()
			servstats.Stats.BytesToClients = 0
			servstats.Stats.BytesToServer = 0
			servstats.Stats.M.Unlock()
		}
	}
}
