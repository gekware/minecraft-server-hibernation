package servctrl

import (
	"strconv"
	"time"

	"msh/lib/errco"
	"msh/lib/servstats"
	"msh/lib/utility"
)

// countPlayerSafe returns the number of players on the server.
// The /list command is used for safety and if it fails, internal player count is returned.
// No error is returned: the integer is always meaningful
// (might be more or less reliable depending from where it retrieved).
// A bool param is returned indicating if player count comes from
// the internal player count (false) or the server player count (true).
func countPlayerSafe() (int, bool) {
	playerCount, errMsh := getPlayersByListCom()
	if errMsh != nil {
		// no need to return an error since the less reliable internal player count is available
		errco.LogMshErr(errMsh.AddTrace("countPlayerSafe"))
		return servstats.Stats.PlayerCount, false
	}

	return playerCount, true
}

// getPlayersByListCom returns the number of players using the /list command
func getPlayersByListCom() (int, *errco.Error) {
	outStr, errMsh := Execute("list", "getPlayersByListCom")
	if errMsh != nil {
		return 0, errMsh.AddTrace("getPlayersByListCom")
	}
	playersStr, errMsh := utility.StrBetween(outStr, "There are ", " of a max")
	if errMsh != nil {
		return 0, errMsh.AddTrace("getPlayersByListCom")
	}
	players, err := strconv.Atoi(playersStr)
	if err != nil {
		return 0, errco.NewErr(errco.CONVERSION_ERROR, errco.LVL_D, "getPlayersByListCom", err.Error())
	}

	return players, nil
}

// printDataUsage prints each second bytes/s to clients and to server.
// (must be launched after ServTerm.IsActive has been set to true)
// [goroutine]
func printDataUsage() {
	for ServTerm.IsActive {
		if servstats.Stats.BytesToClients != 0 || servstats.Stats.BytesToServer != 0 {
			errco.Logln(errco.LVL_D, "data/s: %8.3f KB/s to clients | %8.3f KB/s to server", servstats.Stats.BytesToClients/1024, servstats.Stats.BytesToServer/1024)

			servstats.Stats.M.Lock()
			servstats.Stats.BytesToClients = 0
			servstats.Stats.BytesToServer = 0
			servstats.Stats.M.Unlock()
		}

		time.Sleep(time.Second)
	}
}
