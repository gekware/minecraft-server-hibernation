package servconn

import (
	"io"
	"net"
	"strings"
	"time"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/servctrl"
)

// forward takes a source and a destination net.Conn and forwards them.
// (isServerToClient used to know the forward direction).
// [goroutine]
func forward(source, destination net.Conn, isServerToClient bool, stopC chan bool) {
	data := make([]byte, 1024)

	// set to false after the first for cycle
	firstBuf := true

	for {
		// if stopC receives true, close the source connection, otherwise continue
		select {
		case <-stopC:
			source.Close()
			return
		default:
		}

		// update read and write timeout
		source.SetReadDeadline(time.Now().Add(time.Duration(config.ConfigRuntime.Msh.TimeBeforeStoppingEmptyServer) * time.Second))
		destination.SetWriteDeadline(time.Now().Add(time.Duration(config.ConfigRuntime.Msh.TimeBeforeStoppingEmptyServer) * time.Second))

		// read data from source
		dataLen, err := source.Read(data)
		if err != nil {
			// case in which the connection is closed by the source or closed by target
			if err == io.EOF {
				errco.Logln(errco.LVL_D, "forward: closing %15s --> %15s because of: %s", strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0], err.Error())
			} else {
				errco.Logln(errco.LVL_D, "forward: %v\n%15s --> %15s", err, strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0])
			}

			// close the source connection
			stopC <- true
			source.Close()
			return
		}

		// write data to destination
		destination.Write(data[:dataLen])

		// calculate bytes/s to client/server
		if errco.DebugLvl >= errco.LVL_D {
			servctrl.Stats.M.Lock()
			if isServerToClient {
				servctrl.Stats.BytesToClients = servctrl.Stats.BytesToClients + float64(dataLen)
			} else {
				servctrl.Stats.BytesToServer = servctrl.Stats.BytesToServer + float64(dataLen)
			}
			servctrl.Stats.M.Unlock()
		}

		// version/protocol are only found in serverToClient connection in the first buffer that is read
		if firstBuf && isServerToClient {
			errMsh := extractVersionProtocol(data[:dataLen])
			if errMsh != nil {
				errco.LogMshErr(errMsh.AddTrace("forward"))
			}

			// first cycle is finished, set firstBuf = false
			firstBuf = false
		}
	}
}
