package asyncctrl

import (
	"sync"
)

// Mutex allows for thread safety
var Mutex = &sync.Mutex{}

// WithLock executes a function while Mutex is locked
func WithLock(f func()) {
	Mutex.Lock()
	f()
	Mutex.Unlock()
}
