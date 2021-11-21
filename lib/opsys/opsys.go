package opsys

import (
	"runtime"
	"syscall"

	"msh/lib/errco"
)

// OsSupported returns nil if the OS is supported
func OsSupported() *errco.Error {
	// check if OS is windows/linux/macos
	ros := runtime.GOOS

	if ros != "linux" && ros != "windows" && ros != "darwin" {
		return errco.NewErr(errco.ERROR_OS_NOT_SUPPORTED, errco.LVL_B, "OsSupported", "OS is not supported")
	}

	return nil
}

// NewProcGroupAttr returns a SysProcAttr struct to start a new process group
func NewProcGroupAttr() *syscall.SysProcAttr {
	return newProcGroupAttr()
}
