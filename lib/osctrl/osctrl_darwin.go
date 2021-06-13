// +build darwin

package osctrl

import (
	"syscall"
)

func newProcGroupAttr() *syscall.SysProcAttr {
	newProcGroupAttr := &syscall.SysProcAttr{
		Setpgid: true,
	}

	return newProcGroupAttr
}
