package logger

import (
	"log"
)

// Debug specify if debug should be printed or not
// (default is true so it will log errors before logging the config)
var Debug bool = true

// Logln prints the args if debug option is set to true
func Logln(args ...interface{}) {
	if Debug {
		log.Println(args...)
	}
}
