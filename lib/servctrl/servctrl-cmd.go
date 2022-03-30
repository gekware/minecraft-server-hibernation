package servctrl

import (
	"bufio"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"msh/lib/errco"
	"msh/lib/opsys"
	"msh/lib/servstats"
)

// ServTerm is the variable that represent the running minecraft server
var ServTerm *servTerminal = &servTerminal{IsActive: false}

// servTerminal is the minecraft server terminal
type servTerminal struct {
	IsActive  bool
	Wg        sync.WaitGroup
	startTime time.Time // uptime of minecraft server
	cmd       *exec.Cmd
	outPipe   io.ReadCloser
	errPipe   io.ReadCloser
	inPipe    io.WriteCloser
}

// lastLine is a channel used to communicate the last line got from the printer function
var lastLine = make(chan string)

// Execute executes a command on ServTerm
// [non-blocking]
func Execute(command, origin string) (string, *errco.Error) {
	if !ServTerm.IsActive {
		return "", errco.NewErr(errco.ERROR_TERMINAL_NOT_ACTIVE, errco.LVL_C, "Execute", "terminal not active")
	}

	commands := strings.Split(command, "\n")

	for _, com := range commands {
		if servstats.Stats.Status != errco.SERVER_STATUS_ONLINE {
			return "", errco.NewErr(errco.ERROR_SERVER_NOT_ONLINE, errco.LVL_C, "Execute", "server not online")
		}

		errco.Logln(errco.LVL_C, "terminal execute: %s%s%s\t(origin: %s)", errco.COLOR_YELLOW, com, errco.COLOR_RESET, origin)

		// write to cmd (\n indicates the enter key)
		_, err := ServTerm.inPipe.Write([]byte(com + "\n"))
		if err != nil {
			return "", errco.NewErr(errco.ERROR_PIPE_INPUT_WRITE, errco.LVL_C, "Execute", err.Error())
		}
	}

	return <-lastLine, nil
}

// TermUpTime returns the current minecraft server uptime.
// in case of error -1 is returned.
func TermUpTime() int {
	if !ServTerm.IsActive {
		return 0
	}

	return int(time.Since(ServTerm.startTime).Seconds())
}

// termStart starts a new terminal (non-blocking) and returns a servTerm object
func termStart(dir, command string) *errco.Error {
	errMsh := termLoad(dir, command)
	if errMsh != nil {
		return errMsh.AddTrace("cmdStart")
	}

	go printerOutErr()

	err := ServTerm.cmd.Start()
	if err != nil {
		return errco.NewErr(errco.ERROR_TERMINAL_START, errco.LVL_D, "cmdStart", err.Error())
	}

	go waitForExit()

	// initialization
	servstats.Stats.LoadProgress = "0%"
	servstats.Stats.PlayerCount = 0

	return nil
}

// termLoad loads cmd/pipes into ServTerm
func termLoad(dir, command string) *errco.Error {
	cSplit := strings.Split(command, " ")

	// set terminal cmd
	ServTerm.cmd = exec.Command(cSplit[0], cSplit[1:]...)
	ServTerm.cmd.Dir = dir

	// launch as new process group so that signals (ex: SIGINT) are sent to msh
	// (not relayed to the java server child process)
	ServTerm.cmd.SysProcAttr = opsys.NewProcGroupAttr()

	// set terminal pipes
	var err error
	ServTerm.outPipe, err = ServTerm.cmd.StdoutPipe()
	if err != nil {
		return errco.NewErr(errco.ERROR_PIPE_LOAD, errco.LVL_D, "loadTerm", "StdoutPipe load: "+err.Error())
	}
	ServTerm.errPipe, err = ServTerm.cmd.StderrPipe()
	if err != nil {
		return errco.NewErr(errco.ERROR_PIPE_LOAD, errco.LVL_D, "loadTerm", "StderrPipe load: "+err.Error())
	}
	ServTerm.inPipe, err = ServTerm.cmd.StdinPipe()
	if err != nil {
		return errco.NewErr(errco.ERROR_PIPE_LOAD, errco.LVL_D, "loadTerm", "StdinPipe load: "+err.Error())
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

			errco.Logln(errco.LVL_C, "%s%s%s", errco.COLOR_GRAY, line, errco.COLOR_RESET)

			// communicate to lastLine so that func Execute() can return the first line after the command
			select {
			case lastLine <- line:
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
					errco.Logln(errco.LVL_B, "MINECRAFT SERVER IS ONLINE!")

					// launch a StopMSRequests so that if no players connect the server will shutdown
					StopMSRequest()
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
					errco.LogMshErr(errco.NewErr(errco.ERROR_SERVER_UNEXP_OUTPUT, errco.LVL_C, "printerOutErr", "line does not adhere to expected log format"))
					continue
				}

				lineSplit := strings.SplitN(line, ": ", 2)
				lineHeader := lineSplit[0]
				lineContent := lineSplit[1]

				if strings.Contains(lineHeader, "INFO") {
					switch {
					// player sends a chat message
					case strings.HasPrefix(lineContent, "<") || strings.HasPrefix(lineContent, "["):
						// just log that the line is a chat message
						errco.Logln(errco.LVL_C, "a chat message was sent")

					// player joins the server
					// using "UUID of player" since minecraft server v1.12.2 does not use "joined the game"
					case strings.Contains(lineContent, "UUID of player"):
						servstats.Stats.PlayerCount++
						errco.Logln(errco.LVL_C, "A PLAYER JOINED THE SERVER! - %d players online", servstats.Stats.PlayerCount)

					// player leaves the server
					// using "lost connection" (instead of "left the game") because it's more general (issue #116)
					case strings.Contains(lineContent, "lost connection"):
						servstats.Stats.PlayerCount--
						errco.Logln(errco.LVL_C, "A PLAYER LEFT THE SERVER! - %d players online", servstats.Stats.PlayerCount)
						StopMSRequest()

					// the server is stopping
					case strings.Contains(lineContent, "Stopping"):
						servstats.Stats.Status = errco.SERVER_STATUS_STOPPING
						errco.Logln(errco.LVL_B, "MINECRAFT SERVER IS STOPPING!")
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

			errco.Logln(errco.LVL_C, "%s%s%s", errco.COLOR_GRAY, line, errco.COLOR_RESET)
		}
	}()
}

// waitForExit manages ServTerm.isActive parameter and set ServStats.Status = OFFLINE when minecraft server process exits.
// [goroutine]
func waitForExit() {
	servstats.Stats.Status = errco.SERVER_STATUS_STARTING
	errco.Logln(errco.LVL_B, "MINECRAFT SERVER IS STARTING!")

	ServTerm.IsActive = true
	errco.Logln(errco.LVL_D, "waitForExit: terminal started")

	// set terminal start time
	ServTerm.startTime = time.Now()

	// wait for printer out/err to exit
	ServTerm.Wg.Wait()

	ServTerm.outPipe.Close()
	ServTerm.errPipe.Close()
	ServTerm.inPipe.Close()

	ServTerm.IsActive = false
	errco.Logln(errco.LVL_D, "waitForExit: terminal exited")

	servstats.Stats.Status = errco.SERVER_STATUS_OFFLINE
	errco.Logln(errco.LVL_B, "MINECRAFT SERVER IS OFFLINE!")
}
