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

// SuspendProcTree suspends a process tree by pid
func SuspendProcTree(pid uint32) *errco.Error {
	errMsh := suspendProcTree(pid)
	if errMsh != nil {
		return errMsh.AddTrace("SuspendProcTree")
	}
	return nil
}

// ResumeProcTree resumes a process tree by pid
func ResumeProcTree(pid uint32) *errco.Error {
	errMsh := resumeProcTree(pid)
	if errMsh != nil {
		return errMsh.AddTrace("ResumeProcTree")
	}
	return nil
}
