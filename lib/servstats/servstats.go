package servstats

import (
	"sync"
	"time"

	"msh/lib/errco"
)

// Stats contains the info relative to server
var Stats *serverStats = &serverStats{
	M:                  &sync.Mutex{},
	Status:             errco.SERVER_STATUS_OFFLINE,
	Suspended:          false,
	SuspendRefreshTick: time.NewTicker(5 * time.Minute),
	MajorError:         nil,
	PlayerCount:        0,
	FreezeTimer:        time.NewTimer(5 * time.Minute),
	LoadProgress:       "0%",
	BytesToClients:     0,
	BytesToServer:      0,
}

type serverStats struct {
	M                  *sync.Mutex
	Status             int           // represent the status of the minecraft server
	Suspended          bool          // status of minecraft server process (if ms is offline, should be set to false)
	SuspendRefreshTick *time.Ticker  // ticker that causes the refresh of minecraft server process suspension
	MajorError         *errco.MshLog // if !nil the server is having some major problems
	PlayerCount        int           // tracks players connected to the server
	FreezeTimer        *time.Timer   // timer to freeze minecraft server
	LoadProgress       string        // tracks loading percentage of starting server
	BytesToClients     float64       // tracks bytes/s server->clients
	BytesToServer      float64       // tracks bytes/s clients->server
}

func init() {
	go printDataUsage()
}

// printDataUsage prints each second bytes/s to clients and to server.
// prints data exchanged by clients and server only when servctrl.ServTerm.IsActive
// Stats.BytesToClients and Stats.BytesToServer are only set when there are clients connected to the server
// [goroutine]
func printDataUsage() {
	ticker := time.NewTicker(time.Second)

	for {
		<-ticker.C

		if Stats.BytesToClients != 0 || Stats.BytesToServer != 0 {
			errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "data/s: %8.3f KB/s to clients | %8.3f KB/s to server", Stats.BytesToClients/1024, Stats.BytesToServer/1024)
			Stats.M.Lock()
			Stats.BytesToClients = 0
			Stats.BytesToServer = 0
			Stats.M.Unlock()
		}
	}
}

// SetMajorError sets *serverStats.MajorError only if nil
func (s *serverStats) SetMajorError(e *errco.MshLog) {
	if s.MajorError == nil {
		s.MajorError = e
	}
}
