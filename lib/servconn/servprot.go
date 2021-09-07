package servconn

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"

	"msh/lib/config"
	"msh/lib/data"
	"msh/lib/logger"
)

const (
	CLIENT_REQ_ERROR = -1 // error while analyzing client request
	CLIENT_REQ_UNKN  = 0  // client request unknown
	CLIENT_REQ_INFO  = 1  // client request server info
	CLIENT_REQ_JOIN  = 2  // client request server join
)

// buildMessage takes the format ("txt", "info") and a message to write to the client
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
			"\"version\":{\"name\":\"", config.ConfigRuntime.Server.Version, "\",\"protocol\":", fmt.Sprint(config.ConfigRuntime.Server.Protocol), "},",
			"\"favicon\":\"data:image/png;base64,", data.ServerIcon, "\"",
			"}",
		)

		messageHeader = mountHeader(messageJSON)
	}

	return messageHeader
}

// buildListenPortBytes calculates listen port in BigEndian bytes
func buildListenPortBytes() []byte {
	listenPortUint64, err := strconv.ParseUint(config.ConfigRuntime.Msh.Port, 10, 16) // bitSize: 16 -> since it will be converted to Uint16
	if err != nil {
		logger.Logln("buildListenPortBytes: error during ListenPort conversion to uint64")
		return nil
	}

	listenPortUint16 := uint16(listenPortUint64)
	listenPortBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(listenPortBytes, listenPortUint16) // 25555 ->	[99 211] / hex[63 D3]

	return listenPortBytes
}

// getReqType returns the request type (INFO or JOIN) and playerName (if it's a join request) of the client
func getReqType(clientSocket net.Conn) (int, string, error) {
	listenPortBytes := buildListenPortBytes()

	reqPacket, err := getClientPacket(clientSocket)
	if err != nil {
		return CLIENT_REQ_ERROR, "", fmt.Errorf("getReqType: %v", err)
	}

	playerName, err := extractPlayerName(reqPacket, clientSocket)
	if err != nil {
		// this error is non-blocking, just log it
		logger.Logln("getReqType:", err)
	}

	switch {
	case bytes.Contains(reqPacket, append(listenPortBytes, byte(1))):
		// client is requesting server info and ping
		// client first packet:	[ ... x x x (listenPortBytes) 1 1 0] or [ ... x x x (listenPortBytes) 1 ]
		return CLIENT_REQ_INFO, playerName, nil

	case bytes.Contains(reqPacket, append(listenPortBytes, byte(2))):
		// client is trying to join the server
		// client first packet:	[ ... x x x (listenPortBytes) 2] or [ ... x x x (listenPortBytes) 2 x x x (player name)]
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
// "player unknown" is returned in case of error or [ (listenPortBytes) 2 ] not found
func extractPlayerName(data []byte, clientSocket net.Conn) (string, error) {
	listenPortBytes := buildListenPortBytes()

	if !bytes.Contains(data, append(listenPortBytes, byte(2))) {
		return "player unknown", nil // [ (listenPortBytes) 2 ] where not found, just return player unknown
	}

	dataSplAft := bytes.SplitAfter(data, append(listenPortBytes, byte(2)))

	if len(dataSplAft[1]) > 0 {
		// packet join request and player name:
		// [ ... x x x (listenPortBytes) 2 x x x (player name) ]
		// [ ^---data----------------------------------------^ ]
		// [                               ^--dataSplAft[1]--^ ]

		return string(dataSplAft[1][3:]), nil

	} else {
		// packet join request:
		// (to get player name, a further packet read is required)
		// [ ... x x x (listenPortBytes) 2 ] [ x x x (player name) ]
		// [ ^---data--------------------^ ] [       ^-data[3:]--^ ]
		// [              dataSplitAft[1]-╝] [                     ]

		data, err := getClientPacket(clientSocket)
		if err != nil {
			return "player unknown", fmt.Errorf("getPlayerName: %v", err)
		}

		return string(data[3:]), nil
	}
}

// extractVersionProtocol finds the serverVersion and serverProtocol in (data []byte) and writes them in the config file
func extractVersionProtocol(data []byte) error {
	// if the above specified data contains "\"version\":{\"name\":\"" and ",\"protocol\":" --> extract the serverVersion and serverProtocol
	if bytes.Contains(data, []byte("\"version\":{\"name\":\"")) && bytes.Contains(data, []byte(",\"protocol\":")) {
		newServerVersion := string(bytes.Split(bytes.Split(data, []byte("\"version\":{\"name\":\""))[1], []byte("\","))[0])
		newServerProtocol := string(bytes.Split(bytes.Split(data, []byte(",\"protocol\":"))[1], []byte("}"))[0])

		// if serverVersion or serverProtocol are different from the ones specified in config.json --> update them
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
