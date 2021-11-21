package servstats

import (
	"sync"

	"msh/lib/errco"
)

// Stats contains the info relative to server
var Stats *serverStats

type serverStats struct {
	M              *sync.Mutex
	Status         int     // represent the status of the minecraft server
	PlayerCount    int     // tracks players connected to the server
	StopMSRequests int32   // tracks active StopMSRequest() instances. (int32 for atomic operations)
	LoadProgress   string  // tracks loading percentage of starting server
	BytesToClients float64 // tracks bytes/s server->clients
	BytesToServer  float64 // tracks bytes/s clients->server
}

func init() {
	Stats = &serverStats{
		M:              &sync.Mutex{},
		Status:         errco.SERVER_STATUS_OFFLINE,
		PlayerCount:    0,
		StopMSRequests: 0,
		LoadProgress:   "0%",
		BytesToClients: 0,
		BytesToServer:  0,
	}
}
