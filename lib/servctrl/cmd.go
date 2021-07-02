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

// ServTerminal is the minecraft server terminal
type ServTerminal struct {
	IsActive bool
	Wg       sync.WaitGroup
	cmd      *exec.Cmd
	outPipe  io.ReadCloser
	errPipe  io.ReadCloser
	inPipe   io.WriteCloser
}

var ServTerm *ServTerminal = &ServTerminal{}

// lastLine is a channel used to communicate the last line got from the printer function
var lastLine = make(chan string)

// constants to print color text on terminal
const colRes string = "\033[0m"
const colCya string = "\033[36m"
const colYel string = "\033[33m"

// CmdStart starts a new terminal (non-blocking) and returns a servTerm object
func CmdStart(dir, command string) error {
	ServTerm.loadCmd(dir, command)

	go ServTerm.startInteraction()

	err := ServTerm.cmd.Start()
	if err != nil {
		return fmt.Errorf("CmdStart: %v", err)
	}

	go ServTerm.waitForExit()

	// initialization
	ServStats.Status = "starting"
	ServStats.LoadProgress = "0%"
	ServStats.PlayerCount = 0
	log.Print("*** MINECRAFT SERVER IS STARTING!")

	return nil
}

// Execute executes a command on the specified term
// [non-blocking]
func (ServTerm *ServTerminal) Execute(command, origin string) (string, error) {
	if !ServTerm.IsActive {
		return "", fmt.Errorf("Execute: terminal not active")
	}

	commands := strings.Split(command, "\n")

	for _, com := range commands {
		if ServStats.Status != "online" {
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

// loadCmd loads cmd into server terminal
func (ServTerm *ServTerminal) loadCmd(dir, command string) {
	cSplit := strings.Split(command, " ")

	ServTerm.cmd = exec.Command(cSplit[0], cSplit[1:]...)
	ServTerm.cmd.Dir = dir

	// launch as new process group so that signals (ex: SIGINT) are sent to msh
	// (not relayed to the java server child process)
	ServTerm.cmd.SysProcAttr = osctrl.NewProcGroupAttr()
}

// startInteraction manages the communication from StdoutPipe/StderrPipe
// Should be called before cmd.Start()
// [goroutine]
func (ServTerm *ServTerminal) startInteraction() {
	// set outPipe and errPipe (also inPipe, since it's needed in Execute)
	ServTerm.outPipe, _ = ServTerm.cmd.StdoutPipe()
	ServTerm.errPipe, _ = ServTerm.cmd.StderrPipe()
	ServTerm.inPipe, _ = ServTerm.cmd.StdinPipe()

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

			if ServStats.Status == "starting" {
				// for modded server terminal compatibility, use separate check for "INFO" and flag-word
				// using only "INFO" and not "[Server thread/INFO]"" because paper minecraft servers don't use "[Server thread/INFO]"

				// "Preparing spawn area: " -> update ServStats.LoadProgress
				if strings.Contains(line, "INFO") && strings.Contains(line, "Preparing spawn area: ") {
					ServStats.LoadProgress = strings.Split(strings.Split(line, "Preparing spawn area: ")[1], "\n")[0]
				}

				// "Done" -> set ServStats.Status = "online"
				if strings.Contains(line, "INFO") && strings.Contains(line, "Done") {
					ServStats.Status = "online"
					log.Print("*** MINECRAFT SERVER IS ONLINE!")

					// launch a stopInstance so that if no players connect the server will shutdown
					RequestStopMinecraftServer()
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

			if ServStats.Status == "online" && len(lineSplit) == 2 {

				if strings.Contains(lineSplit[0], "INFO") {
					switch {
					// player sends a chat message
					case strings.HasPrefix(lineSplit[1], "<") || strings.HasPrefix(lineSplit[1], "["):
						// just log that the line is a chat message
						debugctrl.Logln("a chat message was sent")

					// player joins the server
					// using "UUID of player" since minecraft server v1.12.2 does not use "joined the game"
					case strings.Contains(lineSplit[1], "UUID of player"):
						ServStats.PlayerCount++
						log.Printf("*** A PLAYER JOINED THE SERVER! - %d players online", ServStats.PlayerCount)

					// player leaves the server
					case strings.Contains(lineSplit[1], "left the game"):
						ServStats.PlayerCount--
						log.Printf("*** A PLAYER LEFT THE SERVER! - %d players online", ServStats.PlayerCount)
						RequestStopMinecraftServer()

					// the server is stopping
					case strings.Contains(lineSplit[1], "Stopping"):
						ServStats.Status = "stopping"
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

// waitForExit manages term.isActive parameter and set ServStats.Status = "offline" when it exits.
// [goroutine]
func (ServTerm *ServTerminal) waitForExit() {
	ServTerm.IsActive = true

	// wait for printer out/err to exit
	ServTerm.Wg.Wait()

	ServTerm.outPipe.Close()
	ServTerm.errPipe.Close()
	ServTerm.inPipe.Close()

	ServTerm.IsActive = false
	debugctrl.Logln("waitForExit: terminal exited")

	ServStats.Status = "offline"
	log.Print("*** MINECRAFT SERVER IS OFFLINE!")
}
