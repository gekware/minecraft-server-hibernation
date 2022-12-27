package conn

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/progmgr"
)

// HandlerQuery handles query requests
//
// this is just a test
func HandlerQuery() {
	// TODO
	// check query enabled in server.properties
	// get query port from server.properties

	buf := make([]byte, 1024)

	listenerUDP, err := net.ListenPacket("udp", fmt.Sprintf("%s:%d", config.ListenHost, config.ListenPort))
	if err != nil {
		errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CLIENT_LISTEN, err.Error())
		progmgr.AutoTerminate()
	}

	// infinite cycle to handle new clients queries
	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "listening for new clients queries on %s:%d ...", config.ListenHost, config.ListenPort)
	for {
		n, addr, err := listenerUDP.ReadFrom(buf)
		if err != nil {
			fmt.Println("Error reading from connection:", err)
		}
		fmt.Println("recived:", buf[:n])
		sessID := buf[3:7]
		fmt.Println("session id:", sessID)

		res := []byte{buf[2]}
		res = append(res, sessID...)
		res = append(res, []byte("9513307\x00")...)

		fmt.Println("writing:", res)
		listenerUDP.WriteTo(res, addr)

		n, addr, err = listenerUDP.ReadFrom(buf)
		if err != nil {
			fmt.Println("Error reading from connection:", err)
		}
		fmt.Println("recived:", buf[:n])

		challNum, err := strconv.ParseUint("9513307", 10, 32)
		if err != nil {
			fmt.Println(err)
		}
		sessID = buf[3:7]
		fmt.Println("session id:", sessID)
		fmt.Println("received challenge:", buf[7:11])
		if binary.BigEndian.Uint32(buf[7:11]) != uint32(challNum) {
			fmt.Println("challenge failed")
		}
		fmt.Println("challenge verified")

		// check if there is Padding (Full stat) or no padding (Basic stat)

		// write response (full stat / basic stat)
		res = []byte{buf[0]}
		res = append(res, sessID...)
		res = append(res, []byte{115, 112, 108, 105, 116, 110, 117, 109, 0, 128, 0, 104, 111, 115, 116, 110, 97, 109, 101, 0, 65, 32, 77, 105, 110, 101, 99, 114, 97, 102, 116, 32, 83, 101, 114, 118, 101, 114, 0, 103, 97, 109, 101, 116, 121, 112, 101, 0, 83, 77, 80, 0, 103, 97, 109, 101, 95, 105, 100, 0, 77, 73, 78, 69, 67, 82, 65, 70, 84, 0, 118, 101, 114, 115, 105, 111, 110, 0, 66, 101, 116, 97, 32, 49, 46, 57, 32, 80, 114, 101, 114, 101, 108, 101, 97, 115, 101, 32, 52, 0, 112, 108, 117, 103, 105, 110, 115, 0, 0, 109, 97, 112, 0, 119, 111, 114, 108, 100, 0, 110, 117, 109, 112, 108, 97, 121, 101, 114, 115, 0, 50, 0, 109, 97, 120, 112, 108, 97, 121, 101, 114, 115, 0, 50, 48, 0, 104, 111, 115, 116, 112, 111, 114, 116, 0, 50, 53, 53, 54, 53, 0, 104, 111, 115, 116, 105, 112, 0, 49, 50, 55, 46, 48, 46, 48, 46, 49, 0, 0, 1, 112, 108, 97, 121, 101, 114, 95, 0, 0, 98, 97, 114, 110, 101, 121, 103, 97, 108, 101, 0, 86, 105, 118, 97, 108, 97, 104, 101, 108, 118, 105, 103, 0, 0}...)

		fmt.Printf("writing: %s\n", res)
		listenerUDP.WriteTo(res, addr)
	}
}
