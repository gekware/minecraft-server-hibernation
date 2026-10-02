package conn

import (
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// Test_HandlerClientConn_closesOnBadRequest checks that the client connection is closed
// when the request can't be understood.
//
// HandlerClientConn only reads the client address and calls getReqType before failing,
// so it can be called without setting up servstats / config.
func Test_HandlerClientConn_closesOnBadRequest(t *testing.T) {
	// a well framed packet that is not a handshake: packet length 5, packet id 9
	badRequest := []byte{5, 9, 1, 0, 0, 0}

	// open the listener before dialing, to avoid a race between listener and client
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", 25555))
	if err != nil {
		t.Fatalf("%s\n", err.Error())
	}

	var wg sync.WaitGroup
	defer wg.Wait()        // executed last: wait for the listener goroutine to return
	defer listener.Close() // executed first: unblocks the listener goroutine

	wg.Add(1)
	go func() {
		defer wg.Done()

		clientConn, err := listener.Accept()
		if err != nil {
			return
		}

		HandlerClientConn(clientConn)
	}()

	serverSocket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", 25555))
	if err != nil {
		t.Fatalf("%s\n", err.Error())
	}
	defer serverSocket.Close()

	serverSocket.Write(badRequest)

	// msh must close the connection: the read returns EOF instead of blocking.
	// the deadline is only a guard against the test hanging if the connection is leaked
	// (it must be longer than the time msh waits for a complete client packet)
	serverSocket.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, err = serverSocket.Read(make([]byte, 1))

	switch {
	case err == io.EOF:
		// connection closed by msh, as expected

	case err == nil:
		t.Errorf("\tmsh answered an unknown request instead of closing the connection\n")

	default:
		t.Errorf("\tclient connection was not closed by msh (%s)\n", err.Error())
	}
}
