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
	"syscall"

	"msh/lib/debugctrl"
)

// ServTerm is the minecraft server terminal
type ServTerm struct {
	isActive bool
	Wg       sync.WaitGroup
	cmd      *exec.Cmd
	out      readcl
	err      readcl
	in       writecl
}

// readcl inherits io.ReadCloser and a string is used to indentify it as "out" or "err"
type readcl struct {
	io.ReadCloser
	typ string
}

// writecl inherits io.WriteCloser
type writecl struct {
	io.WriteCloser
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

	term.Wg.Add(2)
	go term.out.printer(term)
	go term.err.printer(term)
	go term.scanner()

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

func (term *ServTerm) loadCmd(dir, command string) {
	cSplit := strings.Split(command, " ")

	term.cmd = exec.Command(cSplit[0], cSplit[1:]...)
	term.cmd.Dir = dir

	// launch as new process group so that signals (ex: SIGINT) are not sent also the the child process
	term.cmd.SysProcAttr = &syscall.SysProcAttr{
		// windows	//
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		// linux	//
		// Setpgid: true,
	}
}

func (term *ServTerm) loadStdPipes() error {
	stdOut, err := term.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stdErr, err := term.cmd.StderrPipe()
	if err != nil {
		return err
	}
	stdIn, err := term.cmd.StdinPipe()
	if err != nil {
		return err
	}

	term.out = readcl{stdOut, "out"}
	term.err = readcl{stdErr, "err"}
	term.in = writecl{stdIn}

	return nil
}

// waitForExit manages term.isActive parameter and set ServStats.Status = "offline" when it exits
func (term *ServTerm) waitForExit() {
	term.isActive = true

	// wait for printer (out-err) to exit
	term.Wg.Wait()

	term.out.Close()
	term.err.Close()
	term.in.Close()

	term.isActive = false
	debugctrl.Logger("cmd: waitForExit: terminal exited")

	ServStats.Status = "offline"
	log.Print("*** MINECRAFT SERVER IS OFFLINE!")
}

func (cmdOutErrReader *readcl) printer(term *ServTerm) {
	var line string

	defer term.Wg.Done()

	scanner := bufio.NewScanner(cmdOutErrReader)

	for scanner.Scan() {
		line = scanner.Text()

		fmt.Println(colCya + line + colRes)

		if cmdOutErrReader.typ == "out" {

			// case where the server is starting
			if ServStats.Status == "starting" {
				if strings.Contains(line, "Preparing spawn area: ") {
					ServStats.LoadProgress = strings.Split(strings.Split(line, "Preparing spawn area: ")[1], "\n")[0]
				}
				if strings.Contains(line, "[Server thread/INFO]: Done") {
					ServStats.Status = "online"
					log.Print("*** MINECRAFT SERVER IS ONLINE!")

					// launch a stopInstance so that if no players connect the server will shutdown
					RequestStopMinecraftServer()
				}
			}

			// case where the server is online
			if ServStats.Status == "online" {
				// if the terminal contains "Stopping" this means that the minecraft server is stopping
				if strings.Contains(line, "[Server thread/INFO]: Stopping") {
					ServStats.Status = "stopping"
					log.Print("*** MINECRAFT SERVER IS STOPPING!")
				}
			}
		}

		// communicate to lastLine so that Execute function can return the first line after the command
		select {
		case lastLine <- line:
		default:
		}
	}
}

func (term *ServTerm) scanner() {
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
}
