package errco

type Error struct {
	Cod int    // code of error
	Lvl int    // debug level of error
	Ori string // stack trace origin of error
	Str string // error string
}

// NewErr returns a new msh error object
func NewErr(code, lvl int, ori, str string) *Error {
	return &Error{code, lvl, ori, str}
}

// AddTrace adds the parent function to the error
func (errMsh *Error) AddTrace(pFunc string) *Error {
	errMsh.Ori = pFunc + ": " + errMsh.Ori
	return errMsh
}
