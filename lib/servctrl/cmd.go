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
)

// ServTerm is the minecraft server terminal
type ServTerm struct {
	IsActive bool
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

	go term.out.printer()
	go term.err.printer()
	go term.in.scanner()

	err = term.cmd.Start()
	if err != nil {
		return nil, err
	}

	go term.waitForExit()

	return term, nil
}

// Execute executes a command on the specified term
func (term *ServTerm) Execute(command string) error {
	if !term.IsActive {
		return fmt.Errorf("terminal is not active")
	}

	commands := strings.Split(command, "\n")

	for _, com := range commands {
		// needs to be added otherwise the virtual "enter" button is not pressed
		com += "\n"

		log.Print("terminal execute: ", com)

		// write to cmd
		_, err := term.in.Write([]byte(com))
		if err != nil {
			return err
		}
	}

	return nil
}

func (term *ServTerm) loadCmd(dir, command string) {
	cSplit := strings.Split(command, " ")

	term.cmd = exec.Command(cSplit[0], cSplit[1:]...)
	term.cmd.Dir = dir
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

func (term *ServTerm) waitForExit() {
	term.IsActive = true

	term.Wg.Add(1)
	err := term.cmd.Wait()
	if err != nil {
		debugctrl.Logger("waitForExit: error while waiting for cmd exit")
	}
	term.Wg.Done()

	term.IsActive = false

	term.out.Close()
	term.err.Close()
	term.in.Close()

	fmt.Println("terminal exited correctly")
}

func (cmdOutErrReader *readcl) printer() {
	var line string

	scanner := bufio.NewScanner(cmdOutErrReader)

	for scanner.Scan() {
		line = scanner.Text()

		fmt.Println(colCya + line + colRes)

		if cmdOutErrReader.typ == "out" {
			// look for flag strings in stdout
		}
	}
}

func (cmdInWriter *writecl) scanner() {
	var line string
	var err error

	reader := bufio.NewReader(os.Stdin)

	for {
		line, err = reader.ReadString('\n')
		if err != nil {
			debugctrl.Logger("cmdInWriter scanner:", err.Error())
			continue
		}

		cmdInWriter.Write([]byte(line))
	}
}
