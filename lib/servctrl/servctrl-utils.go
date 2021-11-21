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
// The /list command is used for safety and if it fails, internal player count is returned.
// No error is returned: the integer is always meaningful
// (might be more or less reliable depending from where it retrieved).
// The method used to count players is returned as second parameter.
func countPlayerSafe() (int, string) {
	errco.Logln(errco.LVL_B, "retrieving  player count...")

	playerCount, errMsh := getPlayersByServInfo()
	if errMsh == nil {
		return playerCount, "server info"
	}
	errco.LogMshErr(errMsh.AddTrace("countPlayerSafe"))

	playerCount, errMsh = getPlayersByListCom()
	if errMsh == nil {
		return playerCount, "list command"
	}
	errco.LogMshErr(errMsh.AddTrace("countPlayerSafe"))

	return servstats.Stats.PlayerCount, "internal"
}

// getPlayersByListCom returns the number of players using "list" command
func getPlayersByListCom() (int, *errco.Error) {
	outStr, errMsh := Execute("list", "getPlayersByListCom")
	if errMsh != nil {
		return 0, errMsh.AddTrace("getPlayersByListCom")
	}
	playersStr, errMsh := utility.StrBetween(outStr, "There are ", " of a max")
	if errMsh != nil {
		return 0, errMsh.AddTrace("getPlayersByListCom")
	}
	players, err := strconv.Atoi(playersStr)
	if err != nil {
		return 0, errco.NewErr(errco.ERROR_CONVERSION, errco.LVL_D, "getPlayersByListCom", err.Error())
	}

	return players, nil
}

// getPlayersByServInfo returns the number of players using server info request
func getPlayersByServInfo() (int, *errco.Error) {
	servInfo, errMsh := getServInfo()
	if errMsh != nil {
		return -1, errMsh.AddTrace("getPlayersByServInfo")
	}

	return servInfo.Players.Online, nil
}

// getServInfo returns server info after emulating a server info request to the minecraft server
func getServInfo() (*model.DataInfo, *errco.Error) {
	if servstats.Stats.Status != errco.SERVER_STATUS_ONLINE {
		return &model.DataInfo{}, errco.NewErr(errco.ERROR_SERVER_NOT_ONLINE, errco.LVL_D, "getServInfo", "")
	}

	// open connection to minecraft server
	serverSocket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", config.TargetHost, config.TargetPort))
	if err != nil {
		return nil, errco.NewErr(errco.ERROR_SERVER_DIAL, errco.LVL_D, "getServInfo", err.Error())
	}
	defer serverSocket.Close()

	// timeout can be low since its a connection to 127.0.0.1
	serverSocket.SetDeadline(time.Now().Add(100 * time.Millisecond))

	// building byte array to request minecraft server info
	// [16 0 244 5 9 49 50 55 46 48 46 48 46 49 99 211 1    ]
	//                                          └port┘ └info
	reqInfoMessage := bytes.NewBuffer([]byte{16, 0, 244, 5, 9, 49, 50, 55, 46, 48, 46, 48, 46, 49})
	reqInfoMessage.Write(big.NewInt(int64(config.ListenPort)).Bytes())
	reqInfoMessage.Write([]byte{1, 1, 0})

	serverSocket.Write(reqInfoMessage.Bytes())

	// read response from server
	recInfoData := []byte{}
	buf := make([]byte, 1024)
	for {
		n, err := serverSocket.Read(buf)
		if err != nil {
			// cannot break on io.EOF since it's not sent, so break happens on timeout
			// using io.EOF would be better
			if err, ok := err.(net.Error); ok && err.Timeout() {
				break
			}
			return &model.DataInfo{}, errco.NewErr(errco.ERROR_SERVER_REQUEST_INFO, errco.LVL_D, "getServInfo", err.Error())
		}

		recInfoData = append(recInfoData, buf[:n]...)
	}

	// remove first 5 bytes that are used as header to get only the json data
	// [178 88 0 175 88]{"description":{ ...
	recInfoData = recInfoData[5:]

	recInfo := &model.DataInfo{}
	err = json.Unmarshal(recInfoData, recInfo)
	if err != nil {
		return &model.DataInfo{}, errco.NewErr(errco.ERROR_JSON_UNMARSHAL, errco.LVL_D, "getServInfo", err.Error())
	}

	// update server version and protocol in config
	if recInfo.Version.Name != config.ConfigRuntime.Server.Version || recInfo.Version.Protocol != config.ConfigRuntime.Server.Protocol {
		errco.Logln(errco.LVL_D, "server version found! serverVersion: %s serverProtocol: %d", recInfo.Version.Name, recInfo.Version.Protocol)

		// update the runtime config
		config.ConfigRuntime.Server.Version = recInfo.Version.Name
		config.ConfigRuntime.Server.Protocol = recInfo.Version.Protocol

		// update the file config
		config.ConfigDefault.Server.Version = recInfo.Version.Name
		config.ConfigDefault.Server.Protocol = recInfo.Version.Protocol

		errMsh := config.SaveConfigDefault()
		if errMsh != nil {
			return nil, errMsh.AddTrace("getServInfo")
		}
	}

	return recInfo, nil
}
