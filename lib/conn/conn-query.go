package conn

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"strconv"
	"time"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/progmgr"
	"msh/lib/servctrl"
	"msh/lib/servstats"
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
	connCli, err := net.ListenPacket("udp", fmt.Sprintf("%s:%d", config.MshHost, config.MshPortQuery))
	if err != nil {
		errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CLIENT_LISTEN, err.Error())
		return
	}

	// infinite cycle to handle new clients queries
	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "%-40s %s:%d ...", "listening for new clients queries on", config.MshHost, config.MshPortQuery)
	for {
		// handshake / stats request read
		var buf []byte = make([]byte, 1024)
		n, addrCli, err := connCli.ReadFrom(buf)
		if err != nil {
			errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_READ, err.Error())
			continue
		}

		// if minecraft server is not warm, handle request
		logMsh := handleRequest(connCli, addrCli, buf[:n])
		if logMsh != nil {
			logMsh.Log(true)
		}
	}
}

// handleRequest handles handshake / stats request from client performing handshake / stats response.
func handleRequest(connCli net.PacketConn, addr net.Addr, reqClient []byte) *errco.MshLog {
	switch len(reqClient) {

	case 7: // handshake request from client
		errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "recv handshake req:\t%v", reqClient)

		sessionID := reqClient[3:7]

		// handshake response composition
		rsp := bytes.NewBuffer([]byte{9})                       // type: handshake
		rsp.Write(sessionID)                                    // session id
		rsp.WriteString(fmt.Sprintf("%d", clib.gen()) + "\x00") // challenge (int32 written as string, null terminated)

		// handshake response send
		errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "send handshake rsp:\t%v", rsp.Bytes())
		_, err := connCli.WriteTo(rsp.Bytes(), addr)
		if err != nil {
			return errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_WRITE, err.Error())
		}

		return nil

	case 11, 15: // full / base stats request from client
		errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "recv stats req:\t%v", reqClient)

		sessionID := reqClient[3:7]
		challenge := reqClient[7:11]

		// check that received challenge is known and not expired
		if !clib.inLibrary(binary.BigEndian.Uint32(challenge)) {
			return errco.NewLog(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_QUERY_CHALLENGE, "challenge failed")
		}

		// if ms is not warm emulate response
		logMsh := servctrl.CheckMSWarm()
		if logMsh != nil {
			switch len(reqClient) {
			case 11: // base stats response
				statsRespBase(connCli, addr, sessionID)
			case 15: // full stats response
				statsRespFull(connCli, addr, sessionID)
			}
			return nil
		}

		// if ms is warm get response and send it to client
		stats, logMsh := statsGet(reqClient)
		if logMsh != nil {
			return logMsh.AddTrace()
		}

		_, err := connCli.WriteTo(stats, addr)
		if err != nil {
			return errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_WRITE, err.Error())
		}
		errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "send stats rsp:\t%v", stats)

		return nil

	default:
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_READ, "unexpected number of bytes in stats / handshake request")
	}
}

// statsGet connects to ms and performs a stats base/full request.
// Returns the stats data already adapted for the client response.
func statsGet(reqClient []byte) ([]byte, *errco.MshLog) {
	// Dial the server using a UDP connection
	conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", config.ServHost, config.ServPortQuery))
	if err != nil {
		return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_SERVER_DIAL, err.Error())
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(time.Second))

	// ---------- ms query handshake ----------- //

	// request handshake
	data := bytes.NewBuffer([]byte{254, 253}) // magic
	data.WriteByte(9)                         // handshake code
	data.Write([]byte{1, 2, 3, 4})            // session id
	_, err = conn.Write(data.Bytes())
	if err != nil {
		return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_WRITE, err.Error())
	}
	errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, " ├ send handshake req (-> ms):\t%v", data.Bytes())

	// receive handshake
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_READ, err.Error())
	}
	errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, " ├ recv handshake rsp (<- ms):\t%v", buf[:n])

	// calculate challenge
	chall := bytes.NewBuffer(nil)
	if i, err := strconv.ParseUint(string(buf[5:n-1]), 10, 32); err != nil {
		return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_ANALYSIS, err.Error())
	} else if err = binary.Write(chall, binary.BigEndian, uint32(i)); err != nil {
		return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_ANALYSIS, err.Error())
	}

	// ------------ ms query stats ------------- //

	// request base / full stats
	data = bytes.NewBuffer([]byte{254, 253}) // magic
	data.WriteByte(0)                        // stats code
	data.Write([]byte{1, 2, 3, 4})           // session id
	data.Write(chall.Bytes())                // challenge
	if len(reqClient) == 15 {
		data.Write([]byte{0, 0, 0, 0}) // full request
	}
	_, err = conn.Write(data.Bytes())
	if err != nil {
		return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_WRITE, err.Error())
	}
	errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, " ├ send stats req (-> ms):\t%v", data.Bytes())

	// receive base / full stats
	n, err = conn.Read(buf)
	if err != nil {
		return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_READ, err.Error())
	}
	errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, " └ recv stats rsp (<- ms):\t%v", buf[:n])

	// adapt server stats response to client session id
	data = bytes.NewBuffer(reqClient[2:7]) // stats code (0) + session id (from client request)
	data.Write(buf[5:n])                   // stats (from server response)

	return data.Bytes(), nil
}

// statsRespBase writes a base stats response to client
func statsRespBase(connCli net.PacketConn, addr net.Addr, sessionID []byte) {
	levelName, _ := config.ConfigRuntime.ParsePropertiesString("level-name")
	mshPortSmallEndian := utility.Reverse(big.NewInt(int64(config.MshPort)).Bytes())
	var motd string
	switch {
	case servstats.Stats.Status == errco.SERVER_STATUS_OFFLINE || servstats.Stats.Suspended:
		motd = config.ConfigRuntime.Msh.InfoHibernation
	case servstats.Stats.Status == errco.SERVER_STATUS_STARTING:
		motd = config.ConfigRuntime.Msh.InfoStarting
	case servstats.Stats.Status == errco.SERVER_STATUS_ONLINE:
		// server can't be online if this function was called
	case servstats.Stats.Status == errco.SERVER_STATUS_STOPPING:
		motd = "minecraft server is stopping..."
	}

	buf := bytes.NewBuffer(nil)
	buf.WriteByte(0)                                                 // type
	buf.Write(sessionID)                                             // session ID
	buf.WriteString(fmt.Sprintf("%s\x00", motd))                     // MOTD
	buf.WriteString("SMP\x00")                                       // gametype hardcoded (default)
	buf.WriteString(fmt.Sprintf("%s\x00", levelName))                // map
	buf.WriteString("0\x00")                                         // numplayers hardcoded
	buf.WriteString("0\x00")                                         // maxplayers hardcoded
	buf.Write(append(mshPortSmallEndian, byte(0)))                   // hostport
	buf.WriteString(fmt.Sprintf("%s\x00", utility.GetOutboundIP4())) // hostip

	errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "send stats base rsp:\t%v", buf.Bytes())
	_, err := connCli.WriteTo(buf.Bytes(), addr)
	if err != nil {
		errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_WRITE, err.Error())
	}
}

// statsRespFull writes a full stats response to client
func statsRespFull(connCli net.PacketConn, addr net.Addr, sessionID []byte) {
	levelName, _ := config.ConfigRuntime.ParsePropertiesString("level-name")
	var motd string
	switch {
	case servstats.Stats.Status == errco.SERVER_STATUS_OFFLINE || servstats.Stats.Suspended:
		motd = config.ConfigRuntime.Msh.InfoHibernation
	case servstats.Stats.Status == errco.SERVER_STATUS_STARTING:
		motd = config.ConfigRuntime.Msh.InfoStarting
	case servstats.Stats.Status == errco.SERVER_STATUS_ONLINE:
		// server can't be online if this function was called
	case servstats.Stats.Status == errco.SERVER_STATUS_STOPPING:
		motd = "minecraft server is stopping..."
	}

	buf := bytes.NewBuffer(nil)
	buf.WriteByte(0)                        // type
	buf.Write(sessionID)                    // session ID
	buf.WriteString("splitnum\x00\x80\x00") // padding (default)

	// K, V section
	buf.WriteString(fmt.Sprintf("hostname\x00%s\x00", motd))
	buf.WriteString(fmt.Sprintf("gametype\x00%s\x00", "SMP"))      // hardcoded (default)
	buf.WriteString(fmt.Sprintf("game_id\x00%s\x00", "MINECRAFT")) // hardcoded (default)
	buf.WriteString(fmt.Sprintf("version\x00%s\x00", config.ConfigRuntime.Server.Version))
	buf.WriteString(fmt.Sprintf("plugins\x00msh/%s: msh %s\x00", config.ConfigRuntime.Server.Version, progmgr.MshVersion)) // example: "plugins\x00{ServerVersion}: {Name} {Version}; {Name} {Version}\x00"
	buf.WriteString(fmt.Sprintf("map\x00%s\x00", levelName))
	buf.WriteString("numplayers\x000\x00") // hardcoded
	buf.WriteString("maxplayers\x000\x00") // hardcoded
	buf.WriteString(fmt.Sprintf("hostport\x00%d\x00", config.MshPort))
	buf.WriteString(fmt.Sprintf("hostip\x00%s\x00", utility.GetOutboundIP4()))
	buf.WriteByte(0) // termination of section (?)

	// Players
	buf.WriteString("\x01player_\x00\x00") // padding (default)
	buf.WriteString("\x00")                // example: "aaa\x00bbb\x00\x00"

	errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "send stats full rsp:\t%v", buf.Bytes())
	_, err := connCli.WriteTo(buf.Bytes(), addr)
	if err != nil {
		errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONN_WRITE, err.Error())
	}
}

// Gen generates a int32 challenge and adds it to the challenge library
func (cl *challengeLibrary) gen() uint32 {
	rand.Seed(time.Now().UnixNano())
	cval := uint32(rand.Int31n(9_999_999-1_000_000+1) + 1_000_000)

	c := challenge{
		Timer: *time.NewTimer(time.Hour),
		val:   cval,
	}

	cl.list = append(cl.list, c)

	return cval
}

// InLibrary searches library for non-expired test value
func (cl *challengeLibrary) inLibrary(t uint32) bool {
	// remove expired challenges
	// (reverse list loop to remove elements while iterating on them)
	for i := len(cl.list) - 1; i >= 0; i-- {
		select {
		case <-cl.list[i].C:
			cl.list = append(cl.list[:i], cl.list[i+1:]...)
		default:
		}
	}

	// search for non-expired test value
	for i := 0; i < len(cl.list); i++ {
		if t == cl.list[i].val {
			return true
		}
	}

	return false
}
