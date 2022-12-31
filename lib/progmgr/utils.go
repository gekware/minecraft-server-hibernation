package progmgr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/model"
	"msh/lib/servctrl"
	"msh/lib/utility"
)

// buildApi2Req returns Api2Req struct containing data
func buildApi2Req(preTerm bool) *model.Api2Req {
	reqJson := &model.Api2Req{}

	reqJson.ProtV = protv

	reqJson.Msh.V = MshVersion
	reqJson.Msh.ID = config.ConfigRuntime.Msh.ID
	reqJson.Msh.Uptime = utility.RoundSec(time.Since(msh.startTime))
	reqJson.Msh.SuspendAllow = config.ConfigRuntime.Msh.SuspendAllow
	reqJson.Msh.Sgm.Dur = sgm.stats.dur
	reqJson.Msh.Sgm.HibeDur = sgm.stats.hibeDur
	reqJson.Msh.Sgm.UsageCpu = sgm.stats.usageCpu
	reqJson.Msh.Sgm.UsageMem = sgm.stats.usageMem
	reqJson.Msh.Sgm.PlaySec = sgm.stats.playSec
	reqJson.Msh.Sgm.PreTerm = preTerm

	reqJson.Machine.Os = runtime.GOOS
	reqJson.Machine.Arch = runtime.GOARCH
	reqJson.Machine.JavaV = config.JavaV

	// get cpu model and vendor
	if cpuInfo, err := cpu.Info(); err != nil {
		errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_GET_CPU_INFO, err.Error()) // log warning
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
		errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_GET_CORES, err.Error()) // log warning
		reqJson.Machine.CoresSys = -1
	} else {
		reqJson.Machine.CoresSys = cores
	}

	// get memory dedicated to system
	if memInfo, err := mem.VirtualMemory(); err != nil {
		errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_GET_MEMORY, err.Error()) // log warning
		reqJson.Machine.Mem = -1
	} else {
		reqJson.Machine.Mem = int(memInfo.Total)
	}

	reqJson.Server.Uptime = servctrl.TermUpTime()
	reqJson.Server.V = config.ConfigRuntime.Server.Version
	reqJson.Server.Prot = config.ConfigRuntime.Server.Protocol

	return reqJson
}

// sendApi2Req sends api2 request
func sendApi2Req(url string, api2req *model.Api2Req) (*http.Response, *errco.MshLog) {
	// before returning, communicate that request has been sent
	defer func() {
		select {
		case ReqSent <- true:
		default:
		}
	}()

	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "sending api2 request")

	// marshal request struct
	reqByte, err := json.Marshal(api2req)
	if err != nil {
		return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_VERSION, err.Error())
	}

	// create http request
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqByte))
	if err != nil {
		return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_VERSION, err.Error())
	}

	// add header User-Agent, Content-Type
	req.Header.Add("User-Agent", fmt.Sprintf("msh/%s (%s) %s", MshVersion, runtime.GOOS, runtime.GOARCH)) // format: msh/vx.x.x (linux) i386
	req.Header.Set("Content-Type", "application/json")                                                    // necessary for post request

	// execute http request
	errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smsh --> mshc%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, string(reqByte))
	client := &http.Client{Timeout: 4 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_VERSION, err.Error())
	}

	return res, nil
}

// readApi2Res returns response in api2 struct
func readApi2Res(res *http.Response) (*model.Api2Res, *errco.MshLog) {
	defer res.Body.Close()

	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "reading api2 response")

	// read http response
	resByte, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_VERSION, err.Error())
	}
	errco.NewLogln(errco.TYPE_BYT, errco.LVL_4, errco.ERROR_NIL, "%smshc --> msh%s: %v", errco.COLOR_PURPLE, errco.COLOR_RESET, resByte)

	// load res data into resJson
	var resJson *model.Api2Res
	err = json.Unmarshal(resByte, &resJson)
	if err != nil {
		return nil, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_VERSION, err.Error())
	}

	return resJson, nil
}

// getMshTreeStats returns current msh tree cpu/mem usage
func getMshTreeStats() (float64, float64) {
	var mshTreeCpu, mshTreeMem float64 = 0, 0

	if mshProc, err := process.NewProcess(int32(os.Getpid())); err != nil {
		// return current avg usage in case of error
		return sgm.stats.usageCpu, sgm.stats.usageMem
	} else {
		for _, c := range treeProc(mshProc) {
			if pCpu, err := c.CPUPercent(); err != nil {
				// return current avg usage in case of error
				return sgm.stats.usageCpu, sgm.stats.usageMem
			} else if pMem, err := c.MemoryPercent(); err != nil {
				// return current avg usage in case of error
				return sgm.stats.usageCpu, sgm.stats.usageMem
			} else {
				mshTreeCpu += float64(pCpu)
				mshTreeMem += float64(pMem)
			}
		}
	}

	return mshTreeCpu, mshTreeMem
}

// treeProc returns the list of tree pids (with ppid)
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
