package progmgr

import (
	"runtime"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/model"
	"msh/lib/servctrl"
)

// buildApi2Req returns Api2Req struct containing data
func buildApi2Req(preTerm bool) *model.Api2Req {
	reqJson := &model.Api2Req{}

	reqJson.Protv = protv

	reqJson.Msh.ID = config.ConfigRuntime.Msh.ID
	reqJson.Msh.Mshv = MshVersion
	reqJson.Msh.Uptime = int(time.Since(msh.startTime).Seconds())
	reqJson.Msh.AllowSuspend = config.ConfigRuntime.Msh.AllowSuspend
	reqJson.Msh.Sgm.Seconds = sgm.stats.seconds
	reqJson.Msh.Sgm.SecondsHibe = sgm.stats.secondsHibe
	reqJson.Msh.Sgm.CpuUsage = sgm.stats.cpuUsage
	reqJson.Msh.Sgm.MemUsage = sgm.stats.memUsage
	reqJson.Msh.Sgm.PlayerSec = sgm.stats.playerSec
	reqJson.Msh.Sgm.PreTerm = preTerm

	reqJson.Machine.Os = runtime.GOOS
	reqJson.Machine.Arch = runtime.GOARCH
	reqJson.Machine.Javav = config.Javav

	// get cpu model and vendor
	if cpuInfo, err := cpu.Info(); err != nil {
		errco.LogMshErr(errco.NewErr(errco.ERROR_GET_CPU_INFO, errco.LVL_D, "buildReq", err.Error())) // non blocking error
		reqJson.Machine.CpuModel = ""
		reqJson.Machine.CpuVendor = ""
	} else {
		if reqJson.Machine.CpuModel = cpuInfo[0].ModelName; reqJson.Machine.CpuModel == "" {
			reqJson.Machine.CpuModel = cpuInfo[0].Model
		}
		reqJson.Machine.CpuVendor = cpuInfo[0].VendorID
	}

	// get cores dedicated to msh
	reqJson.Machine.CoresMsh = runtime.NumCPU()

	// get cores dedicated to system
	if cores, err := cpu.Counts(true); err != nil {
		errco.LogMshErr(errco.NewErr(errco.ERROR_GET_CORES, errco.LVL_D, "buildReq", err.Error())) // non blocking error
		reqJson.Machine.CoresSys = -1
	} else {
		reqJson.Machine.CoresSys = cores
	}

	// get memory dedicated to system
	if memInfo, err := mem.VirtualMemory(); err != nil {
		errco.LogMshErr(errco.NewErr(errco.ERROR_GET_MEMORY, errco.LVL_D, "buildReq", err.Error())) // non blocking error
		reqJson.Machine.Mem = -1
	} else {
		reqJson.Machine.Mem = int(memInfo.Total)
	}

	reqJson.Server.Uptime = servctrl.TermUpTime()
	reqJson.Server.Msv = config.ConfigRuntime.Server.Version
	reqJson.Server.MsProt = config.ConfigRuntime.Server.Protocol

	return reqJson
}

// treeProc returns the list of tree pids (also original ppid)
func treeProc(proc *process.Process) []*process.Process {
	children, err := proc.Children()
	if err != nil {
		// on linux, if a process does not have children an error is returned
		if err.Error() != "process does not have children" {
			return []*process.Process{proc}
		}
		// return pid -1 to indicate that an error occurred
		return []*process.Process{{Pid: -1}}
	}

	tree := []*process.Process{proc}
	for _, child := range children {
		tree = append(tree, treeProc(child)...)
	}
	return tree
}
