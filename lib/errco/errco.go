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
func NewLog(o LogOri, t LogTyp, l LogLvl, c LogCod, m string, a ...interface{}) *MshLog {
	return &MshLog{o, t, l, c, m, a}
}

// AddTrace adds the parent function to the error
func (log *MshLog) AddTrace(o LogOri) *MshLog {
	log.Ori = o + LogOri(": ") + log.Ori
	return log
}

// Orig returns the function name it is called from
func Orig() LogOri {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return "?"
	}

	f := runtime.FuncForPC(pc)
	if f == nil {
		return "?"
	}

	fn := f.Name()

	return LogOri(fn[strings.LastIndex(fn, ".")+1:])
}
