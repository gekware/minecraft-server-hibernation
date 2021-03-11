package servctrl

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	"msh/lib/debugctrl"
	"msh/lib/osctrl"
)

// ServTerm is the minecraft server terminal
type ServTerm struct {
	isActive bool
	Wg       sync.WaitGroup
	cmd      *exec.Cmd
	out      io.ReadCloser
	err      io.ReadCloser
	in       io.WriteCloser
}

// lastLine is a channel used to communicate the last line got from the printer function
var lastLine = make(chan string)

var colRes string = "\033[0m"
var colCya string = "\033[36m"

// CmdStart starts a new terminal (non-blocking) and returns a servTerm object
func CmdStart(dir, command string) (*ServTerm, error) {
	term := &ServTerm{}

	term.loadCmd(dir, command)

	err := term.loadStdPipes()
	if err != nil {
		return nil, err
	}

	term.startInteraction()

	err = term.cmd.Start()
	if err != nil {
		return nil, err
	}

	go term.waitForExit()

	// initialization
	ServStats.Status = "starting"
	ServStats.LoadProgress = "0%"
	ServStats.Players = 0
	log.Print("*** MINECRAFT SERVER IS STARTING!")

	return term, nil
}

// Execute executes a command on the specified term
func (term *ServTerm) Execute(command string) (string, error) {
	if !term.isActive {
		return "", fmt.Errorf("servctrl-cmd: Execute: terminal not active")
	}

	commands := strings.Split(command, "\n")

	for _, com := range commands {
		if ServStats.Status == "online" {
			debugctrl.Logger("sending", com, "to terminal")

			// write to cmd (\n indicates the enter key)
			_, err := term.in.Write([]byte(com + "\n"))
			if err != nil {
				return "", err
			}
		} else {
			return "", fmt.Errorf("servctrl-cmd: Execute: server not online")
		}
	}

	return <-lastLine, nil
}

// loadCmd loads cmd into server terminal
func (term *ServTerm) loadCmd(dir, command string) {
	cSplit := strings.Split(command, " ")

	term.cmd = exec.Command(cSplit[0], cSplit[1:]...)
	term.cmd.Dir = dir

	// launch as new process group so that signals (ex: SIGINT) are not sent also the the child process
	term.cmd.SysProcAttr = osctrl.GetSyscallNewProcessGroup()
}

// loadStdPipes loads stdpipes into server terminal
func (term *ServTerm) loadStdPipes() error {
	outPipe, err := term.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	errPipe, err := term.cmd.StderrPipe()
	if err != nil {
		return err
	}
	inPipe, err := term.cmd.StdinPipe()
	if err != nil {
		return err
	}

	term.out = outPipe
	term.err = errPipe
	term.in = inPipe

	return nil
}

// waitForExit manages term.isActive parameter and set ServStats.Status = "offline" when it exits
func (term *ServTerm) waitForExit() {
	term.isActive = true

	// wait for printer out/err to exit
	term.Wg.Wait()

	term.out.Close()
	term.err.Close()
	term.in.Close()

	term.isActive = false
	debugctrl.Logger("cmd: waitForExit: terminal exited")

	ServStats.Status = "offline"
	log.Print("*** MINECRAFT SERVER IS OFFLINE!")
}

// startInteraction manages the communication from term.out/term.err and input to term.in (non-blocking)
func (term *ServTerm) startInteraction() {
	// add printer-out + printer-err to waitgroup
	term.Wg.Add(2)

	// print term.out
	go func() {
		var line string

		defer term.Wg.Done()

		scanner := bufio.NewScanner(term.out)

		for scanner.Scan() {
			line = scanner.Text()

			fmt.Println(colCya + line + colRes)

			// split line into a header (lineSplit[0]) and message contents (lineSplit[1]) for more robust parsing
			var lineSplit = strings.SplitN(line, ": ", 2)

			// case where the server is starting
			if ServStats.Status == "starting" {
				// for modded server terminal compatibility, use separate check for "[Server thread/INFO]" and flag-word

				// if the terminal contains flag-word "Preparing spawn area:", update ServStats.LoadProgress
				if strings.Contains(lineSplit[0], "[Server thread/INFO]") && strings.HasPrefix(lineSplit[1], "Preparing spawn area:") {
					ServStats.LoadProgress = strings.Split(strings.Split(lineSplit[1], "Preparing spawn area: ")[1], "\n")[0]
				}
				// if the terminal contains flag-word "Done", the minecraft server is online
				if strings.Contains(lineSplit[0], "[Server thread/INFO]") && strings.HasPrefix(lineSplit[1], "Done") {
					ServStats.Status = "online"
					log.Print("*** MINECRAFT SERVER IS ONLINE!")

					// launch a stopInstance so that if no players connect the server will shutdown
					RequestStopMinecraftServer()
				}
			}

			// case where the server is online
			if ServStats.Status == "online" {
				// if the terminal contains "Stopping", the minecraft server is stopping
				if strings.Contains(lineSplit[0], "[Server thread/INFO]") && strings.HasPrefix(lineSplit[1], "Stopping") {
					ServStats.Status = "stopping"
					log.Print("*** MINECRAFT SERVER IS STOPPING!")
				}
			}

			// communicate to lastLine so that func Execute() can return the first line after the command
			select {
			case lastLine <- line:
			default:
			}
		}
	}()

	// print term.err
	go func() {
		var line string

		defer term.Wg.Done()

		scanner := bufio.NewScanner(term.err)

		for scanner.Scan() {
			line = scanner.Text()

			fmt.Println(colCya + line + colRes)
		}
	}()

	// input from os.Stdin
	go func() {
		var line string
		var err error

		reader := bufio.NewReader(os.Stdin)

		for {
			line, err = reader.ReadString('\n')
			if err != nil {
				debugctrl.Logger("servTerm scanner:", err.Error())
				continue
			}

			_, err = term.Execute(line)
			if err != nil {
				debugctrl.Logger("servTerm scanner:", err.Error())
				continue
			}
		}
	}()
}
