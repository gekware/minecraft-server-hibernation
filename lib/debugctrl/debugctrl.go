package debugctrl

import (
	"fmt"
	"log"
	"strings"
	"time"

	"msh/lib/asyncctrl"
)

// DataCountBytesToClients tracks bytes/s server->clients
var DataCountBytesToClients float64 = 0

// DataCountBytesToServer tracks bytes/s clients->server
var DataCountBytesToServer float64 = 0

// Debug specify if debug should be printed or not
var Debug bool = false

// PrintDataUsage prints each second bytes/s to clients and to server
func PrintDataUsage() {
	asyncctrl.WithLock(func() {
		if DataCountBytesToClients != 0 || DataCountBytesToServer != 0 {
			Logger(fmt.Sprintf("data/s: %8.3f KB/s to clients | %8.3f KB/s to server", DataCountBytesToClients/1024, DataCountBytesToServer/1024))
			DataCountBytesToClients = 0
			DataCountBytesToServer = 0
		}
	})
	time.AfterFunc(1*time.Second, func() { PrintDataUsage() })
}

// Logger prints the args if debug option is set to true
func Logger(args ...string) {
	if Debug {
		log.Println(strings.Join(args, " "))
	}
}
