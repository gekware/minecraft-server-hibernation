package errco

type Error struct {
	Cod      int    // code of error
	Lvl      int    // debug level of error
	Ori      string // stack trace origin of error
	Str      string // error string
	Blocking bool   // if the error blocks the go-routine
}

// NewErr returns a new msh error object
func NewErr(code, lvl int, ori, str string, blocking bool) *Error {
	return &Error{code, lvl, ori, str, blocking}
}

// AddTrace adds the parent function to the error
func (errMsh *Error) AddTrace(pFunc string) *Error {
	errMsh.Ori = pFunc + ": " + errMsh.Ori
	return errMsh
}

// MustReturn indicates if the error should block execution or not.
// In case the execution is not blocked, it will log the error within
// the function itself
func (errMsh *Error) MustReturn() bool {
	mustReturn := errMsh != nil && errMsh.Blocking

	// if the error does not cause the function to return, log the error
	if !mustReturn {
		LogMshErr(errMsh)
	}

	return mustReturn
}
