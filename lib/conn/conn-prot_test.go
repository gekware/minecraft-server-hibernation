package conn

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"

	"msh/lib/config"
	"msh/lib/errco"
)

type test struct {
	title   string
	packets [][]byte
	wait    time.Duration
	expect  interface{}
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
	}

	// open a listener and read request type for each new connection
	go func() {
		listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", 25555))
		if err != nil {
			t.Errorf("%s\n", err.Error())
		}

		for _, test := range tests {
			clientConn, err := listener.Accept()
			if err != nil {
				t.Errorf("%s\n", err.Error())
				continue
			}

			_, reqType, logMsh := getReqType(clientConn)
			if logMsh != nil {
				t.Errorf(logMsh.Mex, logMsh.Arg...)
			}

			if reqType != test.expect.(int) {
				t.Errorf("\treceived request is different from expected\n")
			}
		}
	}()

	for _, test := range tests {
		fmt.Printf("testing \"%s\"\n", test.title)
		serverSocket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", 25555))
		if err != nil {
			t.Errorf("%s\n", err.Error())
		}

		for _, packet := range test.packets {
			serverSocket.Write(packet)
			time.Sleep(test.wait)
		}

		serverSocket.Close()
		time.Sleep(100 * time.Millisecond)
	}
}

func Test_getPing(t *testing.T) {
	// set port which was used to get hardcoded test bytes
	config.MshPort = 25555

	tests := []test{
		// positive cases
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

	// emulate msh ping response
	go func() {
		listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", 25555))
		if err != nil {
			t.Errorf("%s\n", err.Error())
		}

		for {
			clientConn, err := listener.Accept()
			if err != nil {
				t.Errorf("%s\n", err.Error())
				continue
			}

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
			t.Errorf("%s\n", err.Error())
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
