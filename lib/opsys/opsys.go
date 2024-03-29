package opsys

import (
	"runtime"
	"syscall"

	"msh/lib/errco"
)

// OsSupported returns nil if the OS is supported
func OsSupported() *errco.MshLog {
	// check if OS is windows/linux/macos
	ros := runtime.GOOS

	if ros != "linux" && ros != "windows" && ros != "darwin" {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_1, errco.ERROR_OS_NOT_SUPPORTED, "OS is not supported")
	}

	return nil
}

// NewProcGroupAttr returns a SysProcAttr struct to start a new process group
func NewProcGroupAttr() *syscall.SysProcAttr {
	return newProcGroupAttr()
}

// ProcTreeSuspend suspends a process tree by pid.
// when succeeds returns: true, nil
func ProcTreeSuspend(ppid uint32) (bool, *errco.MshLog) {
	logMsh := procTreeSuspend(ppid)
	if logMsh != nil {
		return false, logMsh.AddTrace()
	}

	errco.NewLogln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "EXECUTED PROCESS TREE SUSPEND!")

	return true, nil
}

// ProcTreeResume resumes a process tree by pid.
// when succeeds returns: false, nil
func ProcTreeResume(ppid uint32) (bool, *errco.MshLog) {
	logMsh := procTreeResume(ppid)
	if logMsh != nil {
		return true, logMsh.AddTrace()
	}

	errco.NewLogln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "EXECUTED PROCESS TREE RESUME!")

	return false, nil
}

// ProcTreeKill kills a process tree by pid.
// when succeeds returns nil
func ProcTreeKill(ppid uint32) *errco.MshLog {
	return procTreeKill(ppid)
}

// FileId returns file id
func FileId(filePath string) (uint64, error) {
	return fileId(filePath)
}
