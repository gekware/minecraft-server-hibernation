package conn

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"time"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/utility"
)

// reference:
// - wiki.vg/Query
// - github.com/dreamscached/minequery/v2

// clib is a group of query challenges
var clib *challengeLibrary = &challengeLibrary{}

// challenge represents a query challenge uint32 value and its expiration timer
type challenge struct {
	time.Timer
	val uint32
}

// challengeLibrary represents a group of query challenges
type challengeLibrary struct {
	list []challenge
}

// HandlerQuery handles query stats requests.
//
// can only receive requests on config.ListenHost, config.ListenPort
func HandlerQuery() {
	// TODO
	// remove fmt, use errco
	// get query port from msh.config
	// respond with real server info
	// emulate/forward depending on server status

	connUDP, err := net.ListenPacket("udp", fmt.Sprintf("%s:%d", config.ListenHost, config.ListenPort))
	if err != nil {
		errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CLIENT_LISTEN, err.Error())
		return
	}

	// infinite cycle to handle new clients queries
	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "listening for new clients queries on %s:%d ...", config.ListenHost, config.ListenPort)
	for {
		res, addr, sessionID, logMsh := getStatRequest(connUDP)
		if logMsh != nil {
			logMsh.Log(true)
			continue
		}

		switch len(res) {
		// basic stats response
		case 11:
			statRespBasic(connUDP, addr, sessionID)
		// full stats response
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
func getStatRequest(connUDP net.PacketConn) ([]byte, net.Addr, []byte, *errco.MshLog) {
	var buf []byte = make([]byte, 1024)

	// read request (can be a handshake request or a stats request)
	n, addr, err := connUDP.ReadFrom(buf)
	if err != nil {
		return nil, nil, nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_READ, err.Error())
	}
	fmt.Println("received:", buf[:n])

	switch n {

	// handshake request from client
	case 7:
		fmt.Println("performing handshake")

		fmt.Println("received:", buf[:7])

		sessionID := buf[3:7]
		fmt.Println("\tsession id:", sessionID)

		challenge := clib.gen()
		fmt.Println("\tchallenge:", challenge)

		res := bytes.NewBuffer([]byte{9})
		res.Write(sessionID)
		res.WriteString(fmt.Sprintf("%d", challenge) + "\x00")

		_, err = connUDP.WriteTo(res.Bytes(), addr)
		if err != nil {
			return nil, nil, nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_WRITE, err.Error())
		}

		n, addr, err = connUDP.ReadFrom(buf)
		if err != nil {
			return nil, nil, nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_READ, err.Error())
		}
		fmt.Println("received:", buf[:n])

		// if stats request is different from 11 (basic) or 15 (full) then it's unexpected
		if n != 11 && n != 15 {
			return nil, nil, nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_READ, "unexpected number of bytes in stats request")
		}

		fallthrough

	// full/basic stat request from client
	case 11, 15:
		// challenge verification with challenge library
		if !clib.inLibrary(binary.BigEndian.Uint32(buf[7:11])) {
			return nil, nil, nil, errco.NewLog(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_QUERY_CHALLENGE, "challenge failed")
		}

		return buf[:n], addr, buf[3:7], nil

	default:
		return nil, nil, nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_READ, "unexpected number of bytes in stats/handshake request")
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

// InLibrary searches library for non-expired test value
func (cl *challengeLibrary) inLibrary(t uint32) bool {
	for i := 0; i < len(cl.list); i++ {
		select {
		case <-cl.list[i].C:
			// if timer expired, remove challenge and continue iterating
			cl.list = append(cl.list[:i], cl.list[i+1:]...)
			continue
		default:
		}
		if t == cl.list[i].val {
			return true
		}
	}
	return false
}

// Gen generates a int32 challenge and adds it to the challenge library
func (cl *challengeLibrary) gen() uint32 {
	rand.Seed(time.Now().UnixNano())
	cval := (rand.Uint32() % 10_000_000) + 1_000_000

	c := challenge{
		Timer: *time.NewTimer(60 * time.Second),
		val:   cval,
	}

	cl.list = append(cl.list, c)

	return cval
}
