package servctrl

import (
	"strconv"

	"msh/lib/errco"
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
		return Stats.PlayerCount, false
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
		return 0, errco.NewErr(errco.STRING_CONVERSION_ERROR, errco.LVL_D, "getPlayersByListCom", err.Error())
	}

	return players, nil
}
