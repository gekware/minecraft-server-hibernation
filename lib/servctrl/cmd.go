package servctrl

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"

	"msh/lib/debugctrl"
	"msh/lib/osctrl"
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

// CmdStart starts a new terminal (non-blocking) and returns a servTerm object
func CmdStart(dir, command string) error {
	err := loadTerm(dir, command)
	if err != nil {
		return fmt.Errorf("loadTerm: %v", err)
	}

	go goPrinterOutErr()

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

		debugctrl.Logln("terminal execute:"+colYel, com, colRes, "\t(origin:", origin+")")

		// write to cmd (\n indicates the enter key)
		_, err := ServTerm.inPipe.Write([]byte(com + "\n"))
		if err != nil {
			return "", fmt.Errorf("Execute: %v", err)
		}
	}

	return <-lastLine, nil
}

// loadTerm loads cmd/pipes into ServTerm
func loadTerm(dir, command string) error {
	cSplit := strings.Split(command, " ")

	// set terminal cmd
	ServTerm.cmd = exec.Command(cSplit[0], cSplit[1:]...)
	ServTerm.cmd.Dir = dir

	// launch as new process group so that signals (ex: SIGINT) are sent to msh
	// (not relayed to the java server child process)
	ServTerm.cmd.SysProcAttr = osctrl.NewProcGroupAttr()

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

// goPrinterOutErr manages the communication from StdoutPipe/StderrPipe.
// Launches 1 goroutine to scan StdoutPipe and 1 goroutine to scan StderrPipe
// (Should be called before cmd.Start())
// [goroutine]
func goPrinterOutErr() {
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

			if Stats.Status == "starting" {
				// for modded server terminal compatibility, use separate check for "INFO" and flag-word
				// using only "INFO" and not "[Server thread/INFO]"" because paper minecraft servers don't use "[Server thread/INFO]"

				// "Preparing spawn area: " -> update ServStats.LoadProgress
				if strings.Contains(line, "INFO") && strings.Contains(line, "Preparing spawn area: ") {
					Stats.LoadProgress = strings.Split(strings.Split(line, "Preparing spawn area: ")[1], "\n")[0]
				}

				// "Done" -> set ServStats.Status = "online"
				if strings.Contains(line, "INFO") && strings.Contains(line, "Done") {
					Stats.Status = "online"
					log.Print("*** MINECRAFT SERVER IS ONLINE!")

					// launch a stopInstance so that if no players connect the server will shutdown
					StopMSRequest()
				}
			}

			/*
			 * It is possible that a player could send a message that contains text similar to server output:
			 *		[14:08:43] [Server thread/INFO]: <player> : Stopping
			 * 		[14:09:12] [Server thread/INFO]: <player> ]: Stopping
			 * 		[14:09:32] [Server thread/INFO]: [player] : Stopping
			 * 		[14:09:46] [Server thread/INFO]: [player: Stopping the server]
			 *
			 * When these variations are actually the server logging its shutdown:
			 * 		[14:09:46] [Server thread/INFO]: Stopping the server
			 *		[15Mar2021 14:09:46.581] [Server thread/INFO] [net.minecraft.server.dedicated.DedicatedServer/]: Stopping the server
			 *
			 * One way to handle this is to split the line in two parts:
			 */

			var lineSplit = strings.SplitN(line, ": ", 2)

			/*
			 * lineSplit[0] is the log's "header" (e.g. "[14:09:46] [Server thread/INFO]")
			 * lineSplit[1] is the log's "content" (e.g. "<player> ciao" or "Stopping the server")
			 *
			 * Since lineSplit[1] starts immediately after ": ",
			 * comparison can be done using Strings.HasPrefix (or even direct '==' comparison)
			 *
			 * If line does not contain ": ", there is no reason to check it
			 * (it does not adhere to expected log format or it is a multiline java exception)
			 * This is enforced via checking that len(lineSplit) == 2
			 */

			if Stats.Status == "online" && len(lineSplit) == 2 {

				if strings.Contains(lineSplit[0], "INFO") {
					switch {
					// player sends a chat message
					case strings.HasPrefix(lineSplit[1], "<") || strings.HasPrefix(lineSplit[1], "["):
						// just log that the line is a chat message
						debugctrl.Logln("a chat message was sent")

					// player joins the server
					// using "UUID of player" since minecraft server v1.12.2 does not use "joined the game"
					case strings.Contains(lineSplit[1], "UUID of player"):
						Stats.PlayerCount++
						log.Printf("*** A PLAYER JOINED THE SERVER! - %d players online", Stats.PlayerCount)

					// player leaves the server
					case strings.Contains(lineSplit[1], "left the game"):
						Stats.PlayerCount--
						log.Printf("*** A PLAYER LEFT THE SERVER! - %d players online", Stats.PlayerCount)
						StopMSRequest()

					// the server is stopping
					case strings.Contains(lineSplit[1], "Stopping"):
						Stats.Status = "stopping"
						log.Print("*** MINECRAFT SERVER IS STOPPING!")
					}
				}
			}

			// communicate to lastLine so that func Execute() can return the first line after the command
			select {
			case lastLine <- line:
			default:
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
	debugctrl.Logln("waitForExit: terminal exited")

	Stats.Status = "offline"
	log.Print("*** MINECRAFT SERVER IS OFFLINE!")
}
