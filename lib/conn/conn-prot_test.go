package conn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/model"
	"msh/lib/utility"
)

type test struct {
	title   string
	packets [][]byte
	wait    time.Duration
	expect  interface{}
}

// varInt encodes v as a minecraft protocol VarInt
func varInt(v int) []byte {
	byt := []byte{}

	for {
		if v&^0x7f == 0 {
			return append(byt, byte(v))
		}
		byt = append(byt, byte(v&0x7f|0x80))
		v >>= 7
	}
}

// mountHandshake builds a minecraft handshake packet:
// [ packet length (VarInt) | packet id (VarInt, 0) | protocol version (VarInt) | server address (String) | server port (unsigned short) | next state (VarInt) ]
func mountHandshake(protocol int, address string, port uint16, nextState int) []byte {
	data := []byte{0} // packet id
	data = append(data, varInt(protocol)...)
	data = append(data, varInt(len(address))...)
	data = append(data, []byte(address)...)
	data = append(data, byte(port>>8), byte(port))
	data = append(data, varInt(nextState)...)

	return append(varInt(len(data)), data...)
}

// splitAt splits data in fragments at the specified offsets
// (to emulate a packet arriving in more than one tcp segment)
func splitAt(data []byte, offsets ...int) [][]byte {
	fragments := [][]byte{}

	prev := 0
	for _, offset := range offsets {
		if offset <= prev || offset >= len(data) {
			continue
		}
		fragments = append(fragments, data[prev:offset])
		prev = offset
	}

	return append(fragments, data[prev:])
}

// splitBytewise splits data in fragments of 1 byte
func splitBytewise(data []byte) [][]byte {
	fragments := make([][]byte, 0, len(data))

	for i := range data {
		fragments = append(fragments, data[i:i+1])
	}

	return fragments
}

func Test_getReqType(t *testing.T) {
	// set port which was used to get hardcoded test bytes
	config.MshPort = 25555

	tests := []test{
		{
			"client info request (1.18.2 local)",
			[][]byte{
				{16, 0, 246, 5, 9, 49, 50, 55, 46, 48, 46, 48, 46, 49, 99, 211, 1},
			},
			0,
			errco.CLIENT_REQ_INFO,
		},
		{
			"client info request (1.18.2 local)",
			[][]byte{
				{16, 0, 246, 5, 9, 49, 50, 55, 46, 48, 46, 48, 46, 49, 99, 211, 1, 1, 0},
			},
			0,
			errco.CLIENT_REQ_INFO,
		},
		{
			"client join request (1.18.2 local) [1,2]",
			[][]byte{
				{33, 0, 246, 5, 26, 107, 117, 98, 101, 114, 110, 101, 116, 101, 115, 46, 100, 111, 99, 107, 101, 114, 46, 105, 110, 116, 101, 114, 110, 97, 108, 99, 211, 2},
				{11, 0, 9, 103, 101, 107, 105, 103, 101, 107, 57, 57},
			},
			0,
			errco.CLIENT_REQ_JOIN,
		},
		{
			"client join request (1.18.2 local)",
			[][]byte{
				{33, 0, 246, 5, 26, 107, 117, 98, 101, 114, 110, 101, 116, 101, 115, 46, 100, 111, 99, 107, 101, 114, 46, 105, 110, 116, 101, 114, 110, 97, 108, 99, 211, 2, 11, 0, 9, 103, 101, 107, 105, 103, 101, 107, 57, 57},
			},
			0,
			errco.CLIENT_REQ_JOIN,
		},
		{
			"client info request (1.19.3 local)",
			[][]byte{
				{16, 0, 249, 5, 9, 49, 50, 55, 46, 48, 46, 48, 46, 49, 99, 211, 1},
			},
			0,
			errco.CLIENT_REQ_INFO,
		},
		{
			"client info request (1.19.3 local)",
			[][]byte{
				{16, 0, 249, 5, 9, 49, 50, 55, 46, 48, 46, 48, 46, 49, 99, 211, 1, 1, 0},
			},
			0,
			errco.CLIENT_REQ_INFO,
		},
		{
			"client join request (1.19.3 local) [1,2]",
			[][]byte{
				{33, 0, 249, 5, 26, 107, 117, 98, 101, 114, 110, 101, 116, 101, 115, 46, 100, 111, 99, 107, 101, 114, 46, 105, 110, 116, 101, 114, 110, 97, 108, 99, 211, 2},
				{28, 0, 9, 103, 101, 107, 105, 103, 101, 107, 57, 57, 1, 196, 93, 252, 169, 146, 189, 69, 1, 169, 208, 156, 201, 205, 197, 2, 113},
			},
			0,
			errco.CLIENT_REQ_JOIN,
		},
		{
			"client join request (1.19.3 local) [1,.....2]",
			[][]byte{
				{33, 0, 249, 5, 26, 107, 117, 98, 101, 114, 110, 101, 116, 101, 115, 46, 100, 111, 99, 107, 101, 114, 46, 105, 110, 116, 101, 114, 110, 97, 108, 99, 211, 2},
				{28, 0, 9, 103, 101, 107, 105, 103, 101, 107, 57, 57, 1, 196, 93, 252, 169, 146, 189, 69, 1, 169, 208, 156, 201, 205, 197, 2, 113},
			},
			500 * time.Millisecond,
			errco.CLIENT_REQ_JOIN,
		},
		{
			"client join request (1.19.3 local)",
			[][]byte{
				{33, 0, 249, 5, 26, 107, 117, 98, 101, 114, 110, 101, 116, 101, 115, 46, 100, 111, 99, 107, 101, 114, 46, 105, 110, 116, 101, 114, 110, 97, 108, 99, 211, 2, 28, 0, 9, 103, 101, 107, 105, 103, 101, 107, 57, 57, 1, 196, 93, 252, 169, 146, 189, 69, 1, 169, 208, 156, 201, 205, 197, 2, 113},
			},
			0,
			errco.CLIENT_REQ_JOIN,
		},

		// fragmented requests: tcp has no message boundaries, so a client packet
		// can arrive split in more than one tcp segment

		{
			// this is the packet that made msh log "client request unknown (received: [26])":
			// [26] is the packet length VarInt, the rest of the handshake arrives later
			"client info request fragmented after packet length",
			[][]byte{
				{26},
				{0, 246, 5, 19, 109, 105, 110, 101, 46, 101, 120, 97, 109, 112, 108, 101, 46, 99, 111, 109, 46, 98, 114, 99, 211, 1},
			},
			50 * time.Millisecond,
			errco.CLIENT_REQ_INFO,
		},
		{
			"client join request fragmented inside server address",
			append(
				splitAt(mountHandshake(761, "kubernetes.docker.internal", 25555, 2), 20),
				[]byte{11, 0, 9, 103, 101, 107, 105, 103, 101, 107, 57, 57},
			),
			50 * time.Millisecond,
			errco.CLIENT_REQ_JOIN,
		},
		{
			"client info request fragmented byte by byte",
			splitBytewise(mountHandshake(758, "127.0.0.1", 25555, 1)),
			10 * time.Millisecond,
			errco.CLIENT_REQ_INFO,
		},
		{
			// server address long enough to make the packet length VarInt 2 bytes long
			// (a packet length of more than 127 bytes can't be read as a single byte)
			"client info request with long server address, fragmented",
			splitAt(mountHandshake(758, strings.Repeat("sub.", 49)+"example.com", 25555, 1), 1, 100),
			50 * time.Millisecond,
			errco.CLIENT_REQ_INFO,
		},
		{
			// the server address is length prefixed and can contain any byte, msh port bytes
			// followed by the info request byte included: searching the packet for that
			// sequence classifies this join request as an info request
			"client join request with msh port flag inside server address",
			[][]byte{
				mountHandshake(761, "srv"+string([]byte{99, 211, 1})+".example.com", 25555, 2),
				{11, 0, 9, 103, 101, 107, 105, 103, 101, 107, 57, 57},
			},
			0,
			errco.CLIENT_REQ_JOIN,
		},
	}

	// check that the test helpers build the same bytes as the hardcoded packets above
	if handshake := mountHandshake(758, "127.0.0.1", 25555, 1); !bytes.Equal(handshake, []byte{16, 0, 246, 5, 9, 49, 50, 55, 46, 48, 46, 48, 46, 49, 99, 211, 1}) {
		t.Fatalf("mountHandshake built unexpected bytes: %v\n", handshake)
	}

	// open the listener before dialing, to avoid a race between listener and client
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", 25555))
	if err != nil {
		t.Fatalf("%s\n", err.Error())
	}

	var wg sync.WaitGroup
	defer wg.Wait()        // executed last: wait for the listener goroutine to return
	defer listener.Close() // executed first: unblocks the listener goroutine

	reqTypes := make([]int, len(tests))
	logsMsh := make([]*errco.MshLog, len(tests))

	// read request type for each new connection.
	// results are checked by the test goroutine, to avoid logging after the test has completed
	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := range tests {
			clientConn, err := listener.Accept()
			if err != nil {
				return
			}

			_, reqTypes[i], logsMsh[i] = getReqType(clientConn)

			clientConn.Close()
		}
	}()

	for _, test := range tests {
		fmt.Printf("testing \"%s\"\n", test.title)
		serverSocket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", 25555))
		if err != nil {
			t.Fatalf("%s\n", err.Error())
		}

		for _, packet := range test.packets {
			serverSocket.Write(packet)
			time.Sleep(test.wait)
		}

		serverSocket.Close()
		time.Sleep(100 * time.Millisecond)
	}

	wg.Wait()

	for i, test := range tests {
		if logsMsh[i] != nil {
			t.Errorf("\t\"%s\": %s\n", test.title, fmt.Sprintf(logsMsh[i].Mex, logsMsh[i].Arg...))
			continue
		}

		if reqTypes[i] != test.expect.(int) {
			t.Errorf("\t\"%s\": received request is different from expected (%d instead of %d)\n", test.title, reqTypes[i], test.expect.(int))
		}
	}
}

func Test_getPing(t *testing.T) {
	// set port which was used to get hardcoded test bytes
	config.MshPort = 25555

	tests := []test{
		// positive cases
		{
			"ping",
			[][]byte{
				{9, 1, 0, 0, 0, 0, 0, 89, 73, 114},
			},
			0,
			[]byte{9, 1, 0, 0, 0, 0, 0, 89, 73, 114},
		},
		{
			"2 bytes + ping",
			[][]byte{
				{1, 0, 9, 1, 0, 0, 0, 0, 0, 89, 73, 114},
			},
			0,
			[]byte{9, 1, 0, 0, 0, 0, 0, 89, 73, 114},
		},
		{
			"2 bytes, ping",
			[][]byte{
				{1, 0},
				{9, 1, 0, 0, 0, 0, 0, 89, 73, 114},
			},
			0,
			[]byte{9, 1, 0, 0, 0, 0, 0, 89, 73, 114},
		},
		{
			"2 bytes, sleep, ping",
			[][]byte{
				{1, 0},
				{9, 1, 0, 0, 0, 0, 0, 89, 73, 114},
			},
			100 * time.Millisecond,
			[]byte{9, 1, 0, 0, 0, 0, 0, 89, 73, 114},
		},

		// negative cases
		{
			"1 bytes, sleep, ping -> expected client timeout",
			[][]byte{
				{1},
				{9, 1, 0, 0, 0, 0, 0, 89, 73, 114},
			},
			100 * time.Millisecond,
			nil,
		},
		{
			"1 bytes different + ping -> expected client timeout",
			[][]byte{
				{5, 9, 1, 0, 0, 0, 0, 0, 89, 73, 114},
			},
			0,
			nil,
		},
		{
			"1 bytes different, sleep, ping -> expected client timeout",
			[][]byte{
				{5},
				{9, 1, 0, 0, 0, 0, 0, 89, 73, 114},
			},
			100 * time.Millisecond,
			nil,
		},
		{
			"2 bytes different, sleep, ping -> expected client timeout",
			[][]byte{
				{5, 6},
				{9, 1, 0, 0, 0, 0, 0, 89, 73, 114},
			},
			100 * time.Millisecond,
			nil,
		},
	}

	// open the listener before dialing, to avoid a race between listener and client
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", 25555))
	if err != nil {
		t.Fatalf("%s\n", err.Error())
	}

	var wg sync.WaitGroup
	defer wg.Wait()        // executed last: wait for the listener goroutine to return
	defer listener.Close() // executed first: unblocks the listener goroutine

	// emulate msh ping response
	wg.Add(1)
	go func() {
		defer wg.Done()

		for range tests {
			clientConn, err := listener.Accept()
			if err != nil {
				return
			}

			// the connection is not closed here: the negative test cases expect
			// the client to time out while reading, not to receive an EOF
			logMsh := getPing(clientConn)
			if logMsh != nil {
				logMsh.Log(true)
			}
		}
	}()

	for _, test := range tests {
		fmt.Printf("\ntesting \"%s\": %v\n", test.title, test.packets)
		serverSocket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", 25555))
		if err != nil {
			t.Fatalf("%s\n", err.Error())
		}

		for _, packet := range test.packets {
			serverSocket.Write(packet)
			time.Sleep(test.wait)
		}

		buf := make([]byte, 1024)
		serverSocket.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		n, err := serverSocket.Read(buf)
		if err != nil {
			// if timeout and it's expected continue
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() && test.expect == nil {
				fmt.Printf("\tclient will timeout on ping\n")
				serverSocket.Close()
				continue
			}
			t.Errorf("%s\n", err.Error())
		}

		fmt.Printf("\tclient receives: %v\n", buf[:n])

		if !bytes.Equal(buf[:n], test.expect.([]byte)) {
			t.Errorf("\tclient received different bytes from expected\n")
		}

		serverSocket.Close()
	}
}

func Test_getClientPacket(t *testing.T) {
	// packets that getClientPacket must return, whatever the way they are written by the client
	handshake := mountHandshake(758, "127.0.0.1", 25555, 1)
	handshakeBig := mountHandshake(758, strings.Repeat("a", 2000), 25555, 1)

	tests := []test{
		{
			"complete packet in one write",
			[][]byte{{9, 1, 0, 0, 0, 0, 0, 89, 73, 114}},
			0,
			[]byte{9, 1, 0, 0, 0, 0, 0, 89, 73, 114},
		},
		{
			"packet written in fragments",
			splitAt(handshake, 1, 5),
			50 * time.Millisecond,
			handshake,
		},
		{
			"packet written byte by byte",
			splitBytewise(handshake),
			10 * time.Millisecond,
			handshake,
		},
		{
			// a fixed 1024 bytes buffer would truncate this packet
			"packet bigger than 1024 bytes",
			[][]byte{handshakeBig},
			0,
			handshakeBig,
		},
		{
			// the bytes following the packet belong to the next packet:
			// they must be left on the socket for the caller / minecraft server
			"packet followed by an other packet",
			[][]byte{append(append([]byte{}, handshake...), 1, 0)},
			0,
			handshake,
		},

		// negative cases
		{
			// msh must not allocate a buffer of the size declared by the client
			"declared packet length out of range",
			[][]byte{varInt(maxPacketLen + 1)},
			0,
			nil,
		},
	}

	// open the listener before dialing, to avoid a race between listener and client
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", 25555))
	if err != nil {
		t.Fatalf("%s\n", err.Error())
	}

	var wg sync.WaitGroup
	defer wg.Wait()        // executed last: wait for the listener goroutine to return
	defer listener.Close() // executed first: unblocks the listener goroutine

	packets := make([][]byte, len(tests))
	logsMsh := make([]*errco.MshLog, len(tests))

	// read one packet for each new connection
	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := range tests {
			clientConn, err := listener.Accept()
			if err != nil {
				return
			}

			packets[i], logsMsh[i] = getClientPacket(clientConn)

			clientConn.Close()
		}
	}()

	for _, test := range tests {
		fmt.Printf("testing \"%s\"\n", test.title)
		serverSocket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", 25555))
		if err != nil {
			t.Fatalf("%s\n", err.Error())
		}

		for _, packet := range test.packets {
			serverSocket.Write(packet)
			time.Sleep(test.wait)
		}

		serverSocket.Close()
		time.Sleep(100 * time.Millisecond)
	}

	wg.Wait()

	for i, test := range tests {
		// expect nil means that getClientPacket must return an error
		if test.expect == nil {
			if logsMsh[i] == nil {
				t.Errorf("\t\"%s\": expected an error, received packet %v\n", test.title, packets[i])
			}
			continue
		}

		if logsMsh[i] != nil {
			t.Errorf("\t\"%s\": %s\n", test.title, fmt.Sprintf(logsMsh[i].Mex, logsMsh[i].Arg...))
			continue
		}

		if !bytes.Equal(packets[i], test.expect.([]byte)) {
			t.Errorf("\t\"%s\": received packet is different from expected\n\treceived: %v\n\texpected: %v\n", test.title, packets[i], test.expect.([]byte))
		}
	}
}

// Test_answerClient checks that a failed answer to the client is reported
// instead of being silently discarded
func Test_answerClient(t *testing.T) {
	mes := []byte{9, 1, 0, 0, 0, 0, 0, 89, 73, 114}

	// open the listener before dialing, to avoid a race between listener and client
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", 25555))
	if err != nil {
		t.Fatalf("%s\n", err.Error())
	}
	defer listener.Close()

	// positive case: the client receives the message and no log is returned
	serverSocket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", 25555))
	if err != nil {
		t.Fatalf("%s\n", err.Error())
	}

	clientConn, err := listener.Accept()
	if err != nil {
		t.Fatalf("%s\n", err.Error())
	}

	if logMsh := answerClient(clientConn, mes); logMsh != nil {
		t.Errorf("\tanswer to the client failed: %s\n", fmt.Sprintf(logMsh.Mex, logMsh.Arg...))
	}

	buf := make([]byte, len(mes))
	serverSocket.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := io.ReadFull(serverSocket, buf); err != nil {
		t.Errorf("\tclient did not receive the answer: %s\n", err.Error())
	} else if !bytes.Equal(buf, mes) {
		t.Errorf("\tclient received different bytes from expected\n\treceived: %v\n\texpected: %v\n", buf, mes)
	}

	serverSocket.Close()

	// negative case: answering on a closed connection must be reported
	clientConn.Close()

	if logMsh := answerClient(clientConn, mes); logMsh == nil {
		t.Errorf("\tanswer to the client on a closed connection did not return a log\n")
	}
}

// Test_buildMessage checks that the header msh mounts on its answers is a valid
// minecraft packet whatever the message length
func Test_buildMessage(t *testing.T) {
	tests := []struct {
		title   string
		message string
	}{
		{"short message", "server is hibernating"},
		{"message just below the 2 bytes length limit", strings.Repeat("a", 16000)},
		// a fixed 2 bytes length field overflows here: the second byte gets the
		// continuation bit set and the client waits for a third byte that never comes
		{"message above the 2 bytes length limit", strings.Repeat("a", 20000)},
	}

	for _, test := range tests {
		fmt.Printf("testing \"%s\"\n", test.title)

		packet := buildMessage(errco.CLIENT_REQ_JOIN, test.message)

		// packet length
		packetLen, n, logMsh := utility.ParseVarInt(packet, 0)
		if logMsh != nil {
			t.Errorf("\t\"%s\": %s\n", test.title, fmt.Sprintf(logMsh.Mex, logMsh.Arg...))
			continue
		}
		if n+packetLen != len(packet) {
			t.Errorf("\t\"%s\": packet length field (%d) does not match the packet (%d bytes of data)\n", test.title, packetLen, len(packet)-n)
			continue
		}
		offset := n

		// packet id
		packetID, n, logMsh := utility.ParseVarInt(packet, offset)
		if logMsh != nil {
			t.Errorf("\t\"%s\": %s\n", test.title, fmt.Sprintf(logMsh.Mex, logMsh.Arg...))
			continue
		}
		if packetID != 0 {
			t.Errorf("\t\"%s\": packet id is %d instead of 0\n", test.title, packetID)
			continue
		}
		offset += n

		// json length
		jsonLen, n, logMsh := utility.ParseVarInt(packet, offset)
		if logMsh != nil {
			t.Errorf("\t\"%s\": %s\n", test.title, fmt.Sprintf(logMsh.Mex, logMsh.Arg...))
			continue
		}
		offset += n
		if offset+jsonLen != len(packet) {
			t.Errorf("\t\"%s\": json length field (%d) does not match the remaining bytes (%d)\n", test.title, jsonLen, len(packet)-offset)
			continue
		}

		// the json must contain the original message
		dataTxt := &model.DataTxt{}
		if err := json.Unmarshal(packet[offset:], dataTxt); err != nil {
			t.Errorf("\t\"%s\": json is not valid: %s\n", test.title, err.Error())
			continue
		}
		if dataTxt.Text != test.message {
			t.Errorf("\t\"%s\": message in the json is different from expected\n", test.title)
		}
	}
}
