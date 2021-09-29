package errco

type error struct {
	Cod      int    // code of error
	Lvl      int    // debug level of error
	Ori      string // stack trace origin of error
	Str      string // error string
	Blocking bool   // if the error blocks the go-routine
}

func NewErr(code, lvl int, ori, str string, blocking bool) error {
	return error{code, lvl, ori, str, blocking}
}

func (e error) AddTrace(motherFunc string) error {
	e.Ori = motherFunc + ": " + e.Ori
	return e
}
