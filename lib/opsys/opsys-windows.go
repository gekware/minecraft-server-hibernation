// +build windows

package opsys

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"msh/lib/errco"
	"msh/lib/utility"
)

// suspend process
// https://github.com/iDigitalFlame/XMT/blob/819fc4e4eeeed6d78b55ea88415f918990666b1b/cmd/cmd_windows.go
var (
	dllNtdll             = windows.NewLazySystemDLL("ntdll.dll")
	funcNtResumeProcess  = dllNtdll.NewProc("NtResumeProcess")
	funcNtSuspendProcess = dllNtdll.NewProc("NtSuspendProcess")
)

func newProcGroupAttr() *syscall.SysProcAttr {
	newProcGroupAttr := &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	return newProcGroupAttr
}

func suspendProcTree(ppid uint32) *errco.Error {
	// get process tree
	treePid, errMsh := getTreePids(uint32(ppid))
	if errMsh != nil {
		return errMsh.AddTrace("suspendProcTree")
	}

	errco.Logln(errco.LVL_D, "suspendProcTree: tree pid is %v", treePid)

	// suspend all processes in tree
	for _, pid := range treePid {
		// https://github.com/iDigitalFlame/XMT/blob/819fc4e4eeeed6d78b55ea88415f918990666b1b/cmd/cmd_windows.go#L122

		h, err := windows.OpenProcess(windows.PROCESS_SUSPEND_RESUME, false, pid)
		if err != nil {
			return errco.NewErr(errco.ERROR_PROCESS_OPEN, errco.LVL_D, "suspendProcTree", err.Error())
		}
		r, _, err := funcNtSuspendProcess.Call(uintptr(h))
		if windows.CloseHandle(h); r != 0 {
			return errco.NewErr(errco.ERROR_PROCESS_SUSPEND_CALL, errco.LVL_D, "suspendProcTree", err.Error())
		}
	}

	return nil
}

func resumeProcTree(ppid uint32) *errco.Error {
	// get process tree
	treePid, errMsh := getTreePids(uint32(ppid))
	if errMsh != nil {
		return errMsh.AddTrace("resumeProcTree")
	}

	errco.Logln(errco.LVL_D, "resumeProcTree: tree pid is %v", treePid)

	// suspend all processes in tree
	for _, pid := range treePid {
		// https://github.com/iDigitalFlame/XMT/blob/819fc4e4eeeed6d78b55ea88415f918990666b1b/cmd/cmd_windows.go#L106

		h, err := windows.OpenProcess(windows.PROCESS_SUSPEND_RESUME, false, pid)
		if err != nil {
			return errco.NewErr(errco.ERROR_PROCESS_OPEN, errco.LVL_D, "resumeProcTree", err.Error())
		}
		r, _, err := funcNtResumeProcess.Call(uintptr(h))
		if windows.CloseHandle(h); r != 0 {
			return errco.NewErr(errco.ERROR_PROCESS_RESUME_CALL, errco.LVL_D, "resumeProcTree", err.Error())
		}
	}

	return nil
}

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

			if utility.SliceContain(parentLayer, procEntry.ParentProcessID) {
				// avoid adding a pid if it's already contained in treePids
				// (pid 0's ppid is 0 and this leads to recursion)
				if !utility.SliceContain(treePids, procEntry.ProcessID) {
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
