package osctrl

import (
	"syscall"
)

// GetSyscallNewProcessGroup returns a SysProcAttr struct to start a new process group
func GetSyscallNewProcessGroup() *syscall.SysProcAttr {
	syscallNewProcessGroupWin := &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	return syscallNewProcessGroupWin
}
