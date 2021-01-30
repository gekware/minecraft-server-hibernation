package asyncctrl

import (
	"sync"
)

// Mutex allows for thread safety
var Mutex = &sync.Mutex{}

// WithLock executes a function while Mutex is locked
func WithLock(f func()) {
	Mutex.Lock()
	defer Mutex.Unlock()

	f()
}

// WithLockBool executes a function returning bool while Mutex is locked
func WithLockBool(f func() bool) bool {
	Mutex.Lock()
	defer Mutex.Unlock()

	return f()
}
