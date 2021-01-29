package servprotocol

import (
	"bytes"
	"fmt"
	"math"
	"net"
	"strings"

	"msh/lib/confctrl"
	"msh/lib/data"
	"msh/lib/debugctrl"
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
			secondByte := math.Floor(float64(len(message) / 128))
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
			"\"version\":{\"name\":\"", confctrl.Config.Advanced.ServerVersion, "\",\"protocol\":", fmt.Sprint(confctrl.Config.Advanced.ServerProtocol), "},",
			"\"favicon\":\"data:image/png;base64,", data.ServerIcon, "\"",
			"}",
		)

		messageHeader = mountHeader(messageJSON)
	}

	return messageHeader
}

// AnswerPingReq responds to the ping request
func AnswerPingReq(clientSocket net.Conn) {
	req := make([]byte, 1024)

	// read the first packet
	dataLen, err := clientSocket.Read(req)
	if err != nil {
		debugctrl.Logger("answerPingReq: error while reading [1] ping request:", err.Error())
		return
	}

	// if req == [1, 0] --> read again (the correct ping byte array has still to arrive)
	if bytes.Equal(req[:dataLen], []byte{1, 0}) {
		dataLen, err = clientSocket.Read(req)
		if err != nil {
			debugctrl.Logger("answerPingReq: error while reading [2] ping request:", err.Error())
			return
		}
	} else if bytes.Equal(req[:2], []byte{1, 0}) {
		// sometimes the [1 0] is at the beginning and needs to be removed.
		// Example: [1 0 9 1 0 0 0 0 0 89 73 114] -> [9 1 0 0 0 0 0 89 73 114]
		req = req[2:dataLen]
		dataLen = dataLen - 2
	}

	// answer the ping request
	clientSocket.Write(req[:dataLen])
}
