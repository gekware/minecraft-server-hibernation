package errco

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// DebugLvl specify the level of debugging
// (default is LVL_4 so it will log everything)
var DebugLvl LogLvl = LVL_4

type MshLog struct {
	Ori LogOri        // log origin function
	Typ LogTyp        // log type
	Lvl LogLvl        // log debug level
	Cod LogCod        // log code
	Mex string        // log string
	Arg []interface{} // log args
}

type LogOri string
type LogTyp string
type LogLvl int
type LogCod int

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

// NewLog returns a new msh log object.
//
// When a function fails and returns msh log using NewLog, msh log type must be TYPE_ERR or TYPE_WAR.
// Find bad usage with reg exp: `return (.*)NewLog\((.*)TYPE_(?!ERR|WAR)`
func NewLog(t LogTyp, l LogLvl, c LogCod, m string, a ...interface{}) *MshLog {
	logMsh := &MshLog{trace(2), t, l, c, m, a}
	return logMsh
}

// NewLogln prints to terminal msh log struct and returns a new msh log struct.
//
// When a function fails it should not return msh log using NewLogln.
// There is the risk of printing 2 times the same error:
// the parent function should handle the logging of msh log struct
// Find bad usage with reg exp: `return (.*)NewLogln\(`
func NewLogln(t LogTyp, l LogLvl, c LogCod, m string, a ...interface{}) *MshLog {
	logMsh := &MshLog{trace(2), t, l, c, m, a}
	logMsh.Log()
	return logMsh
}

// Log prints to terminal msh log struct.
//
// returns the original log for convenience.
// returns nil if msh log struct is nil
func (log *MshLog) Log() *MshLog {
	// return original log if it's nil
	if log == nil {
		return log
	}

	// ------- operations on original log -------

	// add trace if Log() was not called by NewLogln()
	// 1) example()               -> Log() -> trace(2) : example
	// 2) example() -> NewLogln() -> trace(2)          : example
	//                            \> Log() -> trace(2) : NewLogln (!)
	// example 2:
	// - trace(2) from Log() results in "NewLogln",
	// - trace(3) from Log() results in "example" (but it's wrong as NewLogln() already set the correct trace)
	pc := trace(2)
	if pc != LogOri("NewLogln") {
		log.Ori = pc + LogOri(": ") + log.Ori
	}

	// return original log if log level is not high enough
	if log.Lvl > DebugLvl {
		return log
	}

	// make a copy of original log
	logMod := *log

	// -------- operations on copied log --------

	// set logC colors depending on logC level
	switch logMod.Lvl {
	case LVL_0:
		// make important logs more visible
		logMod.Mex = COLOR_CYAN + logMod.Mex + COLOR_RESET
	}

	// set log colors depending on log type
	var t string
	switch logMod.Typ {
	case TYPE_INF:
		t = COLOR_BLUE + string(logMod.Typ) + COLOR_RESET
	case TYPE_SER:
		t = COLOR_GRAY + string(logMod.Typ) + COLOR_RESET
		logMod.Mex = COLOR_GRAY + logMod.Mex + "\x00" + COLOR_RESET
	case TYPE_BYT:
		t = COLOR_PURPLE + string(logMod.Typ) + COLOR_RESET
	case TYPE_WAR:
		t = COLOR_YELLOW + string(logMod.Typ) + COLOR_RESET
	case TYPE_ERR:
		t = COLOR_RED + string(logMod.Typ) + COLOR_RESET
	}

	switch logMod.Typ {
	case TYPE_INF, TYPE_SER, TYPE_BYT:
		fmt.Printf("%s [%-16s %-4s] %s\n",
			time.Now().Format("2006/01/02 15:04:05"),
			t,
			strings.Repeat("≡", 4-int(logMod.Lvl)),
			fmt.Sprintf(logMod.Mex, logMod.Arg...))
	case TYPE_WAR, TYPE_ERR:
		fmt.Printf("%s [%-16s %-4s] %s %s %s\n",
			time.Now().Format("2006/01/02 15:04:05"),
			t,
			strings.Repeat("≡", 4-int(logMod.Lvl)),
			LogOri(COLOR_YELLOW)+logMod.Ori+":"+LogOri(COLOR_RESET),
			fmt.Sprintf(logMod.Mex, logMod.Arg...),
			fmt.Sprintf("[%08x]", logMod.Cod))
	}

	// return original log
	return log
}

// AddTrace adds the caller function to the msh log trace
func (log *MshLog) AddTrace() *MshLog {
	// return original log if it's nil
	if log == nil {
		return log
	}

	log.Ori = trace(2) + LogOri(": ") + log.Ori

	return log
}

// trace returns the function name the parent was called from
//
// skip == 2: example() -> NewLog() -> trace()
//
// result:	  example
func trace(skip int) LogOri {
	var o string = "?"

	if pc, _, _, ok := runtime.Caller(skip); !ok {
	} else if f := runtime.FuncForPC(pc); f == nil {
	} else {
		fn := f.Name()
		o = fn[strings.LastIndex(fn, ".")+1:]
	}

	return LogOri(o)
}
