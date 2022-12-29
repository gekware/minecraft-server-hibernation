package conn

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"

	"msh/lib/config"
	"msh/lib/errco"
)

// HandlerQuery handles query requests
//
// this is just a test
//
// can only receive query requests on config.ListenHost, config.ListenPort
func HandlerQuery() {
	// TODO
	// get query port from server.properties

	connUDP, err := net.ListenPacket("udp", fmt.Sprintf("%s:%d", config.ListenHost, config.ListenPort))
	if err != nil {
		errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CLIENT_LISTEN, err.Error())
		return
	}

	// infinite cycle to handle new clients queries
	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "listening for new clients queries on %s:%d ...", config.ListenHost, config.ListenPort)
	for {
		// ----------- Handshake ----------- //
		buf := make([]byte, 1024)

		// read request
		_, addr, err := connUDP.ReadFrom(buf)
		if err != nil {
			errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_READ, err.Error())
			continue
		}
		sessionID := buf[3:7]
		fmt.Println("session id:", sessionID)

		res := []byte{buf[2]}
		res = append(res, sessionID...)
		res = append(res, []byte("9513307\x00")...)

		// send response
		_, err = connUDP.WriteTo(res, addr)
		if err != nil {
			errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_WRITE, err.Error())
			continue
		}

		// ------------- Stats ------------- //
		buf = make([]byte, 1024)

		// read request
		n, addr, err := connUDP.ReadFrom(buf)
		if err != nil {
			errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CLIENT_SOCKET_READ, err.Error())
		}

		challNum, err := strconv.ParseUint("9513307", 10, 32)
		if err != nil {
			errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_ANALYSIS, err.Error())
		}

		if binary.BigEndian.Uint32(buf[7:11]) != uint32(challNum) {
			errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_QUERY_CHALLENGE, err.Error())
		}
		fmt.Println("challenge verified")

		sessionID = buf[3:7]
		fmt.Println("session id:", sessionID)

		// check if there is Padding (Full stat) or no padding (Basic stat)
		switch {
		case n > 11:
			statRespFull(connUDP, addr, sessionID)
		case n <= 11:
			// statRespBase(connUDP, addr, sessionID)
		}
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
