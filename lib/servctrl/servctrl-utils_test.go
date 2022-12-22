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
	}
	expected := 0

	for _, o := range output {
		// TEST: reproduce function behaviour

		// check test function for possible `list` outputs
		firstNumber := regexp.MustCompile(`\d+`).FindString(strings.Split(o, "INFO]:")[1])

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
