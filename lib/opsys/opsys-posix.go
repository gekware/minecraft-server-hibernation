// +build linux darwin

package opsys

import (
	"syscall"

	"msh/lib/errco"
)

func newProcGroupAttr() *syscall.SysProcAttr {
	newProcGroupAttr := &syscall.SysProcAttr{
		Setpgid: true,
	}

	return newProcGroupAttr
}

func procTreeSuspend(ppid uint32) *errco.Error {
	/*
		check also https://github.com/shirou/gopsutil/blob/2f8da0a39487ceddf44cebe53a1b563b0b7173cc/process/process_posix.go#L141-L153
		proc, _ := os.FindProcess(-int(ppid))
		_ = process.Signal(syscall.SIGSTOP)
	*/

	errco.Logln(errco.LVL_D, "procTreeSuspend: suspending %d", ppid)
	err := syscall.Kill(-int(ppid), syscall.SIGSTOP) // negative ppid to suspend whole group
	if err != nil {
		return errco.NewErr(errco.ERROR_PROCESS_SIGNAL, errco.LVL_D, "procTreeSuspend", err.Error())
	}

	return nil
}

func procTreeResume(ppid uint32) *errco.Error {
	/*
		check also https://github.com/shirou/gopsutil/blob/2f8da0a39487ceddf44cebe53a1b563b0b7173cc/process/process_posix.go#L141-L153
		proc, _ := os.FindProcess(-int(ppid))
		_ = process.Signal(syscall.SIGCONT)
	*/

	errco.Logln(errco.LVL_D, "procTreeResume: resuming %d", ppid)
	err := syscall.Kill(-int(ppid), syscall.SIGCONT) // negative ppid to resume whole group
	if err != nil {
		return errco.NewErr(errco.ERROR_PROCESS_SIGNAL, errco.LVL_D, "procTreeResume", err.Error())
	}

	return nil
}
