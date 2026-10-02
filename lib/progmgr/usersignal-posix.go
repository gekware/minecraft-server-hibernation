//go:build linux || darwin
// +build linux darwin

package progmgr

import (
	"msh/lib/errco"
	"msh/lib/servctrl"
	"os/signal"
	"syscall"
)

// On POSIX, this function setups a goroutine that handles incoming SIGUSR1 and
// SIGUSR2 signals and freeze or warms the server respectively
func SetupUserSignalHandling(program *program) {
	// set program.sigUser to relay user-defined signals
	signal.Notify(program.sigUser, syscall.SIGUSR1, syscall.SIGUSR2)
	go handleUserSignal()
}

// handleUserSignal is responsable for handling SIGUSR1 and SIGUSR2, to both freeze
// and warm the minecraft server respectively
// [goroutine]
func handleUserSignal() {
	for {
		sig := <-msh.sigUser
		errco.NewLogln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "received signal: %s", sig.String())

		const (
			SIGNAL_FREEZE = syscall.SIGUSR1
			SIGNAL_WARM   = syscall.SIGUSR2
		)

		if sig == SIGNAL_WARM {
			servctrl.WarmMS()
		} else if sig == SIGNAL_FREEZE {
			servctrl.FreezeMS(false)
		}
	}
}
