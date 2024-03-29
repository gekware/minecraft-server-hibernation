//go:build linux || darwin

package opsys

import (
	"fmt"
	"os"
	"syscall"

	"msh/lib/errco"
)

func newProcGroupAttr() *syscall.SysProcAttr {
	newProcGroupAttr := &syscall.SysProcAttr{
		Setpgid: true, // start process in a new group
		// Pdeathsig: syscall.SIGKILL, // terminate process when its parent dies (linux only)
	}

	return newProcGroupAttr
}

func procTreeSuspend(ppid uint32) *errco.MshLog {
	/*
		check also https://github.com/shirou/gopsutil/blob/2f8da0a39487ceddf44cebe53a1b563b0b7173cc/process/process_posix.go#L141-L153
		proc, _ := os.FindProcess(-int(ppid))
		_ = process.Signal(syscall.SIGSTOP)
	*/

	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "suspending proc tree (pid: %d)", ppid)

	err := syscall.Kill(-int(ppid), syscall.SIGSTOP) // negative ppid to suspend whole group
	if err != nil {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_PROCESS_SIGNAL, err.Error())
	}

	return nil
}

func procTreeResume(ppid uint32) *errco.MshLog {
	/*
		check also https://github.com/shirou/gopsutil/blob/2f8da0a39487ceddf44cebe53a1b563b0b7173cc/process/process_posix.go#L141-L153
		proc, _ := os.FindProcess(-int(ppid))
		_ = process.Signal(syscall.SIGCONT)
	*/

	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "resuming proc tree (pid: %d)", ppid)

	err := syscall.Kill(-int(ppid), syscall.SIGCONT) // negative ppid to resume whole group
	if err != nil {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_PROCESS_SIGNAL, err.Error())
	}

	return nil
}

func procTreeKill(ppid uint32) *errco.MshLog {
	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "killing proc tree (pid: %d)", ppid)

	err := syscall.Kill(-int(ppid), syscall.SIGKILL) // negative ppid to kill whole group
	if err != nil {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_PROCESS_SIGNAL, err.Error())
	}

	return nil
}

func fileId(filePath string) (uint64, error) {
	// https://github.com/hymkor/go-windows-fileid/blob/master/main_unix.go
	fileInf, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	stat, ok := fileInf.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("os.Fileinfo.Sys() is not syscall.Stat_t")
	}
	return stat.Ino, nil
}
