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
	errco.Logln(errco.LVL_1, "retrieving  player count...")

	playerCount, errMsh := getPlayersByServInfo()
	if errMsh == nil {
		return playerCount, "server info"
	}
	errco.LogWarn(errMsh.AddTrace("countPlayerSafe"))

	playerCount, errMsh = getPlayersByListCom()
	if errMsh == nil {
		return playerCount, "list command"
	}
	errco.LogWarn(errMsh.AddTrace("countPlayerSafe"))

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
		return 0, errco.NewErr(errco.ERROR_CONVERSION, errco.LVL_3, "getPlayersByListCom", err.Error())
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
		return &model.DataInfo{}, errco.NewErr(errco.ERROR_SERVER_NOT_ONLINE, errco.LVL_3, "getServInfo", "")
	}

	// open connection to minecraft server
	serverSocket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", config.TargetHost, config.TargetPort))
	if err != nil {
		return &model.DataInfo{}, errco.NewErr(errco.ERROR_SERVER_DIAL, errco.LVL_3, "getServInfo", err.Error())
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
	errco.Logln(errco.LVL_4, "%smsh --> server%s:%v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

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
			return &model.DataInfo{}, errco.NewErr(errco.ERROR_SERVER_REQUEST_INFO, errco.LVL_3, "getServInfo", err.Error())
		}

		errco.Logln(errco.LVL_4, "%sserver --> msh%s:%v", errco.COLOR_PURPLE, errco.COLOR_RESET, buf[:dataLen])

		recInfoData = append(recInfoData, buf[:dataLen]...)
	}

	// remove first 5 bytes that are used as header to get only the json data
	// [178 88 0 175 88]{"description":{ ...
	if len(recInfoData) < 5 {
		return &model.DataInfo{}, errco.NewErr(errco.ERROR_SERVER_REQUEST_INFO, errco.LVL_3, "getServInfo", "received data unexpected format")
	}
	recInfoData = recInfoData[5:]

	recInfo := &model.DataInfo{}
	err = json.Unmarshal(recInfoData, recInfo)
	if err != nil {
		return &model.DataInfo{}, errco.NewErr(errco.ERROR_JSON_UNMARSHAL, errco.LVL_3, "getServInfo", err.Error())
	}

	// update server version and protocol in config
	if recInfo.Version.Name != config.ConfigRuntime.Server.Version || recInfo.Version.Protocol != config.ConfigRuntime.Server.Protocol {
		errco.Logln(errco.LVL_3, "server version found! serverVersion: %s serverProtocol: %d", recInfo.Version.Name, recInfo.Version.Protocol)

		// update runtime config if version is not specified
		if config.ConfigRuntime.Server.Version == "" {
			config.ConfigRuntime.Server.Version = recInfo.Version.Name
			config.ConfigRuntime.Server.Protocol = recInfo.Version.Protocol
		}

		// update and save default config
		config.ConfigDefault.Server.Version = recInfo.Version.Name
		config.ConfigDefault.Server.Protocol = recInfo.Version.Protocol
		errMsh := config.ConfigDefault.Save()
		if errMsh != nil {
			return nil, errMsh.AddTrace("getServInfo")
		}
	}

	return recInfo, nil
}
