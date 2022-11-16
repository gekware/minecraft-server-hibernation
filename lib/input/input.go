package input

import (
	"bufio"
	"io"
	"os"
	"strings"

	"msh/lib/errco"
	"msh/lib/progmgr"
	"msh/lib/servctrl"
	"msh/lib/servstats"
)

// GetInput is used to read input from user.
// [goroutine]
func GetInput() {
	var line string
	var err error

	reader := bufio.NewReader(os.Stdin)

	for {
		line, err = reader.ReadString('\n')
		if err != nil {
			// if stdin is unavailable (msh running without terminal console)
			// exit from input goroutine to avoid an infinite loop
			if err == io.EOF {
				// in case input goroutine returns abnormally while msh is running in terminal,
				// the user must be notified with errco.LVL_1
				errco.Logln(errco.TYPE_WAR, errco.LVL_1, errco.ERROR_INPUT_EOF, "stdin unavailable, exiting input goroutine")
				return
			}
			errco.Logln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_INPUT_READ, err.Error())
			continue
		}

		// make sure that only 1 space separates words
		line = strings.ReplaceAll(line, "\n", "")
		line = strings.ReplaceAll(line, "\r", "")
		line = strings.ReplaceAll(line, "\t", " ")
		for strings.Contains(line, "  ") {
			line = strings.ReplaceAll(line, "  ", " ")
		}
		lineSplit := strings.Split(line, " ")

		errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "user input: %s", lineSplit[:])

		switch lineSplit[0] {
		// target msh
		case "msh":
			// check that there is a command for the target
			if len(lineSplit) < 2 {
				errco.Logln(errco.TYPE_WAR, errco.LVL_0, errco.ERROR_COMMAND_INPUT, "specify msh command (start - freeze - exit)")
				continue
			}

			switch lineSplit[1] {
			case "start":
				logMsh := servctrl.WarmMS()
				if logMsh != nil {
					errco.Log(logMsh.AddTrace())
				}
			case "freeze":
				// stop minecraft server forcefully
				logMsh := servctrl.FreezeMS(true)
				if logMsh != nil {
					errco.Log(logMsh.AddTrace())
				}
			case "exit":
				// stop minecraft server forcefully
				logMsh := servctrl.FreezeMS(true)
				if logMsh != nil {
					errco.Log(logMsh.AddTrace())
				}
				// exit msh
				errco.Logln(errco.TYPE_INF, errco.LVL_0, errco.ERROR_NIL, "issuing msh termination")
				progmgr.AutoTerminate()
			default:
				errco.Logln(errco.TYPE_WAR, errco.LVL_0, errco.ERROR_COMMAND_UNKNOWN, "unknown command (start - freeze - exit)")
			}

		// taget minecraft server
		case "mine":
			// check that there is a command for the target
			if len(lineSplit) < 2 {
				errco.Logln(errco.TYPE_WAR, errco.LVL_0, errco.ERROR_COMMAND_INPUT, "specify mine command")
				continue
			}

			// check if server is online
			if servstats.Stats.Status != errco.SERVER_STATUS_ONLINE {
				errco.Logln(errco.TYPE_ERR, errco.LVL_0, errco.ERROR_SERVER_NOT_ONLINE, "minecraft server is not online (try \"msh start\")")
				continue
			}

			// pass the command to the minecraft server terminal
			_, logMsh := servctrl.Execute(strings.Join(lineSplit[1:], " "), "user input")
			if logMsh != nil {
				errco.Log(logMsh.AddTrace())
			}

		// wrong target
		default:
			errco.Logln(errco.TYPE_WAR, errco.LVL_0, errco.ERROR_COMMAND_INPUT, "specify the target application by adding \"msh\" or \"mine\" before the command.\nExample to get op: mine op <yourname>\nExample to freeze minecraft: msh freeze")
		}
	}
}
