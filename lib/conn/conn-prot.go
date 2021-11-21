package conn

import (
	"bytes"
	"encoding/json"
	"math/big"
	"net"
	"strings"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/model"
)

// buildMessage takes the message format (TXT/INFO) and a message to write to the client
func buildMessage(messageFormat int, message string) []byte {
	// mountHeader mounts the full header to a specified message
	var mountHeader = func(data []byte) []byte {
		//                  ┌--------------------full header--------------------┐
		// scheme:          [ sub-header1     | sub-header2 | sub-header3       | message   ]
		// bytes used:      [ 2               | 1           | 2                 | 0 - 16379 ]
		// value range:     [ 128 0 - 255 127 | 0           | 128 0 - 255 127	| --------- ]

		// addSubHeader mounts 1 sub-header to a specified message
		var addSubHeader = func(message []byte) []byte {
			//              ┌------sub-header1/3------┐
			// scheme:      [ firstByte | secondByte  | data ]
			// value range: [ 128 - 255 | 0 - 127     | ---- ]
			// it's a number composed of 2 digits in base-128 (firstByte is least significant byte)
			// sub-header represents the length of the following data

			firstByte := len(message)%128 + 128
			secondByte := float64(len(message) / 128)
			return append([]byte{byte(firstByte), byte(secondByte)}, message...)
		}

		// sub-header3 calculation
		data = addSubHeader(data)

		// sub-header2 calculation
		data = append([]byte{0}, data...)

		// sub-header1 calculation
		data = addSubHeader(data)

		return data
	}

	switch messageFormat {
	case errco.MESSAGE_FORMAT_TXT:
		// send text to be shown in the loadscreen

		messageStruct := &model.DataTxt{}
		messageStruct.Text = message

		dataTxtJSON, err := json.Marshal(messageStruct)
		if err != nil {
			// don't return error, just log it
			errco.LogMshErr(errco.NewErr(errco.ERROR_JSON_MARSHAL, errco.LVL_D, "buildMessage", err.Error()))
			return nil
		}

		return mountHeader(dataTxtJSON)

	case errco.MESSAGE_FORMAT_INFO:
		// send server info

		// "&" [\x26] is converted to "§" [\xc2\xa7]
		// this step is not strictly necessary if in msh-config is used the character "§"
		message = strings.ReplaceAll(message, "&", "§")

		messageStruct := &model.DataInfo{}
		messageStruct.Description.Text = message
		messageStruct.Players.Max = 0
		messageStruct.Players.Online = 0
		messageStruct.Version.Name = config.ConfigRuntime.Server.Version
		messageStruct.Version.Protocol = config.ConfigRuntime.Server.Protocol
		messageStruct.Favicon = "data:image/png;base64," + config.ServerIcon

		dataInfJSON, err := json.Marshal(messageStruct)
		if err != nil {
			// don't return error, just log it
			errco.LogMshErr(errco.NewErr(errco.ERROR_JSON_MARSHAL, errco.LVL_D, "buildMessage", err.Error()))
			return nil
		}

		return mountHeader(dataInfJSON)

	default:
		return nil
	}
}

// getReqType returns the request type (INFO or JOIN) and playerName of the client
func getReqType(clientSocket net.Conn) (int, string, *errco.Error) {
	reqPacket, errMsh := getClientPacket(clientSocket)
	if errMsh != nil {
		return errco.ERROR_CLIENT_REQ, "", errMsh.AddTrace("getReqType")
	}

	// generate flags
	listenPortByt := big.NewInt(int64(config.ListenPort)).Bytes() // calculates listen port in BigEndian bytes
	reqFlagInfo := append(listenPortByt, byte(1))                 // flag contained in INFO request packet -> [99 211 1]
	reqFlagJoin := append(listenPortByt, byte(2))                 // flag contained in JOIN request packet -> [99 211 2]

	playerName := extractPlayerName(reqPacket, reqFlagJoin, clientSocket)

	switch {
	case bytes.Contains(reqPacket, reqFlagInfo):
		// client is requesting server info and ping
		// client first packet:	[ ... x x x (listenPortBytes) 1 1 0] or [ ... x x x (listenPortBytes) 1 ]
		//                      [           ^---reqFlagInfo---^    ]    [           ^---reqFlagInfo---^ ]
		return errco.CLIENT_REQ_INFO, playerName, nil

	case bytes.Contains(reqPacket, reqFlagJoin):
		// client is trying to join the server
		// client first packet:	[ ... x x x (listenPortBytes) 2 ] or [ ... x x x (listenPortBytes) 2 x x x (player name) ]
		//                      [           ^---reqFlagJoin---^ ]    [           ^---reqFlagJoin---^                     ]
		return errco.CLIENT_REQ_JOIN, playerName, nil

	default:
		return errco.CLIENT_REQ_UNKN, "", errco.NewErr(errco.CLIENT_REQ_UNKN, errco.LVL_D, "getReqType", "client request unknown")
	}
}

// getPing responds to the ping request
func getPing(clientSocket net.Conn) *errco.Error {
	// read the first packet
	pingData, errMsh := getClientPacket(clientSocket)
	if errMsh != nil {
		return errMsh.AddTrace("getPing [1]")
	}

	switch {
	case bytes.Equal(pingData, []byte{1, 0}):
		// packet is [1 0]
		// read the second packet
		pingData, errMsh = getClientPacket(clientSocket)
		if errMsh != nil {
			return errMsh.AddTrace("getPing [2]")
		}

	case bytes.Equal(pingData[:2], []byte{1, 0}):
		// packet is [1 0 9 1 0 0 0 0 0 89 73 114]
		// remove first 2 bytes: [1 0 9 1 0 0 0 0 0 89 73 114] -> [9 1 0 0 0 0 0 89 73 114]
		pingData = pingData[2:]
	}

	// answer ping
	clientSocket.Write(pingData)

	return nil
}

// getClientPacket reads the client socket and returns only the bytes containing data
func getClientPacket(clientSocket net.Conn) ([]byte, *errco.Error) {
	buf := make([]byte, 1024)

	// read first packet
	dataLen, err := clientSocket.Read(buf)
	if err != nil {
		return nil, errco.NewErr(errco.ERROR_CLIENT_SOCKET_READ, errco.LVL_D, "getClientPacket", "error during clientSocket.Read()")
	}

	return buf[:dataLen], nil
}

// extractPlayerName retrieves the name of the player that is trying to connect.
// "player unknown" is returned in case of error or reqFlagJoin not found
func extractPlayerName(data, reqFlagJoin []byte, clientSocket net.Conn) string {
	// player name is found only in join request packet
	if !bytes.Contains(data, reqFlagJoin) {
		// reqFlagJoin not found
		return "player unknown"
	}

	dataSplAft := bytes.SplitAfter(data, reqFlagJoin)

	if len(dataSplAft[1]) > 0 {
		// packet join request and player name:
		// [ ... x x x (listenPortBytes) 2 x x x (player name) ]
		// [ ^---data----------------------------------------^ ]
		// [           ^---reqFlagJoin---^ ^--dataSplAft[1]--^ ]

		return string(dataSplAft[1][3:])

	} else {
		// packet join request:
		// (to get player name, a further packet read is required)
		// [ ... x x x (listenPortBytes) 2 ] [ x x x (player name) ]
		// [ ^---data--------------------^ ] [       ^-data[3:]--^ ]
		// [              dataSplitAft[1]-╝] [                     ]

		data, errMsh := getClientPacket(clientSocket)
		if errMsh != nil {
			// this error is non-blocking: log the error and return "player unknown"
			errco.LogMshErr(errMsh.AddTrace("extractPlayerName"))
			return "player unknown"
		}

		return string(data[3:])
	}
}
