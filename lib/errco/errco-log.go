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

func Logln(t LogTyp, l LogLvl, c LogCod, m string, a ...interface{}) {
	Log(&MshLog{trace(), t, l, c, m, a})
}

func Log(log *MshLog) {
	if log.Lvl > DebugLvl {
		return
	}

	// set log colors depending on log type
	t := ""
	switch log.Typ {
	case TYPE_INF:
		t = COLOR_BLUE + string(log.Typ) + COLOR_RESET
	case TYPE_SER:
		t = COLOR_GRAY + string(log.Typ) + COLOR_RESET
		log.Mex = COLOR_GRAY + log.Mex + COLOR_RESET
	case TYPE_BYT:
		t = COLOR_PURPLE + string(log.Typ) + COLOR_RESET
	case TYPE_WAR:
		t = COLOR_YELLOW + string(log.Typ) + COLOR_RESET
	case TYPE_ERR:
		t = COLOR_RED + string(log.Typ) + COLOR_RESET
	}

	// set log colors depending on log level
	switch log.Lvl {
	case LVL_0:
		// make important logs more visible
		log.Mex = COLOR_CYAN + log.Mex + COLOR_RESET
	}

	header := fmt.Sprintf("%s [%-16s  %-4s]", time.Now().Format("2006/01/02 15:04:05"), t, strings.Repeat("≡", 4-int(log.Lvl)))

	// print line layout depending on log type
	switch log.Typ {
	case TYPE_INF:
		fmt.Printf(header+" "+log.Mex+"\n", log.Arg...)
	case TYPE_SER:
		fmt.Printf(header+" "+log.Mex+"\n", log.Arg...)
	case TYPE_BYT:
		fmt.Printf(header+" "+log.Mex+"\n", log.Arg...)
	case TYPE_WAR:
		fmt.Printf(header+" "+string(log.Ori)+": "+log.Mex+" ["+fmt.Sprintf("%08x", log.Cod)+"]\n", log.Arg...)
	case TYPE_ERR:
		fmt.Printf(header+" "+string(log.Ori)+": "+log.Mex+" ["+fmt.Sprintf("%08x", log.Cod)+"]\n", log.Arg...)
	}
}
