package errco

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"golang.org/x/sys/windows"
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

// ------------------- colors ------------------ //

var (
	COLOR_RESET  = "\033[0m"
	COLOR_GRAY   = "\033[1;30m" // used for server
	COLOR_RED    = "\033[0;31m" // used for errors
	COLOR_GREEN  = "\033[0;32m"
	COLOR_YELLOW = "\033[0;33m" // used for commands
	COLOR_BLUE   = "\033[0;34m"
	COLOR_PURPLE = "\033[0;35m"
	COLOR_CYAN   = "\033[0;36m" // used for important logs
)

func init() {
	// enable virtual terminal processing to enable colors on windows terminal
	if runtime.GOOS == "windows" {
		stdout := windows.Handle(os.Stdout.Fd())
		var originalMode uint32
		if err := windows.GetConsoleMode(stdout, &originalMode); err != nil {
			LogMshErr(NewErr(ERROR_COLOR_ENABLE, LVL_D, "errco init", "error while enabling colors on terminal"))
		} else if windows.SetConsoleMode(stdout, originalMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
			LogMshErr(NewErr(ERROR_COLOR_ENABLE, LVL_D, "errco init", "error while enabling colors on terminal"))
		}
	}
}

// Logln prints the args if debug option is set to true
func Logln(lvl int, s string, args ...interface{}) {
	if lvl <= DebugLvl {
		var logType string
		switch lvl {
		case LVL_C:
			logType = "serv"
		case LVL_E:
			logType = "byte"
		default:
			logType = "info"
		}

		header := fmt.Sprintf("%s [%s%s%s  %-4s]", time.Now().Format("2006/01/02 15:04:05"), COLOR_BLUE, logType, COLOR_RESET, strings.Repeat("*", 4-lvl))

		// make important logs more visible
		if lvl == LVL_A {
			s = COLOR_CYAN + s + COLOR_RESET
		}

		fmt.Printf(header+" "+s+"\n", args...)
	}
}

func LogMshErr(errMsh *Error) {
	if errMsh.Lvl <= DebugLvl {
		header := fmt.Sprintf("%s [%serror %s%-4s]", time.Now().Format("2006/01/02 15:04:05"), COLOR_RED, COLOR_RESET, strings.Repeat("*", 4-errMsh.Lvl))
		fmt.Printf(header + " " + errMsh.Ori + ": " + errMsh.Str + "\n")
	}
}
