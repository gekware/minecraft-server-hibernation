package servconn

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"

	"msh/lib/config"
	"msh/lib/logger"
)

const (
	CLIENT_REQ_ERROR = 0x0001ffff // error while analyzing client request
	CLIENT_REQ_UNKN  = 0x00010000 // client request unknown
	CLIENT_REQ_INFO  = 0x00010001 // client request server info
	CLIENT_REQ_JOIN  = 0x00010002 // client request server join

	MESSAGE_FORMAT_TXT  = 0x00020001 // message to client should be built as TXT
	MESSAGE_FORMAT_INFO = 0x00020002 // message to client should be built as INFO
)

// buildMessage takes the message format (TXT/INFO) and a message to write to the client
func buildMessage(messageFormat int, message string) []byte {
	// mountHeader mounts the full header to a specified message
	var mountHeader = func(messageStr string) []byte {
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

		messageByte := []byte(messageStr)

		// sub-header3 calculation
		messageByte = addSubHeader(messageByte)

		// sub-header2 calculation
		messageByte = append([]byte{0}, messageByte...)

		// sub-header1 calculation
		messageByte = addSubHeader(messageByte)

		return messageByte
	}

	switch messageFormat {
	case MESSAGE_FORMAT_TXT:
		// send text to be shown in the loadscreen

		messageJSON := fmt.Sprint(
			"{",
			"\"text\":\"", message, "\"",
			"}",
		)

		return mountHeader(messageJSON)

	case MESSAGE_FORMAT_INFO:
		// send server info

		// "\n" should be encoded as "\xc2\xa7r\\n"
		// "&"  should be encoded as "\xc2\xa7"
		message = strings.ReplaceAll(strings.ReplaceAll(message, "\n", "&r\\n"), "&", "\xc2\xa7")

		messageJSON := fmt.Sprint("{",
			"\"description\":{\"text\":\"", message, "\"},",
			"\"players\":{\"max\":0,\"online\":0},",
			"\"version\":{\"name\":\"", config.ConfigRuntime.Server.Version, "\",\"protocol\":", fmt.Sprint(config.ConfigRuntime.Server.Protocol), "},",
			"\"favicon\":\"data:image/png;base64,", config.ServerIcon, "\"",
			"}",
		)

		return mountHeader(messageJSON)

	default:
		return nil
	}
}

// buildReqFlag generates the INFO flag and JOIN flag using the msh port
func buildReqFlag(mshPort string) ([]byte, []byte) {
	// calculates listen port in BigEndian bytes

	listenPortUint64, err := strconv.ParseUint(mshPort, 10, 16) // bitSize: 16 -> since it will be converted to Uint16
	if err != nil {
		logger.Logln("buildListenPortBytes: error during ListenPort conversion to uint64")
		return nil, nil
	}

	listenPortUint16 := uint16(listenPortUint64)
	listenPortBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(listenPortBytes, listenPortUint16) // 25555 -> [99 211] / hex[63 D3]

	// generate flags

	reqFlagInfo := append(listenPortBytes, byte(1)) // flag contained in INFO request packet (first packet of client)
	reqFlagJoin := append(listenPortBytes, byte(2)) // flag contained in JOIN request packet (first packet of client)

	return reqFlagInfo, reqFlagJoin
}

// getReqType returns the request type (INFO or JOIN) and playerName of the client
func getReqType(clientSocket net.Conn) (int, string, error) {
	reqPacket, err := getClientPacket(clientSocket)
	if err != nil {
		return CLIENT_REQ_ERROR, "", fmt.Errorf("getReqType: %v", err)
	}

	reqFlagInfo, reqFlagJoin := buildReqFlag(config.ListenPort)
	playerName := extractPlayerName(reqPacket, reqFlagJoin, clientSocket)

	switch {
	case bytes.Contains(reqPacket, reqFlagInfo):
		// client is requesting server info and ping
		// client first packet:	[ ... x x x (listenPortBytes) 1 1 0] or [ ... x x x (listenPortBytes) 1 ]
		//                      [           ^---reqFlagInfo---^    ]    [           ^---reqFlagInfo---^ ]
		return CLIENT_REQ_INFO, playerName, nil

	case bytes.Contains(reqPacket, reqFlagJoin):
		// client is trying to join the server
		// client first packet:	[ ... x x x (listenPortBytes) 2 ] or [ ... x x x (listenPortBytes) 2 x x x (player name) ]
		//                      [           ^---reqFlagJoin---^ ]    [           ^---reqFlagJoin---^                     ]
		return CLIENT_REQ_JOIN, playerName, nil

	default:
		return CLIENT_REQ_UNKN, "", fmt.Errorf("getReqType: client request unknown")
	}
}

// getPing responds to the ping request
func getPing(clientSocket net.Conn) error {
	// read the first packet
	pingData, err := getClientPacket(clientSocket)
	if err != nil {
		return fmt.Errorf("answerPingReq: error while reading [1] ping request: %v", err)
	}

	switch {
	case bytes.Equal(pingData, []byte{1, 0}):
		// packet is [1 0]
		// read the second packet
		pingData, err = getClientPacket(clientSocket)
		if err != nil {
			return fmt.Errorf("answerPingReq: error while reading [2] ping request: %v", err)
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
func getClientPacket(clientSocket net.Conn) ([]byte, error) {
	buf := make([]byte, 1024)

	// read first packet
	dataLen, err := clientSocket.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("readClientPacket: error during clientSocket.Read()")
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

		data, err := getClientPacket(clientSocket)
		if err != nil {
			// this error is non-blocking, just log it
			logger.Logln("extractPlayerName:", err)
			return "player unknown"
		}

		return string(data[3:])
	}
}

// extractVersionProtocol finds the serverVersion and serverProtocol in (data []byte) and writes them in the config file
func extractVersionProtocol(data []byte) error {
	// if data contains "\"version\":{\"name\":\"" and ",\"protocol\":" --> extract the serverVersion and serverProtocol
	if bytes.Contains(data, []byte("\"version\":{\"name\":\"")) && bytes.Contains(data, []byte(",\"protocol\":")) {
		newServerVersion := string(bytes.Split(bytes.Split(data, []byte("\"version\":{\"name\":\""))[1], []byte("\","))[0])
		newServerProtocol := string(bytes.Split(bytes.Split(data, []byte(",\"protocol\":"))[1], []byte("}"))[0])

		// if serverVersion or serverProtocol are different from the ones specified in config file --> update them
		if newServerVersion != config.ConfigRuntime.Server.Version || newServerProtocol != config.ConfigRuntime.Server.Protocol {
			logger.Logln(
				"server version found!",
				"serverVersion:", newServerVersion,
				"serverProtocol:", newServerProtocol,
			)

			// update the runtime config
			config.ConfigRuntime.Server.Version = newServerVersion
			config.ConfigRuntime.Server.Protocol = newServerProtocol

			// update the file config
			config.ConfigDefault.Server.Version = newServerVersion
			config.ConfigDefault.Server.Protocol = newServerProtocol

			err := config.SaveConfigDefault()
			if err != nil {
				return fmt.Errorf("GetVersionProtocol: %v", err)
			}
		}
	}

	return nil
}
