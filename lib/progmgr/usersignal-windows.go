//go:build windows
// +build windows

package progmgr

// On Windows, we simply do nothing and never handle user-defined signals, since
// SIGUSR does not exists.
func SetupUserSignalHandling(program *program) {
	return
}
