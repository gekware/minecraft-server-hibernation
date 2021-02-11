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
)

// ServTerm is the minecraft server terminal
type ServTerm struct {
	IsActive bool
	Wg       sync.WaitGroup
	cmd      *exec.Cmd
	in       wc
	out      rc
	err      rc
}

// rc inherits io.ReadCloser and a string is used to indentify it as "out" or "err"
type rc struct {
	io.ReadCloser
	string
}

// wc inherits io.WriteCloser
type wc struct {
	io.WriteCloser
}

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

	term.out = rc{stdOut, "out"}
	term.err = rc{stdErr, "err"}
	term.in = wc{stdIn}

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
}

func (stdOutErr *rc) printer() {
	var line string

	scanner := bufio.NewScanner(stdOutErr)

	for scanner.Scan() {
		line = scanner.Text()

		fmt.Println(line)

		if stdOutErr.string == "out" {
			// look for signal strings in stdout
		}
	}
}
