package conn

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
	"net"
	"strconv"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/utility"
)

// reference:
// - wiki.vg/Query
// - github.com/dreamscached/minequery/v2

// HandlerQuery handles query requests
//
// can only receive query requests on config.ListenHost, config.ListenPort
func HandlerQuery() {
	// TODO
	// remove fmt, use errco
	// get query port from server.properties

	connUDP, err := net.ListenPacket("udp", fmt.Sprintf("%s:%d", config.ListenHost, config.ListenPort))
	if err != nil {
		errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CLIENT_LISTEN, err.Error())
		return
	}

	// infinite cycle to handle new clients queries
	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "listening for new clients queries on %s:%d ...", config.ListenHost, config.ListenPort)
	for {
		buf, addr, logMsh := getStatRequest(connUDP)
		if logMsh != nil {
			logMsh.Log(true)
			continue
		}

		fmt.Println(len(buf))

		sessionID := buf[3:7]
		chall := buf[7:11]

		fmt.Println("received:", buf)
		fmt.Println("\tsession id:", sessionID)
		fmt.Println("\tchallenge:           ", chall)

		challNum, err := strconv.ParseUint("9513307", 10, 32)
		if err != nil {
			errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_ANALYSIS, err.Error())
			continue
		}

		if binary.BigEndian.Uint32(chall) != uint32(challNum) {
			errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_QUERY_CHALLENGE, "challenge failed")
			continue
		}
		fmt.Println("challenge ok")

		switch len(buf) {
		case 11:
			statRespBasic(connUDP, addr, sessionID)
		case 15:
			statRespFull(connUDP, addr, sessionID)
		default:
			errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_QUERY_BAD_REQUEST, "cannot define stat request type (unexpected number of bytes)")
			continue
		}
	}
}

// getStatRequest gets stats request from client.
// (performing handshake if necessay)
//
// returns buffer (lenght: 11, 15), address, error
func getStatRequest(connUDP net.PacketConn) ([]byte, net.Addr, *errco.MshLog) {
	var n int
	var addr net.Addr
	var err error
	var buf []byte = make([]byte, 1024)

	// read request (can be a handshake request or a stats request)
	n, addr, err = connUDP.ReadFrom(buf)
	if err != nil {
		return nil, nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_READ, err.Error())
	}

	switch n {
	case 7: // handshake request from client
		fmt.Println("performing handshake")

		fmt.Println("received:", buf[:7])

		sessionID := buf[3:7]
		fmt.Println("\tsession id:", sessionID)

		res := bytes.NewBuffer([]byte{9})
		res.Write(sessionID)
		res.WriteString("9513307\x00")
		_, err = connUDP.WriteTo(res.Bytes(), addr)
		if err != nil {
			return nil, nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_WRITE, err.Error())
		}

		n, addr, err = connUDP.ReadFrom(buf)
		if err != nil {
			return nil, nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_READ, err.Error())
		}

		// if stats request is different from 11 (basic) or 15 (full) then it's unexpected
		if n != 11 && n != 15 {
			return nil, nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_READ, "read unexpected number of bytes in stats request")
		}

		fallthrough

	case 11, 15: // full/basic stat request from client
		return buf[:n], addr, nil

	default:
		return nil, nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_READ, "cannot define stat/handshake request (unexpected number of bytes)")
	}
}

// statRespFull writes a full stats response to udp connection
func statRespFull(connUDP net.PacketConn, addr net.Addr, sessionID []byte) {
	var buf bytes.Buffer
	buf.WriteByte(0)                        // type
	buf.Write(sessionID)                    // session ID
	buf.WriteString("splitnum\x00\x80\x00") // padding (default)

	// K, V section
	buf.WriteString("hostname\x00A Minecraft Server\x00")
	buf.WriteString("gametype\x00SMP\x00")
	buf.WriteString("game_id\x00MINECRAFT\x00")
	buf.WriteString("version\x001.2.5\x00")
	buf.WriteString("plugins\x00vanilla: plug1 v1; plug2 v2\x00")
	buf.WriteString("map\x00world\x00")
	buf.WriteString("numplayers\x001\x00")
	buf.WriteString("maxplayers\x0020\x00")
	buf.WriteString("hostport\x0025565\x00")
	buf.WriteString("hostip\x00127.0.0.1\x00")
	buf.WriteByte(0) // termination of section (?)

	// Players
	buf.WriteString("\x01player_\x00\x00") // padding (default)
	buf.WriteString("aaa\x00bbb\x00\x00")  // null terminated list

	_, err := connUDP.WriteTo(buf.Bytes(), addr)
	if err != nil {
		errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_WRITE, err.Error())
	}
}

// statRespFull writes a full stats response to udp connection
func statRespBasic(connUDP net.PacketConn, addr net.Addr, sessionID []byte) {
	var buf bytes.Buffer
	buf.WriteByte(0)                                                              // type
	buf.Write(sessionID)                                                          // session ID
	buf.WriteString("A Minecraft Server\x00")                                     // MOTD
	buf.WriteString("SMP\x00")                                                    // gametype
	buf.WriteString("world\x00")                                                  // map
	buf.WriteString("1\x00")                                                      // numplayers
	buf.WriteString("20\x00")                                                     // maxplayers
	buf.Write(append(utility.Reverse(big.NewInt(int64(25565)).Bytes()), byte(0))) // hostport
	buf.WriteString("127.0.0.1\x00")                                              // hostip

	_, err := connUDP.WriteTo(buf.Bytes(), addr)
	if err != nil {
		errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_WRITE, err.Error())
	}
}
