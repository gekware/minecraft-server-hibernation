package debugctrl

import (
	"fmt"
	"log"
	"strings"
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
			Log(fmt.Sprintf("data/s: %8.3f KB/s to clients | %8.3f KB/s to server", BytesToClients/1024, BytesToServer/1024))
			BytesToClients = 0
			BytesToServer = 0
		}
	})

	time.AfterFunc(1*time.Second, func() { PrintDataUsage() })
}

// Log prints the args if debug option is set to true
func Log(args ...interface{}) {
	if Debug {
		log.Println(args...)
	}
}

func Boxify(strList []string) string {
	// find longest string in list
	max := 0
	for _, l := range strList {
		if len(l) > max {
			max = len(l)
		}
	}

	// text box generation
	textBox := ""
	textBox += "╔═" + strings.Repeat("═", max) + "═╗" + "\n"
	for _, l := range strList {
		textBox += "║ " + l + strings.Repeat(" ", max-len(l)) + " ║" + "\n"
	}
	textBox += "╚═" + strings.Repeat("═", max) + "═╝"

	return textBox
}
