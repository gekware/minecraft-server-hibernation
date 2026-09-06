package conn

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"strings"
	"time"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/model"
	"msh/lib/utility"
)

const (
	// maxPacketLen is the maximum length (in bytes) of the data of a client packet that msh accepts.
	// handshake / login start / ping packets are a few dozen bytes long: this limit is very generous
	// and prevents a malicious client from making msh allocate a huge buffer
	// by declaring an enormous packet length
	maxPacketLen int = 32 * 1024

	// defaultClientPacketTimeout is the time (in seconds) msh waits for a complete client packet
	// when Msh.ClientPacketTimeout is not set (or is invalid) in msh-config.json
	defaultClientPacketTimeout int = 1
)

// buildMessage takes the request type and message to write to the client
func buildMessage(reqType int, message string) []byte {
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

	switch reqType {

	// send text to be shown in the loadscreen
	case errco.CLIENT_REQ_JOIN:
		messageStruct := &model.DataTxt{}
		messageStruct.Text = message

		dataTxtJSON, err := json.Marshal(messageStruct)
		if err != nil {
			// don't return error, just log a warning
			errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_JSON_MARSHAL, err.Error())
			return nil
		}

		return mountHeader(dataTxtJSON)

	// send server info
	case errco.CLIENT_REQ_INFO:

		// "&" [\x26] is converted to "§" [\xc2\xa7]
		// this step is not strictly necessary if in msh-config is used the character "§"
		message = strings.ReplaceAll(message, "&", "§")

		// replace "\\n" with "\n" in case the new line was set as msh parameter
		message = strings.ReplaceAll(message, "\\n", "\n")

		messageStruct := &model.DataInfo{}
		messageStruct.Description.Text = message
		messageStruct.Players.Max = 0
		messageStruct.Players.Online = 0
		messageStruct.Version.Name = config.ConfigRuntime.Server.Version
		messageStruct.Version.Protocol = config.ConfigRuntime.Server.Protocol
		messageStruct.Favicon = "data:image/png;base64," + config.ServerIcon

		dataInfJSON, err := json.Marshal(messageStruct)
		if err != nil {
			// don't return error, just log a warning
			errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_JSON_MARSHAL, err.Error())
			return nil
		}

		return mountHeader(dataInfJSON)

	default:
		return nil
	}
}

// answerClient writes a message to the client and logs the bytes sent.
//
// A write deadline is set explicitly: getClientPacket sets a deadline that applies to reads
// and writes alike, and by the time msh answers, that deadline is meant for a request
// that has already been read and might be about to expire.
//
// clientConn connection should not be closed here (need to be closed in caller function).
func answerClient(clientConn net.Conn, mes []byte) *errco.MshLog {
	clientConn.SetWriteDeadline(time.Now().Add(clientPacketTimeout()))

	_, err := clientConn.Write(mes)
	if err != nil {
		return errco.NewLog(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_CONN_WRITE, err.Error())
	}

	errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> client%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, mes)

	return nil
}

// clientPacketTimeout returns the time msh waits for a complete client packet.
// Msh.ClientPacketTimeout is not set in configs that predate the parameter,
// in which case the original behaviour is kept.
func clientPacketTimeout() time.Duration {
	timeout := config.ConfigRuntime.Msh.ClientPacketTimeout
	if timeout <= 0 {
		timeout = defaultClientPacketTimeout
	}

	return time.Duration(timeout) * time.Second
}

// getReqType returns the request packet, type (INFO or JOIN).
// Not player name as it's too difficult to extract.
func getReqType(clientConn net.Conn) ([]byte, int, *errco.MshLog) {
	var dataReqFull []byte

	data, logMsh := getClientPacket(clientConn)
	if logMsh != nil {
		return nil, errco.CLIENT_REQ_UNKN, logMsh.AddTrace()
	}

	dataReqFull = data

	// the handshake packet fields are parsed in order to extract the "next state" field.
	// the packet must not be searched for a flag composed of the msh port bytes followed by
	// the request type byte: that same sequence can appear inside the server address field
	// (which is length prefixed and can contain any byte), resulting in a wrong request type
	nextState, logMsh := parseHandshake(dataReqFull)
	if logMsh != nil {
		// log why the handshake could not be parsed (only shown at byte log level),
		// then report the request as unknown
		logMsh.Log(true)
		return nil, errco.CLIENT_REQ_UNKN, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CLIENT_REQ, "client request unknown (received: %v)", dataReqFull)
	}

	switch nextState {
	case 1:
		// client is requesting server info
		// example: [ 16 0 244 5 9 49 50 55 46 48 46 48 46 49 99 211 1 ]
		// scheme:  [ 16  | 0  | 244 5    | 9 49 50 55 46 48 46 48 46 49 | 99 211      | 1          ]
		//          [ len | id | protocol | server address              | server port | next state ]

		return dataReqFull, errco.CLIENT_REQ_INFO, nil

	case 2:
		// client is trying to join the server
		// example: [ 16 0 244 5 9 49 50 55 46 48 46 48 46 49 99 211 2 ][ 11 0 9 x ... x (player name) ]
		//          [                 handshake packet                ][      login start packet      ]

		// the handshake packet is followed by the login start packet (which contains the player name):
		// it must be read too, since HandlerClientConn scans the returned bytes for whitelisted
		// players and openProxy forwards them to the minecraft server.
		// after ms 1.19.3 not always there is a EOF after the handshake packet, so msh can't rely
		// on the client sending it separately (bugfix #197).
		// now that getClientPacket returns exactly one packet, the login start packet
		// is always read with a separate call
		data, logMsh = getClientPacket(clientConn)
		if logMsh != nil {
			return nil, errco.CLIENT_REQ_UNKN, logMsh.AddTrace() // return request unknown as the request failed
		}
		dataReqFull = append(dataReqFull, data...)

		return dataReqFull, errco.CLIENT_REQ_JOIN, nil

	default:
		return nil, errco.CLIENT_REQ_UNKN, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CLIENT_REQ, "client request unknown (received: %v)", dataReqFull)
	}
}

// parseHandshake parses a minecraft handshake packet and returns its "next state" field
// (1 when the client is requesting server info, 2 when the client is trying to join).
//
// handshake packet scheme:
// [ packet length (VarInt) | packet id (VarInt, 0) | protocol version (VarInt) | server address (String) | server port (unsigned short) | next state (VarInt) ]
//
// the fields must be parsed in order: "next state" can't be located by indexing the packet,
// since the fields before it have a variable length (VarInts and the server address string)
func parseHandshake(packet []byte) (int, *errco.MshLog) {
	// packet length
	_, n, logMsh := utility.ParseVarInt(packet, 0)
	if logMsh != nil {
		return 0, logMsh.AddTrace()
	}
	offset := n

	// packet id (must be 0 for a handshake packet)
	packetID, n, logMsh := utility.ParseVarInt(packet, offset)
	if logMsh != nil {
		return 0, logMsh.AddTrace()
	}
	if packetID != 0 {
		return 0, errco.NewLog(errco.TYPE_WAR, errco.LVL_4, errco.ERROR_CLIENT_REQ, "packet id is not of an handshake packet (%d)", packetID)
	}
	offset += n

	// protocol version
	_, n, logMsh = utility.ParseVarInt(packet, offset)
	if logMsh != nil {
		return 0, logMsh.AddTrace()
	}
	offset += n

	// server address (String: VarInt length followed by that amount of bytes)
	addressLen, n, logMsh := utility.ParseVarInt(packet, offset)
	if logMsh != nil {
		return 0, logMsh.AddTrace()
	}
	offset += n
	if addressLen < 0 || offset+addressLen > len(packet) {
		return 0, errco.NewLog(errco.TYPE_WAR, errco.LVL_4, errco.ERROR_CLIENT_REQ, "server address field is truncated")
	}
	offset += addressLen

	// server port (unsigned short)
	if offset+2 > len(packet) {
		return 0, errco.NewLog(errco.TYPE_WAR, errco.LVL_4, errco.ERROR_CLIENT_REQ, "server port field is truncated")
	}
	offset += 2

	// next state
	nextState, _, logMsh := utility.ParseVarInt(packet, offset)
	if logMsh != nil {
		return 0, logMsh.AddTrace()
	}

	return nextState, nil
}

// getPing performs msh PING response to the client PING request
// (must be performed after msh INFO response)
func getPing(clientConn net.Conn) *errco.MshLog {
	// read the first packet
	pingData, logMsh := getClientPacket(clientConn)
	if logMsh != nil {
		return logMsh.AddTrace()
	}

	switch {
	case bytes.HasPrefix(pingData, []byte{9, 1, 0, 0, 0, 0, 0}):
		// this is a normal ping
		// packet is [9 1 0 0 0 0 0 89 73 114]
		// using [9 1 0 0 0 0 0] as prefix is a conservative check
		// in case other "normal" pings are discovered it's possible to use a shorter/different prefix check

	case bytes.Equal(pingData, []byte{1, 0}):
		// packet is [1 0]
		// read the second packet
		pingData, logMsh = getClientPacket(clientConn)
		if logMsh != nil {
			return logMsh.AddTrace()
		}

	// bufix #202: don't use `bytes.Equal(pingData[:2], []byte{1, 0})`
	// (if pingData == [1] then pingData[:2] == [1 0])
	// this results in pingData[2:] -> slice bounds out of range [2:1]
	case bytes.HasPrefix(pingData, []byte{1, 0}):
		// packet is [1 0 9 1 0 0 0 0 0 89 73 114]
		// remove first 2 bytes: [1 0 9 1 0 0 0 0 0 89 73 114] -> [9 1 0 0 0 0 0 89 73 114]
		pingData = pingData[2:]

	default:
		return errco.NewLog(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_PING_PACKET_UNKNOWN, "received unknown ping packet: %v", pingData)
	}

	// answer ping
	logMsh = answerClient(clientConn, pingData)
	if logMsh != nil {
		return logMsh.AddTrace()
	}

	return nil
}

// getClientPacket reads one complete packet from the client socket and returns its bytes
// (packet length VarInt included).
//
// packet scheme: [ packet length (VarInt) | packet data (packet length bytes) ]
//
// tcp is a stream of bytes without message boundaries: a single Read() can return a partial
// packet (or more than one packet), so bytes are read until the packet is complete and
// not a single byte more (the bytes that follow belong to the next packet and must be
// left on the socket for the caller / minecraft server).
// clientConn connection should not be closed here (need to be closed in caller function).
func getClientPacket(clientConn net.Conn) ([]byte, *errco.MshLog) {
	// set deadline to avoid hanging when client is not sending a packet that msh expects.
	// the deadline is absolute and set once: it must not be renewed while reading the packet,
	// otherwise a slow client could keep the connection open indefinitely by sending 1 byte at a time
	clientConn.SetReadDeadline(time.Now().Add(clientPacketTimeout()))

	// read packet length
	packetLen, packetLenByt, logMsh := readVarInt(clientConn)
	if logMsh != nil {
		return nil, logMsh.AddTrace()
	}

	// check packet length before allocating memory for it
	// (packetLen < 0 catches a 5 bytes VarInt overflowing int on 32 bit systems)
	if packetLen < 0 || packetLen > maxPacketLen {
		return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CLIENT_SOCKET_READ, "client declared a packet length out of range (%d)", packetLen)
	}

	// read packet data (keep reading until the whole packet has been received)
	packet := make([]byte, len(packetLenByt)+packetLen)
	copy(packet, packetLenByt)
	_, err := io.ReadFull(clientConn, packet[len(packetLenByt):])
	if err != nil {
		return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CLIENT_SOCKET_READ, err.Error())
	}

	errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%sclient --> msh%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, packet)

	return packet, nil
}

// readVarInt reads a minecraft protocol VarInt from r and returns
// its value and the raw bytes it is composed of.
// bytes are read one at a time (without buffering) so that the bytes following
// the VarInt are left untouched on the socket.
func readVarInt(r io.Reader) (int, []byte, *errco.MshLog) {
	// scheme: [ 1xxxxxxx | 1xxxxxxx | ... | 0xxxxxxx ]
	// every byte carries 7 bits of data and the most significant bit signals that an other byte follows.
	// a VarInt is composed of 5 bytes at most

	var value int
	byt := make([]byte, 1)
	raw := make([]byte, 0, 5)

	for i := 0; i < 5; i++ {
		if _, err := io.ReadFull(r, byt); err != nil {
			return 0, nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CLIENT_SOCKET_READ, err.Error())
		}

		raw = append(raw, byt[0])
		value |= int(byt[0]&0x7f) << (7 * i)

		// most significant bit not set: this is the last byte of the VarInt
		if byt[0]&0x80 == 0 {
			return value, raw, nil
		}
	}

	return 0, nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CLIENT_SOCKET_READ, "client sent a VarInt longer than 5 bytes")
}
