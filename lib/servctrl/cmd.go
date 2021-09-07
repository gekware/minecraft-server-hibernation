package servctrl

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"

	"msh/lib/logger"
	"msh/lib/opsys"
)

var ServTerm *servTerminal = &servTerminal{}

// servTerminal is the minecraft server terminal
type servTerminal struct {
	IsActive bool
	Wg       sync.WaitGroup
	cmd      *exec.Cmd
	outPipe  io.ReadCloser
	errPipe  io.ReadCloser
	inPipe   io.WriteCloser
}

// lastLine is a channel used to communicate the last line got from the printer function
var lastLine = make(chan string)

// constants to print color text on terminal
const (
	colRes string = "\033[0m"
	colCya string = "\033[36m"
	colYel string = "\033[33m"
)

// Execute executes a command on ServTerm
// [non-blocking]
func Execute(command, origin string) (string, error) {
	if !ServTerm.IsActive {
		return "", fmt.Errorf("Execute: terminal not active")
	}

	commands := strings.Split(command, "\n")

	for _, com := range commands {
		if Stats.Status != "online" {
			return "", fmt.Errorf("Execute: server not online")
		}

		logger.Logln("terminal execute:"+colYel, com, colRes, "\t(origin:", origin+")")

		// write to cmd (\n indicates the enter key)
		_, err := ServTerm.inPipe.Write([]byte(com + "\n"))
		if err != nil {
			return "", fmt.Errorf("Execute: %v", err)
		}
	}

	return <-lastLine, nil
}

// cmdStart starts a new terminal (non-blocking) and returns a servTerm object
func cmdStart(dir, command string) error {
	err := loadTerm(dir, command)
	if err != nil {
		return fmt.Errorf("loadTerm: %v", err)
	}

	go printerOutErr()

	err = ServTerm.cmd.Start()
	if err != nil {
		return fmt.Errorf("CmdStart: %v", err)
	}

	go waitForExit()

	go printDataUsage()

	// initialization
	Stats.Status = "starting"
	Stats.LoadProgress = "0%"
	Stats.PlayerCount = 0
	log.Print("*** MINECRAFT SERVER IS STARTING!")

	return nil
}

// loadTerm loads cmd/pipes into ServTerm
func loadTerm(dir, command string) error {
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
		return fmt.Errorf("StdoutPipe load: %v", err)
	}
	ServTerm.errPipe, err = ServTerm.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("StdoutPipe load: %v", err)
	}
	ServTerm.inPipe, err = ServTerm.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("StdoutPipe load: %v", err)
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

			fmt.Println(colCya + line + colRes)

			// communicate to lastLine so that func Execute() can return the first line after the command
			select {
			case lastLine <- line:
			default:
			}

			switch Stats.Status {

			case "starting":
				// for modded server terminal compatibility, use separate check for "INFO" and flag-word
				// using only "INFO" and not "[Server thread/INFO]"" because paper minecraft servers don't use "[Server thread/INFO]"

				// "Preparing spawn area: "	-> update ServStats.LoadProgress
				if strings.Contains(line, "INFO") && strings.Contains(line, "Preparing spawn area: ") {
					Stats.LoadProgress = strings.Split(strings.Split(line, "Preparing spawn area: ")[1], "\n")[0]
				}

				// ": Done ("				-> set ServStats.Status = "online"
				// using ": Done (" instead of "Done" to avoid false positives (issue #112)
				if strings.Contains(line, "INFO") && strings.Contains(line, ": Done (") {
					Stats.Status = "online"
					log.Print("*** MINECRAFT SERVER IS ONLINE!")

					// launch a StopMSRequests so that if no players connect the server will shutdown
					StopMSRequest()
				}

			case "online":
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
					logger.Logln("printerOutErr: line does not adhere to expected log format")
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
						logger.Logln("a chat message was sent")

					// player joins the server
					// using "UUID of player" since minecraft server v1.12.2 does not use "joined the game"
					case strings.Contains(lineContent, "UUID of player"):
						Stats.PlayerCount++
						log.Printf("*** A PLAYER JOINED THE SERVER! - %d players online", Stats.PlayerCount)

					// player leaves the server
					// using "lost connection" (instead of "left the game") because it's more general (issue #116)
					case strings.Contains(lineContent, "lost connection"):
						Stats.PlayerCount--
						log.Printf("*** A PLAYER LEFT THE SERVER! - %d players online", Stats.PlayerCount)
						StopMSRequest()

					// the server is stopping
					case strings.Contains(lineContent, "Stopping"):
						Stats.Status = "stopping"
						log.Print("*** MINECRAFT SERVER IS STOPPING!")
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

			fmt.Println(colCya + line + colRes)
		}
	}()
}

// waitForExit manages ServTerm.isActive parameter and set ServStats.Status = "offline" when minecraft server process exits.
// [goroutine]
func waitForExit() {
	ServTerm.IsActive = true

	// wait for printer out/err to exit
	ServTerm.Wg.Wait()

	ServTerm.outPipe.Close()
	ServTerm.errPipe.Close()
	ServTerm.inPipe.Close()

	ServTerm.IsActive = false
	logger.Logln("waitForExit: terminal exited")

	Stats.Status = "offline"
	log.Print("*** MINECRAFT SERVER IS OFFLINE!")
}
