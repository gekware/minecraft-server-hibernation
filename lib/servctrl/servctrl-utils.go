package servctrl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/model"
	"msh/lib/servstats"
)

// countPlayerSafe returns the number of players on the server.
//
// Players are retrived by (in order): server info, list command, internal connection count.
//
// Internal connection count is reset if a more reliable method is used.
//
// no error is returned: the return integer is always meaningful
// (might be more or less reliable depending from where it retrieved).
func countPlayerSafe() int {
	var logMsh *errco.MshLog
	var playerCount int
	var method string

	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "retrieving player count...")

	if playerCount, logMsh = getPlayersByServInfo(); logMsh.Log(true) == nil {
		method = "server info"
		if playerCount != servstats.Stats.ConnCount {
			errco.NewLogln(errco.TYPE_WAR, errco.LVL_1, errco.ERROR_WRONG_CONNECTION_COUNT, "connection count (%d) different from %s player count (%d)", servstats.Stats.ConnCount, method, playerCount)
		}

	} else if playerCount, logMsh = getPlayersByListCom(); logMsh.Log(true) == nil {
		method = "list command"
		if playerCount != servstats.Stats.ConnCount {
			errco.NewLogln(errco.TYPE_WAR, errco.LVL_1, errco.ERROR_WRONG_CONNECTION_COUNT, "connection count (%d) different from %s player count (%d)", servstats.Stats.ConnCount, method, playerCount)
		}

	} else {
		method = "connection count"
		playerCount = servstats.Stats.ConnCount
	}

	errco.NewLogln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "%d online players - method for player count: %s", playerCount, method)

	return playerCount
}

// getPlayersByListCom returns the number of players using "list" command
func getPlayersByListCom() (int, *errco.MshLog) {
	output, logMsh := Execute("list")
	if logMsh != nil {
		return 0, logMsh.AddTrace()
	}

	// return if output has unexpected format
	if !strings.Contains(output, "INFO]:") {
		return 0, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_UNEXP_OUTPUT, "string does not contain \"INFO]:\"")
	}

	// check test function for possible `list` outputs
	firstNumber := regexp.MustCompile(`\d+`).FindString(strings.Split(output, "INFO]:")[1])

	// check if firstNumber has been found
	if firstNumber == "" {
		return 0, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_UNEXP_OUTPUT, "firstNumber string is empty")
	}

	players, err := strconv.Atoi(firstNumber)
	if err != nil {
		return 0, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONVERSION, err.Error())
	}

	return players, nil
}

// getPlayersByServInfo returns the number of players using server info request
func getPlayersByServInfo() (int, *errco.MshLog) {
	servInfo, logMsh := getServInfo()
	if logMsh != nil {
		return -1, logMsh.AddTrace()
	}

	return servInfo.Players.Online, nil
}

// getServInfo returns server info after emulating a server info request to the minecraft server
func getServInfo() (*model.DataInfo, *errco.MshLog) {
	var recInfoData []byte = []byte{}
	var recInfo *model.DataInfo = &model.DataInfo{}
	var buf []byte = make([]byte, 1024)

	// check if ms is warm and interactable
	logMsh := CheckMSWarm()
	if logMsh != nil {
		return nil, logMsh.AddTrace()
	}

	// open connection to minecraft server
	serverSocket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", config.ServHost, config.ServPort))
	if err != nil {
		return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_DIAL, err.Error())
	}
	defer serverSocket.Close()

	// building byte array to request minecraft server info
	// [16 0 244 5 9 49 50 55 46 48 46 48 46 49 99 211 1 1 0 ]
	//                                          └port┘ └info┘
	reqInfoMessage := bytes.NewBuffer([]byte{16, 0, 244, 5, 9, 49, 50, 55, 46, 48, 46, 48, 46, 49})
	reqInfoMessage.Write(big.NewInt(int64(config.MshPort)).Bytes())
	reqInfoMessage.Write([]byte{1, 1, 0})

	mes := reqInfoMessage.Bytes()
	serverSocket.Write(mes)
	errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> server%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

	// read response from server
	for {
		// timeout can be low since its a connection to 127.0.0.1
		// the first time the ms info are requested it timeout is <100 mills
		// (probably the ms function that handles ms info needs time to load the first time it's called)
		serverSocket.SetReadDeadline(time.Now().Add(200 * time.Millisecond))

		dataLen, err := serverSocket.Read(buf)
		if err != nil {
			// cannot break on io.EOF since it's not sent, so break happens on timeout
			// using io.EOF would be better
			if err, ok := err.(net.Error); ok && err.Timeout() {
				break
			}

			return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_REQUEST_INFO, err.Error())
		}

		errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%sserver --> msh%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, buf[:dataLen])

		recInfoData = append(recInfoData, buf[:dataLen]...)
	}

	// remove first 5 bytes that are used as header to get only the json data
	// [178 88 0 175 88]{"description":{ ...
	if len(recInfoData) < 5 {
		return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_REQUEST_INFO, "not enough data received (%v)", recInfoData)
	}
	recInfoData = recInfoData[5:]

	// load data into struct
	err = json.Unmarshal(recInfoData, recInfo)
	if err != nil {
		return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_JSON_UNMARSHAL, err.Error())
	}

	// update server version and protocol in config
	if recInfo.Version.Name != config.ConfigRuntime.Server.Version || recInfo.Version.Protocol != config.ConfigRuntime.Server.Protocol {
		errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "server version found! serverVersion: %s serverProtocol: %d", recInfo.Version.Name, recInfo.Version.Protocol)

		// update runtime config if version is not specified
		if config.ConfigRuntime.Server.Version == "" {
			config.ConfigRuntime.Server.Version = recInfo.Version.Name
			config.ConfigRuntime.Server.Protocol = recInfo.Version.Protocol
		}

		// update and save default config
		config.ConfigDefault.Server.Version = recInfo.Version.Name
		config.ConfigDefault.Server.Protocol = recInfo.Version.Protocol
		logMsh := config.ConfigDefault.Save()
		if logMsh != nil {
			return nil, logMsh.AddTrace()
		}
	}

	return recInfo, nil
}
