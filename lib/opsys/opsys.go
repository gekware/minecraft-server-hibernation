package opsys

import (
	"fmt"
	"runtime"
	"syscall"
)

// OsSupported returns nil if the OS is supported
func OsSupported() error {
	// check if OS is windows/linux/macos
	ros := runtime.GOOS

	if ros != "linux" && ros != "windows" && ros != "darwin" {
		return fmt.Errorf("OsSupported: OS is not supported")
	}

	return nil
}

// NewProcGroupAttr returns a SysProcAttr struct to start a new process group
func NewProcGroupAttr() *syscall.SysProcAttr {
	return newProcGroupAttr()
}
