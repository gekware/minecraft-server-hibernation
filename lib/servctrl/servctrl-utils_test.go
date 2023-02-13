package servctrl

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func Test_getPlayersByListCom(t *testing.T) {
	output := []string{
		"[12:34:56] [Server thread/INFO]: There are 0 of a max of 20 players online:",
		"[12:34:56] [Server INFO]: There are 0 out of maximum 20 players online.",

		"[12:34:56] [Server ERROR]: There are 0 out of maximum 20 players online",
		"[12:34:56] [Server INFO]: There are no numbers here",

		"[12:34:56 INFO]: [Essentials] CONSOLE issued server command: /list\n[12:34:56 INFO]: There are 0 of a max of 20 players online",

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
		
		var firstNumber string
		if strings.Contains(o, "Essentials") {
			t.Logf("string contains \"Essentials\"")
			firstNumber = regexp.MustCompile(`\d+`).FindString(strings.Split(o, "INFO]:")[2])
		} else {
			t.Logf("string does not contain \"Essentials\"")
			firstNumber = regexp.MustCompile(`\d+`).FindString(strings.Split(o, "INFO]:")[1])
		}

		// check if firstNumber has been found
		if firstNumber == "" {
			t.Logf("firstNumber string is empty")
			continue
		}

		players, err := strconv.Atoi(firstNumber)
		if err != nil {
			t.Fatalf(err.Error())
		}

		// TEST: check return value
		if players != expected {
			t.Fatalf("player count not expected")
		}
	}
}
