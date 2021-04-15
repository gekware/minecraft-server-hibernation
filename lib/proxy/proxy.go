package proxy

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"msh/lib/asyncctrl"
	"msh/lib/confctrl"
	"msh/lib/debugctrl"
	"msh/lib/servprotocol"
)

// Forward takes a source and a destination net.Conn and forwards them.
// (isServerToClient used to know the forward direction)
func Forward(source, destination net.Conn, isServerToClient bool, stopSig *bool) {
	data := make([]byte, 1024)

	// set to false after the first for cycle
	firstBuffer := true

	for {
		if *stopSig {
			// if stopSig is true, close the source connection
			source.Close()
			break
		}

		// update read and write timeout
		source.SetReadDeadline(time.Now().Add(time.Duration(confctrl.Config.Msh.TimeBeforeStoppingEmptyServer) * time.Second))
		destination.SetWriteDeadline(time.Now().Add(time.Duration(confctrl.Config.Msh.TimeBeforeStoppingEmptyServer) * time.Second))

		// read data from source
		dataLen, err := source.Read(data)
		if err != nil {
			// case in which the connection is closed by the source or closed by target
			if err == io.EOF {
				debugctrl.Log(fmt.Sprintf("closing %s --> %s because of: %s", strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0], err.Error()))
			} else {
				debugctrl.Log(fmt.Sprintf("forwardSync: error in forward(): %v\n%s --> %s", err, strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0]))
			}

			// close the source connection
			asyncctrl.WithLock(func() { *stopSig = true })
			source.Close()
			break
		}

		// write data to destination
		destination.Write(data[:dataLen])

		// calculate bytes/s to client/server
		if confctrl.Config.Msh.Debug {
			asyncctrl.WithLock(func() {
				if isServerToClient {
					debugctrl.DataCountBytesToClients = debugctrl.DataCountBytesToClients + float64(dataLen)
				} else {
					debugctrl.DataCountBytesToServer = debugctrl.DataCountBytesToServer + float64(dataLen)
				}
			})
		}

		// version/protocol are only found in serverToClient connection in the first buffer that is read
		if firstBuffer && isServerToClient {
			servprotocol.SearchVersionProtocol(data[:dataLen])

			// first cycle is finished, set firstBuffer = false
			firstBuffer = false
		}
	}
}
