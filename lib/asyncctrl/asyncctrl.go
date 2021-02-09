package asyncctrl

import (
	"sync"
)

var m = &sync.Mutex{}

// WithLock executes a function while Mutex is locked.
// The function can either return nothing or return a value.
// (If the function returns nothing, WithLock will return nil)
func WithLock(i interface{}) interface{} {
	m.Lock()
	defer m.Unlock()

	switch i.(type) {
	case func():
		i.(func())()
	case func() interface{}:
		return i.(func() interface{})()
	}

	return nil
}
