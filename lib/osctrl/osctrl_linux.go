// +build linux

package osctrl

import (
	"syscall"
)

// GetSyscallNewProcessGroup returns a SysProcAttr struct to start a new process group
func GetSyscallNewProcessGroup() *syscall.SysProcAttr {
	syscallNewProcessGroup := &syscall.SysProcAttr{
		Setpgid: true,
	}

	return syscallNewProcessGroup
}
