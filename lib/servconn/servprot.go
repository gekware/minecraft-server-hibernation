package servconn

import (
	"bytes"
	"fmt"
	"net"
	"strings"

	"msh/lib/config"
	"msh/lib/data"
	"msh/lib/logger"
)

// BuildMessage takes the format ("txt", "info") and a message to write to the client
func BuildMessage(format, message string) []byte {
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

// AnswerPingReq responds to the ping request
func AnswerPingReq(clientSocket net.Conn) error {
	req := make([]byte, 1024)

	// read the first packet
	dataLen, err := clientSocket.Read(req)
	if err != nil {
		return fmt.Errorf("answerPingReq: error while reading [1] ping request: %v", err)
	}

	// if req == [1, 0] --> read again (the correct ping byte array has still to arrive)
	if bytes.Equal(req[:dataLen], []byte{1, 0}) {
		dataLen, err = clientSocket.Read(req)
		if err != nil {
			return fmt.Errorf("answerPingReq: error while reading [2] ping request: %v", err)
		}
	} else if bytes.Equal(req[:2], []byte{1, 0}) {
		// sometimes the [1 0] is at the beginning and needs to be removed.
		// Example: [1 0 9 1 0 0 0 0 0 89 73 114] -> [9 1 0 0 0 0 0 89 73 114]
		req = req[2:dataLen]
		dataLen = dataLen - 2
	}

	// answer the ping request
	clientSocket.Write(req[:dataLen])

	return nil
}

// GetVersionProtocol finds the serverVersion and serverProtocol in (data []byte) and writes them in the config file
func GetVersionProtocol(data []byte) error {
	// if the above specified buffer contains "\"version\":{\"name\":\"" and ",\"protocol\":" --> extract the serverVersion and serverProtocol
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
