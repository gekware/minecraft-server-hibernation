package conn

import (
	"fmt"
	"net"
	"testing"
	"time"

	"msh/lib/config"
	"msh/lib/errco"
)

func Test_getReqType(t *testing.T) {
	// set port which was used to get hardcoded test bytes
	config.MshPort = 25555

	// open a listener and read request type for each new connection
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

			_, reqType, logMsh := getReqType(clientConn)
			if logMsh != nil {
				t.Errorf(logMsh.Mex, logMsh.Arg...)
			}

			switch reqType {
			case errco.CLIENT_REQ_INFO:
				fmt.Printf("\t-> received info req\n\n")
			case errco.CLIENT_REQ_JOIN:
				fmt.Printf("\t-> received join req\n\n")
			default:
				t.Errorf("\t-> request unknown\n\n")
			}
		}
	}()

	type test struct {
		title   string
		packets [][]byte
		wait    time.Duration
	}

	tests := []test{
		{
			"client info request (1.18.2 local)",
			[][]byte{
				{16, 0, 246, 5, 9, 49, 50, 55, 46, 48, 46, 48, 46, 49, 99, 211, 1},
			},
			0,
		},
		{
			"client info request (1.18.2 local)",
			[][]byte{
				{16, 0, 246, 5, 9, 49, 50, 55, 46, 48, 46, 48, 46, 49, 99, 211, 1, 1, 0},
			},
			0,
		},
		{
			"client join request (1.18.2 local) [1,2]",
			[][]byte{
				{33, 0, 246, 5, 26, 107, 117, 98, 101, 114, 110, 101, 116, 101, 115, 46, 100, 111, 99, 107, 101, 114, 46, 105, 110, 116, 101, 114, 110, 97, 108, 99, 211, 2},
				{11, 0, 9, 103, 101, 107, 105, 103, 101, 107, 57, 57},
			},
			0,
		},
		{
			"client join request (1.18.2 local)",
			[][]byte{
				{33, 0, 246, 5, 26, 107, 117, 98, 101, 114, 110, 101, 116, 101, 115, 46, 100, 111, 99, 107, 101, 114, 46, 105, 110, 116, 101, 114, 110, 97, 108, 99, 211, 2, 11, 0, 9, 103, 101, 107, 105, 103, 101, 107, 57, 57},
			},
			0,
		},
		{
			"client info request (1.19.3 local)",
			[][]byte{
				{16, 0, 249, 5, 9, 49, 50, 55, 46, 48, 46, 48, 46, 49, 99, 211, 1},
			},
			0,
		},
		{
			"client info request (1.19.3 local)",
			[][]byte{
				{16, 0, 249, 5, 9, 49, 50, 55, 46, 48, 46, 48, 46, 49, 99, 211, 1, 1, 0},
			},
			0,
		},
		{
			"client join request (1.19.3 local) [1,2]",
			[][]byte{
				{33, 0, 249, 5, 26, 107, 117, 98, 101, 114, 110, 101, 116, 101, 115, 46, 100, 111, 99, 107, 101, 114, 46, 105, 110, 116, 101, 114, 110, 97, 108, 99, 211, 2},
				{28, 0, 9, 103, 101, 107, 105, 103, 101, 107, 57, 57, 1, 196, 93, 252, 169, 146, 189, 69, 1, 169, 208, 156, 201, 205, 197, 2, 113},
			},
			0,
		},
		{
			"client join request (1.19.3 local) [1,.....2]",
			[][]byte{
				{33, 0, 249, 5, 26, 107, 117, 98, 101, 114, 110, 101, 116, 101, 115, 46, 100, 111, 99, 107, 101, 114, 46, 105, 110, 116, 101, 114, 110, 97, 108, 99, 211, 2},
				{28, 0, 9, 103, 101, 107, 105, 103, 101, 107, 57, 57, 1, 196, 93, 252, 169, 146, 189, 69, 1, 169, 208, 156, 201, 205, 197, 2, 113},
			},
			500 * time.Millisecond,
		},
		{
			"client join request (1.19.3 local)",
			[][]byte{
				{33, 0, 249, 5, 26, 107, 117, 98, 101, 114, 110, 101, 116, 101, 115, 46, 100, 111, 99, 107, 101, 114, 46, 105, 110, 116, 101, 114, 110, 97, 108, 99, 211, 2, 28, 0, 9, 103, 101, 107, 105, 103, 101, 107, 57, 57, 1, 196, 93, 252, 169, 146, 189, 69, 1, 169, 208, 156, 201, 205, 197, 2, 113},
			},
			0,
		},
	}

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
