package servstats

import (
	"sync"
	"time"

	"msh/lib/errco"
)

// Stats contains the info relative to server
var Stats *serverStats = &serverStats{
	M:              &sync.Mutex{},
	Status:         errco.SERVER_STATUS_OFFLINE,
	Suspended:      false,
	MajorError:     nil,
	ConnCount:      0,
	FreezeTimer:    time.NewTimer(5 * time.Minute),
	WarmUpTime:     time.Unix(0, 0), // use 1970-01-01 00:00:00 as init value
	LoadProgress:   "0%",
	BytesToClients: 0,
	BytesToServer:  0,
}

type serverStats struct {
	M              *sync.Mutex
	Status         int           // represent the status of the minecraft server
	Suspended      bool          // status of minecraft server process (if ms is offline, should be set to false)
	MajorError     *errco.MshLog // if !nil the server is having some major problems
	ConnCount      int           // tracks active client connections to ms (only clients that are playing on ms)
	FreezeTimer    *time.Timer   // timer to freeze minecraft server
	WarmUpTime     time.Time     // time at which minecraft server was warmed up
	LoadProgress   string        // tracks loading percentage of starting server
	BytesToClients float64       // tracks bytes/s server->clients
	BytesToServer  float64       // tracks bytes/s clients->server
}

// SetMajorError sets *serverStats.MajorError only if nil
func (s *serverStats) SetMajorError(e *errco.MshLog) {
	if s.MajorError == nil {
		s.MajorError = e
	}
}
