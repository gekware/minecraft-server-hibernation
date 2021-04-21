package debugctrl

import (
	"fmt"
	"log"
	"time"

	"msh/lib/asyncctrl"
)

// BytesToClients tracks bytes/s server->clients
var BytesToClients float64 = 0

// BytesToServer tracks bytes/s clients->server
var BytesToServer float64 = 0

// Debug specify if debug should be printed or not
var Debug bool = false

// PrintDataUsage prints each second bytes/s to clients and to server.
// [goroutine]
func PrintDataUsage() {
	asyncctrl.WithLock(func() {
		if BytesToClients != 0 || BytesToServer != 0 {
			Logln(fmt.Sprintf("data/s: %8.3f KB/s to clients | %8.3f KB/s to server", BytesToClients/1024, BytesToServer/1024))
			BytesToClients = 0
			BytesToServer = 0
		}
	})

	time.AfterFunc(1*time.Second, func() { PrintDataUsage() })
}

// Logln prints the args if debug option is set to true
func Logln(args ...interface{}) {
	if Debug {
		log.Println(args...)
	}
}
