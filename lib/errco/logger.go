package errco

import (
	"log"
)

// DebugLvl specify the level of debugging
// (default is LVL_E so it will log everything)
var DebugLvl int = LVL_E

const (
	LVL_A = 0 // NONE: no log
	LVL_B = 1 // BASE: basic log
	LVL_C = 2 // SERV: mincraft server log
	LVL_D = 3 // DEVE: developement log
	LVL_E = 4 // BYTE: connection bytes log
)

// Logln prints the args if debug option is set to true
func Logln(lvl int, args ...interface{}) {
	if lvl <= DebugLvl {
		log.Println(args...)
	}
}

func LogMshErr(errMsh *Error) {
	if errMsh.Lvl <= DebugLvl {
		log.Println(errMsh.Ori + ": " + errMsh.Str)
	}
}
