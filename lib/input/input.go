package input

import (
	"io"
	"log"
	"strings"

	"msh/lib/errco"
	"msh/lib/progmgr"
	"msh/lib/servctrl"
	"msh/lib/servstats"

	"github.com/chzyer/readline"
)

// GetInput is used to read input from user.
// [goroutine]
func GetInput() {
	l, err := readline.NewEx(
		&readline.Config{
			Prompt: "» ",
			AutoComplete: readline.NewPrefixCompleter(
				readline.PcItem("msh",
					readline.PcItem("start"),
					readline.PcItem("freeze"),
					readline.PcItem("exit"),
				),
				readline.PcItem("mine"),
			),
			FuncFilterInputRune: func(r rune) (rune, bool) {
				switch r {
				case readline.CharCtrlZ: // block CtrlZ feature
					return r, false
				}
				return r, true
			},
		})
	if err != nil {
		errco.NewLogln(errco.TYPE_ERR, errco.LVL_1, errco.ERROR_INPUT, "error while starting readline: %s", err.Error())
		return
	}
	defer l.Close()

	log.SetOutput(l.Stderr()) // autorefresh prompt line
	for {
		line, err := l.Readline()
		switch err {
		case nil:
			// analyze line
		case io.EOF:
			// if stdin is unavailable (msh running without terminal console)
			// exit from input goroutine to avoid an infinite loop
			// (user must be notified with errco.LVL_1)
			errco.NewLogln(errco.TYPE_WAR, errco.LVL_1, errco.ERROR_INPUT_EOF, "stdin unavailable, exiting input goroutine")
			return
		case readline.ErrInterrupt:
			progmgr.AutoTerminate()
			return
		default:
			errco.NewLogln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_INPUT_READ, err.Error())
			continue
		}

		// make sure that only 1 space separates words
		lineSplit := strings.Fields(line)

		errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "user input: %s", lineSplit[:])

		// prevent crash when input is empty
		if len(lineSplit) == 0 {
			continue
		}

		switch lineSplit[0] {
		// target msh
		case "msh":
			// check that there is a command for the target
			if len(lineSplit) < 2 {
				errco.NewLogln(errco.TYPE_WAR, errco.LVL_0, errco.ERROR_COMMAND_INPUT, "specify msh command (start - freeze - exit)")
				continue
			}

			switch lineSplit[1] {

			case "start":
				logMsh := servctrl.WarmMS()
				if logMsh != nil {
					logMsh.Log(true)
				}
			case "freeze":
				// stop minecraft server forcefully
				logMsh := servctrl.FreezeMS(true)
				if logMsh != nil {
					logMsh.Log(true)
				}
			case "exit":
				// stop minecraft server forcefully
				logMsh := servctrl.FreezeMS(true)
				if logMsh != nil {
					logMsh.Log(true)
				}
				// terminate msh
				progmgr.AutoTerminate()
			default:
				errco.NewLogln(errco.TYPE_WAR, errco.LVL_0, errco.ERROR_COMMAND_UNKNOWN, "unknown command (start - freeze - exit)")
			}

		// taget minecraft server
		case "mine":
			// check that there is a command for the target
			if len(lineSplit) < 2 {
				errco.NewLogln(errco.TYPE_WAR, errco.LVL_0, errco.ERROR_COMMAND_INPUT, "specify mine command")
				continue
			}

			// check if server is online
			if servstats.Stats.Status != errco.SERVER_STATUS_ONLINE {
				errco.NewLogln(errco.TYPE_ERR, errco.LVL_0, errco.ERROR_SERVER_NOT_ONLINE, "minecraft server is not online (try \"msh start\")")
				continue
			}

			// pass the command to the minecraft server terminal
			_, logMsh := servctrl.Execute(strings.Join(lineSplit[1:], " "))
			if logMsh != nil {
				logMsh.Log(true)
			}

		// wrong target
		default:
			errco.NewLogln(errco.TYPE_WAR, errco.LVL_0, errco.ERROR_COMMAND_INPUT, "specify the target application by adding \"msh\" or \"mine\" before the command.\nExample to get op: mine op <yourname>\nExample to freeze minecraft: msh freeze")
		}
	}
}
