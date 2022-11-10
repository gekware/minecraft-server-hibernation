package errco

import (
	"runtime"
	"strings"
)

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

// NewLog returns a new msh log object.
//
// When a function fails and returns using NewLog, msh log type must be TYPE_ERR or TYPE_WAR.
// Find bad usage with reg exp: `return (.*)NewLog(.*)TYPE_(?!ERR|WAR)`
func NewLog(t LogTyp, l LogLvl, c LogCod, m string, a ...interface{}) *MshLog {
	return &MshLog{trace(), t, l, c, m, a}
}

// AddTrace adds the caller function to the msh log trace
func (log *MshLog) AddTrace() *MshLog {
	log.Ori = trace() + LogOri(": ") + log.Ori
	return log
}

// trace returns the function name the parent was called from
//
// aaa() -> NewLog() -> trace() = aaa
func trace() LogOri {
	o := "?"
	if pc, _, _, ok := runtime.Caller(2); !ok { // 2: returns caller of caller
	} else if f := runtime.FuncForPC(pc); f == nil {
	} else {
		fn := f.Name()
		o = fn[strings.LastIndex(fn, ".")+1:]
	}

	return LogOri(o)
}
