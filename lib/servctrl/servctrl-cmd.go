package servctrl

import (
	"bufio"
	"encoding/json"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/model"
	"msh/lib/opsys"
	"msh/lib/servstats"
	"msh/lib/utility"
)

// ServTerm is the variable that represent the running minecraft server
var ServTerm *servTerminal = &servTerminal{IsActive: false}

// servTerminal is the minecraft server terminal
type servTerminal struct {
	IsActive  bool
	Wg        sync.WaitGroup // used to wait terminal StdoutPipe/StderrPipe
	startTime time.Time      // time at which minecraft server terminal was started
	cmd       *exec.Cmd
	outPipe   io.ReadCloser
	errPipe   io.ReadCloser
	inPipe    io.WriteCloser
}

// lastOut is a channel used to communicate the last line got from the printer function
var lastOut = make(chan string)

// Execute executes a command on ms.
//
// Returns the output lines of ms terminal with a timeout of 200ms since last output line.
//
// (Execute on command with no output doesn't cause hanging)
//
// (Execute on command with multiple lines returns them separated by \n, if print time between them was less than timeout)
//
// [non-blocking]
func Execute(command string) (string, *errco.MshLog) {
	// check if ms is warm and interactable
	logMsh := CheckMSWarm()
	if logMsh != nil {
		return "", logMsh.AddTrace()
	}

	errco.NewLogln(errco.TYPE_INF, errco.LVL_2, errco.ERROR_NIL, "ms command: %s%s%s\t(origin: %s%s%s)", errco.COLOR_CYAN, command, errco.COLOR_RESET, errco.COLOR_YELLOW, errco.Trace(2), errco.COLOR_RESET)

	// write to server terminal (\n indicates the enter key)
	_, err := ServTerm.inPipe.Write([]byte(command + "\n"))
	if err != nil {
		return "", errco.NewLog(errco.TYPE_ERR, errco.LVL_2, errco.ERROR_PIPE_INPUT_WRITE, err.Error())
	}

	// read all lines from lastOut
	// (watchdog used in case there are no more lines to read or output takes too long)
	var out string = ""
a:
	for {
		select {
		case lo := <-lastOut:
			out += lo + "\n"
		case <-time.NewTimer(200 * time.Millisecond).C:
			break a
		}
	}

	// return the (possibly) full terminal output of Execute()
	return out, nil
}

// TellRaw executes a tellraw on ms
// [non-blocking]
func TellRaw(reason, text, origin string) *errco.MshLog {
	// check if ms is warm and interactable
	logMsh := CheckMSWarm()
	if logMsh != nil {
		return logMsh.AddTrace()
	}

	gameMessage, err := json.Marshal(&model.GameRawMessage{Text: "[MSH] " + reason + ": " + text, Color: "aqua", Bold: false})
	if err != nil {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_2, errco.ERROR_JSON_MARSHAL, err.Error())
	}

	gameMessage = append([]byte("tellraw @a "), gameMessage...)
	gameMessage = append(gameMessage, []byte("\n")...)

	errco.NewLogln(errco.TYPE_INF, errco.LVL_2, errco.ERROR_NIL, "ms tellraw: %s%s%s\t(origin: %s)", errco.COLOR_YELLOW, string(gameMessage), errco.COLOR_RESET, origin)

	// write to server terminal (\n indicates the enter key)
	_, err = ServTerm.inPipe.Write(gameMessage)
	if err != nil {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_2, errco.ERROR_PIPE_INPUT_WRITE, err.Error())
	}

	return nil
}

// TermUpTime returns the current minecraft server terminal uptime.
// If ms terminal is not running returns -1.
func TermUpTime() int {
	if !ServTerm.IsActive {
		return -1
	}

	return utility.RoundSec(time.Since(ServTerm.startTime))
}

// WarmUpTime returns the current minecraft server warmed uptime.
// If ms is not warm returns -1.
func WarmUpTime() int {
	if err := CheckMSWarm(); err != nil {
		return -1
	}

	return utility.RoundSec(time.Since(servstats.Stats.WarmUpTime))
}

// CheckMSWarm checks if minecraft server is warm and it's possible to interact with it.
//
// Checks if there is no major error, terminal is active, ms status is online and ms process not suspended.
//
// If ms is warm and interactable, returns nil
func CheckMSWarm() *errco.MshLog {
	switch {
	case servstats.Stats.MajorError != nil:
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_2, errco.ERROR_SERVER_UNRESPONDING, "minecraft server not responding")
	case !ServTerm.IsActive:
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_2, errco.ERROR_TERMINAL_NOT_ACTIVE, "minecraft server terminal not active")
	case servstats.Stats.Status != errco.SERVER_STATUS_ONLINE:
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_2, errco.ERROR_SERVER_NOT_ONLINE, "minecraft server not online")
	case servstats.Stats.Suspended:
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_2, errco.ERROR_SERVER_SUSPENDED, "minecraft server is suspended")
	}

	return nil
}

// termStart starts a new terminal.
// If server terminal is already active it returns without doing anything
// [non-blocking]
func termStart() *errco.MshLog {
	if ServTerm.IsActive {
		errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_SERVER_IS_WARM, "minecraft server terminal already active")
		return nil
	}

	logMsh := termLoad()
	if logMsh != nil {
		return logMsh.AddTrace()
	}

	go printerOutErr()

	err := ServTerm.cmd.Start()
	if err != nil {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_TERMINAL_START, err.Error())
	}

	go waitForExit()

	return nil
}

// termLoad loads cmd/pipes into ServTerm
func termLoad() *errco.MshLog {
	// set terminal cmd
	command, logMsh := config.ConfigRuntime.BuildCommandStartServer()
	if logMsh != nil {
		return logMsh.AddTrace()
	}
	ServTerm.cmd = exec.Command(command[0], command[1:]...)
	ServTerm.cmd.Dir = config.ConfigRuntime.Server.Folder

	// launch as new process group so that signals (ex: SIGINT) are sent to msh
	// (not relayed to the java server child process)
	ServTerm.cmd.SysProcAttr = opsys.NewProcGroupAttr()

	// set terminal pipes
	var err error
	ServTerm.outPipe, err = ServTerm.cmd.StdoutPipe()
	if err != nil {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_PIPE_LOAD, "StdoutPipe load: "+err.Error())
	}
	ServTerm.errPipe, err = ServTerm.cmd.StderrPipe()
	if err != nil {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_PIPE_LOAD, "StderrPipe load: "+err.Error())
	}
	ServTerm.inPipe, err = ServTerm.cmd.StdinPipe()
	if err != nil {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_PIPE_LOAD, "StdinPipe load: "+err.Error())
	}

	return nil
}

// printerOutErr manages the communication from StdoutPipe/StderrPipe.
// Launches 1 goroutine to scan StdoutPipe and 1 goroutine to scan StderrPipe
// (Should be called before cmd.Start())
// [goroutine]
func printerOutErr() {
	// add printer-out + printer-err to waitgroup
	ServTerm.Wg.Add(2)

	// print terminal StdoutPipe
	// [goroutine]
	go func() {
		var line string

		defer ServTerm.Wg.Done()

		scanner := bufio.NewScanner(ServTerm.outPipe)

		for scanner.Scan() {
			line = scanner.Text()

			errco.NewLogln(errco.TYPE_SER, errco.LVL_2, errco.ERROR_NIL, line)

			// communicate to lastOut so that func Execute() can return the output of the command.
			// must be a non-blocking select or it might cause hanging
			select {
			case lastOut <- line:
			default:
			}

			switch servstats.Stats.Status {

			case errco.SERVER_STATUS_STARTING:
				// for modded server terminal compatibility, use separate check for "INFO" and flag-word
				// using only "INFO" and not "[Server thread/INFO]"" because paper minecraft servers don't use "[Server thread/INFO]"

				// "Preparing spawn area: " -> update ServStats.LoadProgress
				if strings.Contains(line, "INFO") && strings.Contains(line, "Preparing spawn area: ") {
					servstats.Stats.LoadProgress = strings.Split(strings.Split(line, "Preparing spawn area: ")[1], "\n")[0]
				}

				// ": Done (" -> set ServStats.Status = ONLINE
				// using ": Done (" instead of "Done" to avoid false positives (issue #112)
				if strings.Contains(line, "INFO") && strings.Contains(line, ": Done (") {
					servstats.Stats.Status = errco.SERVER_STATUS_ONLINE
					errco.NewLogln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "MINECRAFT SERVER IS ONLINE!")

					// schedule soft freeze of ms
					// (if no players connect the server will shutdown)
					FreezeMSSchedule()
				}

			case errco.SERVER_STATUS_ONLINE:
				// It is possible that a player could send a message that contains text similar to server output:
				// 		[14:08:43] [Server thread/INFO]: <player> Stopping
				// 		[14:09:32] [Server thread/INFO]: [player] Stopping
				//
				// These are the correct shutdown logs:
				// 		[14:09:46] [Server thread/INFO]: Stopping the server
				// 		[15Mar2021 14:09:46.581] [Server thread/INFO] [net.minecraft.server.dedicated.DedicatedServer/]: Stopping the server
				//
				// lineSplit is therefore implemented:
				//
				// [14:09:46] [Server thread/INFO]: <player> ciao
				// ^-----------header------------^##^--content--^

				// Continue if line does not contain ": "
				// (it does not adhere to expected log format or it is a multiline java exception)
				if !strings.Contains(line, ": ") {
					continue
				}

				lineSplit := strings.SplitN(line, ": ", 2)
				lineHeader := lineSplit[0]
				lineContent := lineSplit[1]

				if strings.Contains(lineHeader, "INFO") {
					switch {
					// player leaves the server
					case strings.Contains(lineContent, "lost connection:"): // "lost connection" is more general compared to "left the game" (even too much: player might write it in chat -> added ":")
						FreezeMSSchedule()

					// the server is stopping
					case strings.Contains(lineContent, "Stopping") && strings.Contains(lineContent, "server"):
						servstats.Stats.Status = errco.SERVER_STATUS_STOPPING
						errco.NewLogln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "MINECRAFT SERVER IS STOPPING!")
					}
				}

				if strings.Contains(lineHeader, "ERROR") {
					switch {
					case strings.Contains(lineContent, "stopped responding!") || strings.Contains(lineContent, "----------"):
						// example:
						// PROCESS TREE UNSUSPEDED!
						// [18:49:08 WARN]: Can't keep up! Is the server overloaded? Running 121938ms or 2438 ticks behind
						// [18:49:08 ERROR]: ------------------------------
						// [18:49:08 ERROR]: The server has stopped responding! This is (probably) not a Paper bug.
						LogMsh := errco.NewLogln(errco.TYPE_ERR, errco.LVL_1, errco.ERROR_SERVER_UNRESPONDING, "MINECRAFT SERVER IS NOT RESPONDING!")
						servstats.Stats.SetMajorError(LogMsh)
					}
				}
			}
		}
	}()

	// print terminal StderrPipe
	// [goroutine]
	go func() {
		var line string

		defer ServTerm.Wg.Done()

		scanner := bufio.NewScanner(ServTerm.errPipe)

		for scanner.Scan() {
			line = scanner.Text()

			errco.NewLogln(errco.TYPE_SER, errco.LVL_2, errco.ERROR_NIL, line)
		}
	}()
}

// waitForExit waits for server terminal to exit and manages:
//
// - ServTerm.isActive, ServTerm.startTime.
//
// - Stats.Status, Stats.Suspended, Stats.ConnCount, Stats.LoadProgress.
//
// - Suspension refresher.
//
// [goroutine]
func waitForExit() {
	ServTerm.IsActive = true
	ServTerm.startTime = time.Now()
	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "ms terminal started")

	servstats.Stats.Status = errco.SERVER_STATUS_STARTING
	servstats.Stats.Suspended = false
	servstats.Stats.ConnCount = 0
	servstats.Stats.LoadProgress = "0%"
	errco.NewLogln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "MINECRAFT SERVER IS STARTING!")

	// start suspension refresher
	stopSuspendRefresherC := make(chan bool, 1)
	go suspendRefresher(stopSuspendRefresherC)

	// wait for server process to finish
	ServTerm.Wg.Wait()  // wait terminal StdoutPipe/StderrPipe to exit
	ServTerm.cmd.Wait() // wait process (to avoid defunct java server process)

	ServTerm.outPipe.Close()
	ServTerm.errPipe.Close()
	ServTerm.inPipe.Close()

	// stop suspension refresher
	stopSuspendRefresherC <- true

	servstats.Stats.Status = errco.SERVER_STATUS_OFFLINE
	servstats.Stats.Suspended = false
	servstats.Stats.ConnCount = 0
	servstats.Stats.LoadProgress = "0%"
	errco.NewLogln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "MINECRAFT SERVER IS OFFLINE!")

	ServTerm.IsActive = false
	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "ms terminal exited")
}

// suspendRefresher refreshes ms suspension by warming and freezing the server every set amount of time.
//
// If (suspension || suspension refresh) is not allowed this func just returns.
//
// [goroutine stoppable]
func suspendRefresher(stop chan bool) {
	if !config.ConfigRuntime.Msh.SuspendAllow {
		return
	}

	if config.ConfigRuntime.Msh.SuspendRefresh <= 0 {
		return
	}

	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "suspension refresher is starting")

	ticker := time.NewTicker(time.Duration(config.ConfigRuntime.Msh.SuspendRefresh) * time.Second)

	for {
		select {

		case <-stop:
			// received stop signal of suspension refresher
			errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "suspension refresher is stopping")
			return

		case <-ticker.C:
			// check if ms is responding, not offline, suspended
			switch {
			case servstats.Stats.MajorError != nil:
				errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_SERVER_UNRESPONDING, "minecraft server is not responding")
				continue
			case servstats.Stats.Status == errco.SERVER_STATUS_OFFLINE:
				errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_SERVER_OFFLINE, "minecraft server is offline")
				continue
			case !servstats.Stats.Suspended:
				errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_SERVER_NOT_SUSPENDED, "minecraft server terminal is not suspended")
				continue
			}

			// warm ms unsuspending process
			errco.NewLogln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "suspension refresh will warm minecraft server...")
			WarmMS()

			// give time to ms to recover from suspension
			time.Sleep(1 * time.Second)

			// freeze ms suspending process (softly in case a player has joined in the meantime)
			errco.NewLogln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "suspension refresh will freeze minecraft server...")
			FreezeMS(false)
		}
	}
}
