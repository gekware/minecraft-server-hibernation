package servctrl

import (
	"fmt"
	"strconv"

	"msh/lib/utility"
)

// CountPlayerSafe returns the number of players on the server.
// The /list command is used for safety and if it fails,
// internal player count is returned.
// The error returned is non blocking, the integer returned
// will be in any case more or less reliable.
func CountPlayerSafe() (int, error) {
	playerCount, err := getPlayersByListCom()
	if err != nil {
		return ServStats.Players, fmt.Errorf("CountPlayerSafe: %v", err)
	}

	return playerCount, nil
}

// getPlayersByListCom returns the number of players using the /list command
func getPlayersByListCom() (int, error) {
	outStr, err := ServTerminal.Execute("/list")
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
