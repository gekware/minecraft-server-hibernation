package minequery

import (
	"errors"
)

// ErrInvalidStatus wraps errors occurred during ping status deserialization.
// Some errors may be ignored if UseStrict is not set to true.
var ErrInvalidStatus = errors.New("invalid status")
