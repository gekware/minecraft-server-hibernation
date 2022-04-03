// +build windows

package opsys

import (
	"msh/lib/errco"
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

func init() {
	// enable virtual terminal processing to enable colors on windows terminal
	stdout := windows.Handle(os.Stdout.Fd())
	var originalMode uint32
	if err := windows.GetConsoleMode(stdout, &originalMode); err != nil {
		errco.LogMshErr(errco.NewErr(errco.ERROR_COLOR_ENABLE, errco.LVL_D, "errco init", "error while enabling colors on terminal"))
	} else if windows.SetConsoleMode(stdout, originalMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
		errco.LogMshErr(errco.NewErr(errco.ERROR_COLOR_ENABLE, errco.LVL_D, "errco init", "error while enabling colors on terminal"))
	}
}

func newProcGroupAttr() *syscall.SysProcAttr {
	newProcGroupAttr := &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	return newProcGroupAttr
}
