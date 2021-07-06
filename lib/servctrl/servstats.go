package servctrl

import (
	"fmt"
	"sync"
	"time"

	"msh/lib/debugctrl"
)

type serverStats struct {
	M *sync.Mutex
	// ServerStatus represent the status of the minecraft server ("offline", "starting", "online", "stopping")
	Status string
	// PlayerCount keeps track of players connected to the server
	PlayerCount int
	// StopMSRequests keeps track of how many times StopMSRequest() has been called in the last {TimeBeforeStoppingEmptyServer} seconds.
	// (It's an int32 variable to allow for atomic operations)
	StopMSRequests int32
	// LoadProgress indicates the loading percentage while the server is starting
	LoadProgress string
	// BytesToClients tracks bytes/s server->clients
	BytesToClients float64
	// BytesToServer tracks bytes/s clients->server
	BytesToServer float64
}

// Stats contains the info relative to server
var Stats *serverStats

func init() {
	Stats = &serverStats{
		M:              &sync.Mutex{},
		Status:         "offline",
		PlayerCount:    0,
		StopMSRequests: 0,
		LoadProgress:   "0%",
		BytesToClients: 0,
		BytesToServer:  0,
	}
}

// PrintDataUsage prints each second bytes/s to clients and to server.
// (must be launched after ServTerm.IsActive has been set to true)
// [goroutine]
func printDataUsage() {
	for ServTerm.IsActive {
		if Stats.BytesToClients != 0 || Stats.BytesToServer != 0 {
			debugctrl.Logln(fmt.Sprintf("data/s: %8.3f KB/s to clients | %8.3f KB/s to server", Stats.BytesToClients/1024, Stats.BytesToServer/1024))

			Stats.M.Lock()
			Stats.BytesToClients = 0
			Stats.BytesToServer = 0
			Stats.M.Unlock()
		}

		time.Sleep(time.Second)
	}
}
