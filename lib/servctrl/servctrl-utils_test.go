package servctrl

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func Test_getPlayersByListCom(t *testing.T) {
	output := []string{
		// possible output after sending /list command

		// positive cases [vanilla]
		"[12:34:56] [Server thread/INFO]: There are 0 of a max of 20 players online:",
		"[12:34:56] [Server INFO]: There are 0 out of maximum 20 players online.",

		// positive cases [plugins]
		"[12:01:34 INFO]: Es sind 0 von maximal 15 Spielern online.", // [EssentialsX]

		// negative cases [plugins]
		"[12:34:56 INFO]: [Essentials] CONSOLE issued server command: /list", // [EssentialsX]

		// negative cases [example]
		"[12:34:56] [Server ERROR]: There are 0 out of maximum 20 players online",
		"[12:34:56] [Server INFO]: Example where there are no numbers",
	}
	expected := 0

	for _, o := range output {
		// TEST: reproduce function behaviour

		// continue if output has unexpected format
		if !strings.Contains(o, "INFO]:") {
			t.Logf("string does not contain \"INFO]:\"")
			continue
		}
		// check test function for possible `list` outputs, also check for Essentials plugin

		playerCount := regexp.MustCompile(` \d+ `).FindString(o)
		playerCount = strings.ReplaceAll(playerCount, " ", "")

		// check if playerCount has been found
		if playerCount == "" {
			t.Logf("playerCount string is empty")
			continue
		}

		players, err := strconv.Atoi(playerCount)
		if err != nil {
			t.Fatalf(err.Error())
		}

		// TEST: check return value
		if players != expected {
			t.Fatalf("player count not expected")
		}
	}
}
