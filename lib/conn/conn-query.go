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
	"msh/lib/progmgr"
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
// Accepts requests on config.MshHost, config.MshPortQuery
func HandlerQuery() {
	// TODO
	// respond with real server info
	// emulate/forward depending on server status

	connUDP, err := net.ListenPacket("udp", fmt.Sprintf("%s:%d", config.MshHost, config.MshPortQuery))
	if err != nil {
		errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CLIENT_LISTEN, err.Error())
		return
	}

	// infinite cycle to handle new clients queries
	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "listening for new clients queries\ton %s:%d ...", config.MshHost, config.MshPortQuery)
	for {
		res, addr, sessionID, logMsh := getStatsRequest(connUDP)
		if logMsh != nil {
			logMsh.Log(true)
			continue
		}

		switch len(res) {
		case 11: // basic stats response
			statRespBasic(connUDP, addr, sessionID)
		case 15: // full stats response
			statRespFull(connUDP, addr, sessionID)
		default:
			errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_QUERY_BAD_REQUEST, "cannot define stat request type (unexpected number of bytes)")
		}
	}
}

// getStatsRequest gets stats request from client.
// (performing handshake if necessay)
//
// returns buffer (lenght: 11, 15), address, session id, error
func getStatsRequest(connUDP net.PacketConn) ([]byte, net.Addr, []byte, *errco.MshLog) {
	var buf []byte = make([]byte, 1024)

	// stats / handshake request read
	n, addr, err := connUDP.ReadFrom(buf)
	if err != nil {
		return nil, nil, nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_READ, err.Error())
	}

	switch n {

	case 7: // handshake request from client
		errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "recv handshake request:\t%v", buf[:7])

		// handshake response composition
		res := bytes.NewBuffer([]byte{9})                       // type: handshake
		res.Write(buf[3:7])                                     // session id
		res.WriteString(fmt.Sprintf("%d", clib.gen()) + "\x00") // challenge (int32 written as string, null terminated)

		// handshake response send
		errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "send handshake response:\t%v", res.Bytes())
		_, err = connUDP.WriteTo(res.Bytes(), addr)
		if err != nil {
			return nil, nil, nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_WRITE, err.Error())
		}

		// stats request read
		n, addr, err = connUDP.ReadFrom(buf)
		if err != nil {
			return nil, nil, nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_READ, err.Error())
		}

		// check that stats request has expected lenght (11: basic, 15: full)
		if n != 11 && n != 15 {
			return nil, nil, nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_READ, "unexpected number of bytes in stats request")
		}

		fallthrough

	case 11, 15: // full / basic stats request from client
		errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "recv stats request:\t%v", buf[:n])

		// check that received challenge is known and not expired
		if !clib.inLibrary(binary.BigEndian.Uint32(buf[7:11])) {
			return nil, nil, nil, errco.NewLog(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_QUERY_CHALLENGE, "challenge failed")
		}

		// return buffer (lenght: 11, 15), address, session id, error
		return buf[:n], addr, buf[3:7], nil

	default:
		return nil, nil, nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_READ, "unexpected number of bytes in stats / handshake request")
	}
}

// statRespFull writes a full stats response to udp connection
func statRespFull(connUDP net.PacketConn, addr net.Addr, sessionID []byte) {
	var buf bytes.Buffer
	buf.WriteByte(0)                        // type
	buf.Write(sessionID)                    // session ID
	buf.WriteString("splitnum\x00\x80\x00") // padding (default)

	// K, V section
	buf.WriteString(fmt.Sprintf("hostname\x00%s\x00", config.ConfigRuntime.Msh.InfoHibernation))
	buf.WriteString(fmt.Sprintf("gametype\x00%s\x00", "SMP"))      // hardcoded (default)
	buf.WriteString(fmt.Sprintf("game_id\x00%s\x00", "MINECRAFT")) // hardcoded (default)
	buf.WriteString(fmt.Sprintf("version\x00%s\x00", config.ConfigRuntime.Server.Version))
	buf.WriteString(fmt.Sprintf("plugins\x00msh/%s: msh %s\x00", config.ConfigRuntime.Server.Version, progmgr.MshVersion)) // example: "plugins\x00{ServerVersion}: {Name} {Version}; {Name} {Version}\x00"
	levelName, _ := config.ConfigRuntime.ParsePropertiesString("level-name")
	buf.WriteString(fmt.Sprintf("map\x00%s\x00", levelName))
	buf.WriteString("numplayers\x000\x00") // hardcoded
	buf.WriteString("maxplayers\x000\x00") // hardcoded
	buf.WriteString(fmt.Sprintf("hostport\x00%d\x00", config.MshPort))
	buf.WriteString(fmt.Sprintf("hostip\x00%s\x00", utility.GetOutboundIP4()))
	buf.WriteByte(0) // termination of section (?)

	// Players
	buf.WriteString("\x01player_\x00\x00") // padding (default)
	buf.WriteString("\x00")                // example: "aaa\x00bbb\x00\x00"

	errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "send stats full response:\t%v", buf.Bytes())
	_, err := connUDP.WriteTo(buf.Bytes(), addr)
	if err != nil {
		errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_WRITE, err.Error())
	}
}

// statRespFull writes a full stats response to udp connection
func statRespBasic(connUDP net.PacketConn, addr net.Addr, sessionID []byte) {
	var buf bytes.Buffer
	buf.WriteByte(0)                                                                 // type
	buf.Write(sessionID)                                                             // session ID
	buf.WriteString(fmt.Sprintf("%s\x00", config.ConfigRuntime.Msh.InfoHibernation)) // MOTD
	buf.WriteString("SMP\x00")                                                       // gametype hardcoded (default)
	levelName, _ := config.ConfigRuntime.ParsePropertiesString("level-name")
	buf.WriteString(fmt.Sprintf("%s\x00", levelName))                                      // map
	buf.WriteString("0\x00")                                                               // numplayers hardcoded
	buf.WriteString("0\x00")                                                               // maxplayers hardcoded
	buf.Write(append(utility.Reverse(big.NewInt(int64(config.MshPort)).Bytes()), byte(0))) // hostport
	buf.WriteString(fmt.Sprintf("%s\x00", utility.GetOutboundIP4()))                       // hostip

	errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "send stats basic response:\t%v", buf.Bytes())
	_, err := connUDP.WriteTo(buf.Bytes(), addr)
	if err != nil {
		errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_WRITE, err.Error())
	}
}

// Gen generates a int32 challenge and adds it to the challenge library
func (cl *challengeLibrary) gen() uint32 {
	rand.Seed(time.Now().UnixNano())
	cval := uint32(rand.Int31n(9_999_999-1_000_000+1) + 1_000_000)

	c := challenge{
		Timer: *time.NewTimer(60 * time.Second),
		val:   cval,
	}

	cl.list = append(cl.list, c)

	return cval
}

// InLibrary searches library for non-expired test value
func (cl *challengeLibrary) inLibrary(t uint32) bool {
	// result var is used so that the list is completely scanned and expired values are removed
	result := false

	// scanning list in reverse to remove elements while iterating on them
	for i := len(cl.list) - 1; i >= 0; i-- {
		select {
		case <-cl.list[i].C:
			// if timer expired, remove challenge and continue iterating
			cl.list = append(cl.list[:i], cl.list[i+1:]...)
			continue
		default:
		}

		if t == cl.list[i].val {
			result = true
		}
	}

	return result
}
