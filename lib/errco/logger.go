package errco

import (
	"log"
)

// Debug specify if debug should be printed or not
// (default is true so it will log errors before logging the config)
var Debug bool = true

var DebugLvl int = LVL_B

const (
	LVL_A = 0 // NONE: no log
	LVL_B = 1 // BASE: basic log type
	LVL_C = 2 // SERV: mincraft server log type
	LVL_D = 3 // DEVE: developement log type
	LVL_E = 4 // BYTE: connection bytes log type
)

// Logln prints the args if debug option is set to true
func Logln(args ...interface{}) {
	if Debug {
		log.Println(args...)
	}
}

func LogMshErr(errMsh *Error) {
	if errMsh.Lvl <= DebugLvl {
		log.Println(errMsh.Ori + ": " + errMsh.Str)
	}
}
