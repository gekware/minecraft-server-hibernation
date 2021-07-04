package servctrl

import (
	"fmt"
	"strconv"

	"msh/lib/debugctrl"
	"msh/lib/utility"
)

// CountPlayerSafe returns the number of players on the server.
// The /list command is used for safety and if it fails, internal player count is returned.
// No error is returned: the integer is always meaningful
// (might be more or less reliable depending from where it retrieved).
// A bool param is returned indicating if player count comes from
// the internal player count (false) or the server player count (true).
func CountPlayerSafe() (int, bool) {
	playerCount, err := getPlayersByListCom()
	if err != nil {
		// no need to return an error since the less reliable internal player count is available
		debugctrl.Logln("CountPlayerSafe: %v", err)
		return Stats.PlayerCount, false
	}

	return playerCount, true
}

// getPlayersByListCom returns the number of players using the /list command
func getPlayersByListCom() (int, error) {
	outStr, err := Execute("/list", "getPlayersByListCom")
	if err != nil {
		return 0, fmt.Errorf("getPlayersByListCom: %v", err)
	}
	playersStr, err := utility.StrBetween(outStr, "There are ", " of a max")
	if err != nil {
		return 0, fmt.Errorf("getPlayersByListCom: %v", err)
	}
	players, err := strconv.Atoi(playersStr)
	if err != nil {
		return 0, fmt.Errorf("getPlayersByListCom: %v", err)
	}

	return players, nil
}
