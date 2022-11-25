package servctrl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"time"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/model"
	"msh/lib/servstats"
	"msh/lib/utility"
)

// countPlayerSafe returns the number of players on the server.
//
// players are retrived by (in order): server info, list command, internal player count.
//
// no error is returned: the return integer is always meaningful
// (might be more or less reliable depending from where it retrieved).
func countPlayerSafe() int {
	var logMsh *errco.MshLog
	var playerCount int
	var method string

	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "retrieving player count...")

	if playerCount, logMsh = getPlayersByServInfo(); logMsh.Log() == nil {
		method = "list command"
	} else if playerCount, logMsh = getPlayersByListCom(); logMsh.Log() == nil {
		method = "server info"
	} else {
		playerCount = servstats.Stats.PlayerCount
		method = "internal"
	}

	errco.NewLogln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "%d online players - method for player count: %s", playerCount, method)

	return playerCount
}

// getPlayersByListCom returns the number of players using "list" command
func getPlayersByListCom() (int, *errco.MshLog) {
	outStr, logMsh := Execute("list", "getPlayersByListCom")
	if logMsh != nil {
		return 0, logMsh.AddTrace()
	}
	playersStr, logMsh := utility.StrBetween(outStr, "There are ", " of a max")
	if logMsh != nil {
		return 0, logMsh.AddTrace()
	}
	players, err := strconv.Atoi(playersStr)
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
	// check if ms is running
	logMsh := checkMSRunning()
	if logMsh != nil {
		return &model.DataInfo{}, logMsh.AddTrace()
	}

	// open connection to minecraft server
	serverSocket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", config.TargetHost, config.TargetPort))
	if err != nil {
		return &model.DataInfo{}, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_DIAL, err.Error())
	}
	defer serverSocket.Close()

	// timeout can be low since its a connection to 127.0.0.1
	serverSocket.SetDeadline(time.Now().Add(100 * time.Millisecond))

	// building byte array to request minecraft server info
	// [16 0 244 5 9 49 50 55 46 48 46 48 46 49 99 211 1 1 0 ]
	//                                          └port┘ └info┘
	reqInfoMessage := bytes.NewBuffer([]byte{16, 0, 244, 5, 9, 49, 50, 55, 46, 48, 46, 48, 46, 49})
	reqInfoMessage.Write(big.NewInt(int64(config.ListenPort)).Bytes())
	reqInfoMessage.Write([]byte{1, 1, 0})

	mes := reqInfoMessage.Bytes()
	serverSocket.Write(mes)
	errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> server%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

	// read response from server
	recInfoData := []byte{}
	buf := make([]byte, 1024)
	for {
		dataLen, err := serverSocket.Read(buf)
		if err != nil {
			// cannot break on io.EOF since it's not sent, so break happens on timeout
			// using io.EOF would be better
			if err, ok := err.(net.Error); ok && err.Timeout() {
				break
			}
			return &model.DataInfo{}, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_REQUEST_INFO, err.Error())
		}

		errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%sserver --> msh%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, buf[:dataLen])

		recInfoData = append(recInfoData, buf[:dataLen]...)
	}

	// remove first 5 bytes that are used as header to get only the json data
	// [178 88 0 175 88]{"description":{ ...
	if len(recInfoData) < 5 {
		return &model.DataInfo{}, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_REQUEST_INFO, "received data unexpected format (%v)", recInfoData)
	}
	recInfoData = recInfoData[5:]

	recInfo := &model.DataInfo{}
	err = json.Unmarshal(recInfoData, recInfo)
	if err != nil {
		return &model.DataInfo{}, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_JSON_UNMARSHAL, err.Error())
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
