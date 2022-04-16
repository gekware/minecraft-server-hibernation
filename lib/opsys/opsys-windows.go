// +build windows

package opsys

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"msh/lib/errco"
	"msh/lib/utility"
)

// suspend process
// https://github.com/iDigitalFlame/XMT/blob/819fc4e4eeeed6d78b55ea88415f918990666b1b/cmd/cmd_windows.go
// https://github.com/shirou/gopsutil/blob/03f9f5557169e3e2cdefcd31351812e5252fba89/process/process_windows.go
var (
	dllNtdll             = windows.NewLazySystemDLL("ntdll.dll")
	procNtResumeProcess  = dllNtdll.NewProc("NtResumeProcess")
	procNtSuspendProcess = dllNtdll.NewProc("NtSuspendProcess")
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

func procTreeSuspend(ppid uint32) *errco.Error {
	// suspendProc suspends a process by pid
	suspendProc := func(pid uint32) *errco.Error {
		// https://github.com/shirou/gopsutil/blob/03f9f5557169e3e2cdefcd31351812e5252fba89/process/process_windows.go#L759-L773

		h, err := windows.OpenProcess(windows.PROCESS_SUSPEND_RESUME, false, pid)
		if err != nil {
			return errco.NewErr(errco.ERROR_PROCESS_OPEN, errco.LVL_D, "suspendProc", err.Error())
		}
		defer windows.CloseHandle(h)

		r1, _, _ := procNtSuspendProcess.Call(uintptr(h))
		if r1 != 0 {
			// See https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-erref/596a1078-e883-4972-9bbc-49e60bebca55
			return errco.NewErr(errco.ERROR_PROCESS_SUSPEND_CALL, errco.LVL_D, "suspendProc", fmt.Sprintf("NtStatus='0x%.8X'", r1))
		}

		return nil
	}

	// get process tree
	treePid, errMsh := getTreePids(ppid)
	if errMsh != nil {
		return errMsh.AddTrace("procTreeSuspend")
	}

	errco.Logln(errco.LVL_D, "procTreeSuspend: tree pid is %v", treePid)

	// suspend all processes in tree
	for _, pid := range treePid {
		errMsh := suspendProc(pid)
		if errMsh != nil {
			return errMsh.AddTrace("procTreeSuspend")
		}
	}

	return nil
}

func procTreeResume(ppid uint32) *errco.Error {
	// resumeProc resumes a process by pid
	resumeProc := func(pid uint32) *errco.Error {
		// https://github.com/shirou/gopsutil/blob/03f9f5557169e3e2cdefcd31351812e5252fba89/process/process_windows.go#L775-L789

		h, err := windows.OpenProcess(windows.PROCESS_SUSPEND_RESUME, false, pid)
		if err != nil {
			return errco.NewErr(errco.ERROR_PROCESS_OPEN, errco.LVL_D, "resumeProc", err.Error())
		}
		defer windows.CloseHandle(h)

		r1, _, _ := procNtResumeProcess.Call(uintptr(h))
		if r1 != 0 {
			// See https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-erref/596a1078-e883-4972-9bbc-49e60bebca55
			return errco.NewErr(errco.ERROR_PROCESS_SUSPEND_CALL, errco.LVL_D, "resumeProc", fmt.Sprintf("NtStatus='0x%.8X'", r1))
		}

		return nil
	}

	// get process tree
	treePid, errMsh := getTreePids(ppid)
	if errMsh != nil {
		return errMsh.AddTrace("procTreeResume")
	}

	errco.Logln(errco.LVL_D, "procTreeResume: tree pid is %v", treePid)

	// resume all processes in tree
	for _, pid := range treePid {
		errMsh := resumeProc(pid)
		if errMsh != nil {
			return errMsh.AddTrace("procTreeResume")
		}
	}

	return nil
}

// ------------------- utils ------------------- //

// getTreePids will return a list of pids that represent the tree of process pids originating from the specified one.
// (they are ordered: [parent, 1 gen children, 2 gen children, ...])
func getTreePids(rootPid uint32) ([]uint32, *errco.Error) {
	// https://docs.microsoft.com/en-us/windows/win32/api/tlhelp32/ns-tlhelp32-processentry32

	procEntry := syscall.ProcessEntry32{}
	parentLayer := []uint32{rootPid}
	treePids := parentLayer
	foundRootPid := false

	// create snapshot of processes running on system
	snapshot, err := syscall.CreateToolhelp32Snapshot(uint32(syscall.TH32CS_SNAPPROCESS), 0)
	if err != nil {
		return nil, errco.NewErr(errco.ERROR_PROCESS_SYSTEM_SNAPSHOT, errco.LVL_D, "getTreePids", err.Error())
	}
	defer syscall.CloseHandle(snapshot)

	procEntry.Size = uint32(unsafe.Sizeof(procEntry))

	for {
		// set procEntry to the first process in the snapshot
		err = syscall.Process32First(snapshot, &procEntry)
		if err != nil {
			return nil, errco.NewErr(errco.ERROR_PROCESS_ENTRY, errco.LVL_D, "getTreePids", err.Error())
		}

		// loop through the processes in the snapshot, if the parent pid of the analyzed process
		// is in in the parent layer, append the analyzed process pid in the child layer
		var childLayer []uint32
		for {
			if procEntry.ProcessID == rootPid {
				foundRootPid = true
			}

			if utility.SliceContain(procEntry.ParentProcessID, parentLayer) {
				// avoid adding a pid if it's already contained in treePids
				// (pid 0's ppid is 0 and this leads to recursion)
				if !utility.SliceContain(procEntry.ProcessID, treePids) {
					childLayer = append(childLayer, procEntry.ProcessID)
				}
			}

			// advance to next process in snapshot
			err = syscall.Process32Next(snapshot, &procEntry)
			if err != nil {
				// if there aren't anymore processes to be analyzed, break out of the loop
				break
			}
		}

		// if the specified rootPid is not found, return error
		if !foundRootPid {
			return nil, errco.NewErr(errco.ERROR_PROCESS_NOT_FOUND, errco.LVL_D, "getTreePids", "specified rootPid not found")
		}

		// there are no more child processes, return the process tree
		if len(childLayer) == 0 {
			return treePids, nil
		}

		// append the child layer to the tree pids
		treePids = append(treePids, childLayer...)

		// to analyze the next layer, set the child layer to be the new parent layer
		parentLayer = childLayer
	}
}
