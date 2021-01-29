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

// PrintDataUsage prints each second bytes/s to clients and to server
func PrintDataUsage() {
	asyncctrl.Mutex.Lock()
	if DataCountBytesToClients != 0 || DataCountBytesToServer != 0 {
		Logger(fmt.Sprintf("data/s: %8.3f KB/s to clients | %8.3f KB/s to server", DataCountBytesToClients/1024, DataCountBytesToServer/1024))
		DataCountBytesToClients = 0
		DataCountBytesToServer = 0
	}
	asyncctrl.Mutex.Unlock()
	time.AfterFunc(1*time.Second, func() { PrintDataUsage() })
}

// Logger prints the args if debug option is set to true
func Logger(args ...string) {
	log.Println(strings.Join(args, " "))
}
