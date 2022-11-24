package errco

import (
	"fmt"
	"strings"
	"time"
)

// DebugLvl specify the level of debugging
// (default is LVL_4 so it will log everything)
var DebugLvl LogLvl = LVL_4

const (
	COLOR_RESET  = "\033[0m"
	COLOR_GRAY   = "\033[1;30m"
	COLOR_RED    = "\033[0;31m"
	COLOR_GREEN  = "\033[0;32m"
	COLOR_YELLOW = "\033[0;33m"
	COLOR_BLUE   = "\033[0;34m"
	COLOR_PURPLE = "\033[0;35m"
	COLOR_CYAN   = "\033[0;36m"
)

// Log prints to terminal *MshLog.
//
// returns the original log for convenience.
// returns nil if msh log struct is nil
func (logO *MshLog) Log() *MshLog {
	// ------- operations on original log -------

	// return original log if log is nil
	if logO == nil {
		return logO
	}

	// return original log if log level is not high enough
	if logO.Lvl > DebugLvl {
		return logO
	}

	// make a copy of original log
	logC := *logO

	// -------- operations on copied log --------

	// set logC colors depending on logC level
	switch logC.Lvl {
	case LVL_0:
		// make important logs more visible
		logC.Mex = COLOR_CYAN + logC.Mex + COLOR_RESET
	}

	// set log colors depending on log type
	var t string
	switch logC.Typ {
	case TYPE_INF:
		t = COLOR_BLUE + string(logC.Typ) + COLOR_RESET
	case TYPE_SER:
		t = COLOR_GRAY + string(logC.Typ) + COLOR_RESET
		logC.Mex = COLOR_GRAY + logC.Mex + "\x00" + COLOR_RESET
	case TYPE_BYT:
		t = COLOR_PURPLE + string(logC.Typ) + COLOR_RESET
	case TYPE_WAR:
		t = COLOR_YELLOW + string(logC.Typ) + COLOR_RESET
	case TYPE_ERR:
		t = COLOR_RED + string(logC.Typ) + COLOR_RESET
	}

	switch logC.Typ {
	case TYPE_INF, TYPE_SER, TYPE_BYT:
		fmt.Printf("%s [%-16s %-4s] %s\n",
			time.Now().Format("2006/01/02 15:04:05"),
			t,
			strings.Repeat("≡", 4-int(logC.Lvl)),
			fmt.Sprintf(logC.Mex, logC.Arg...))
	case TYPE_WAR, TYPE_ERR:
		fmt.Printf("%s [%-16s %-4s] %s %s %s\n",
			time.Now().Format("2006/01/02 15:04:05"),
			t,
			strings.Repeat("≡", 4-int(logC.Lvl)),
			LogOri(COLOR_YELLOW)+logC.Ori+":"+LogOri(COLOR_RESET),
			fmt.Sprintf(logC.Mex, logC.Arg...),
			fmt.Sprintf("[%08x]", logC.Cod))
	}

	// return original log
	return logO
}
