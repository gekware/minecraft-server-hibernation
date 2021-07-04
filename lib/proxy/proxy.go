package proxy

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"msh/lib/confctrl"
	"msh/lib/debugctrl"
	"msh/lib/servctrl"
	"msh/lib/servprotocol"
)

// Forward takes a source and a destination net.Conn and forwards them.
// (isServerToClient used to know the forward direction).
// [goroutine]
func Forward(source, destination net.Conn, isServerToClient bool, stopC chan bool) {
	data := make([]byte, 1024)

	// set to false after the first for cycle
	firstBuffer := true

	for {
		// if stopC receives true, close the source connection, otherwise continue
		select {
		case <-stopC:
			source.Close()
			return
		default:
		}

		// update read and write timeout
		source.SetReadDeadline(time.Now().Add(time.Duration(confctrl.ConfigRuntime.Msh.TimeBeforeStoppingEmptyServer) * time.Second))
		destination.SetWriteDeadline(time.Now().Add(time.Duration(confctrl.ConfigRuntime.Msh.TimeBeforeStoppingEmptyServer) * time.Second))

		// read data from source
		dataLen, err := source.Read(data)
		if err != nil {
			// case in which the connection is closed by the source or closed by target
			if err == io.EOF {
				debugctrl.Logln(fmt.Sprintf("closing %s --> %s because of: %s", strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0], err.Error()))
			} else {
				debugctrl.Logln(fmt.Sprintf("forwardSync: error in forward(): %v\n%s --> %s", err, strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0]))
			}

			// close the source connection
			stopC <- true
			source.Close()
			return
		}

		// write data to destination
		destination.Write(data[:dataLen])

		// calculate bytes/s to client/server
		if debugctrl.Debug {
			servctrl.Stats.M.Lock()
			if isServerToClient {
				servctrl.Stats.BytesToClients = servctrl.Stats.BytesToClients + float64(dataLen)
			} else {
				servctrl.Stats.BytesToServer = servctrl.Stats.BytesToServer + float64(dataLen)
			}
			servctrl.Stats.M.Unlock()
		}

		// version/protocol are only found in serverToClient connection in the first buffer that is read
		if firstBuffer && isServerToClient {
			err = servprotocol.GetVersionProtocol(data[:dataLen])
			if err != nil {
				debugctrl.Logln("Forward:", err)
			}

			// first cycle is finished, set firstBuffer = false
			firstBuffer = false
		}
	}
}
