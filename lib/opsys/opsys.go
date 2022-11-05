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
		return errco.NewErr(errco.ERROR_OS_NOT_SUPPORTED, errco.LVL_1, "OsSupported", "OS is not supported")
	}

	return nil
}

// NewProcGroupAttr returns a SysProcAttr struct to start a new process group
func NewProcGroupAttr() *syscall.SysProcAttr {
	return newProcGroupAttr()
}

// ProcTreeSuspend suspends a process tree by pid.
// when succeeds returns true
func ProcTreeSuspend(ppid uint32) (bool, *errco.Error) {
	errMsh := procTreeSuspend(ppid)
	if errMsh != nil {
		return false, errMsh.AddTrace("ProcTreeSuspend")
	}

	errco.Logln(errco.LVL_1, "PROCESS TREE SUSPENDED!")

	return true, nil
}

// ProcTreeResume resumes a process tree by pid.
// when succeeds returns false
func ProcTreeResume(ppid uint32) (bool, *errco.Error) {
	errMsh := procTreeResume(ppid)
	if errMsh != nil {
		return true, errMsh.AddTrace("ProcTreeResume")
	}

	errco.Logln(errco.LVL_1, "PROCESS TREE UNSUSPEDED!")

	return false, nil
}

// FileId returns file id
func FileId(filePath string) (uint64, error) {
	return fileId(filePath)
}
