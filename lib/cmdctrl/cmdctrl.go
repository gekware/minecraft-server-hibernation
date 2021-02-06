package cmdctrl

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
	in       io.WriteCloser
	out      io.ReadCloser
	err      io.ReadCloser
}

// Start starts a new terminal (non-blocking) and returns a servTerm object
func Start(dir, command string) (*ServTerm, error) {
	var err error

	term := &ServTerm{}

	commandSplit := strings.Split(command, " ")

	term.cmd = exec.Command(commandSplit[0], commandSplit[1:]...)
	term.cmd.Dir = dir

	term.out, err = term.cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	term.err, err = term.cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	term.in, err = term.cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	go term.printOut()
	go term.printErr()

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

func (term *ServTerm) printOut() {
	var line string

	scanner := bufio.NewScanner(term.out)

	for scanner.Scan() {
		line = scanner.Text()

		fmt.Println(line)
	}
}

func (term *ServTerm) printErr() {
	var line string

	scanner := bufio.NewScanner(term.err)

	for scanner.Scan() {
		line = scanner.Text()

		fmt.Println(line)
	}
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
