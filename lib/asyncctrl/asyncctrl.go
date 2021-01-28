package asyncctrl

import (
	"sync"
)

// Mutex allows for thread safety
var Mutex = &sync.Mutex{}
