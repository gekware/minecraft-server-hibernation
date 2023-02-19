package servctrl

import (
	"testing"
)

func Test_searchListCom(t *testing.T) {
	type test struct {
		str    string
		expNum int
		expErr bool
	}

	var tests []test = []test{
		// positive cases [vanilla]
		{
			"[12:34:56] [Server thread/INFO]: There are 0 of a max of 20 players online:",
			0,
			false,
		},
		{
			"[12:34:56] [Server INFO]: There are 0 out of maximum 20 players online.",
			0,
			false,
		},

		// positive cases [plugins]
		{
			"[12:34:56 INFO]: Es sind 0 von maximal 15 Spielern online.", // [EssentialsX]
			0,
			false,
		},
		{
			"[12:34:56 INFO]: [Essentials] CONSOLE issued server command: /list\n[12:34:56 INFO]: Es sind 0 von maximal 15 Spielern online.", // [EssentialsX]
			0,
			false,
		},
		{
			"[12:34:56 Server thread/INFO]: Ci sono 0 giocatori online su un massimo di 20.", // [EssentialsX]
			0,
			false,
		},
		{
			"[12:34:56 Server thread/INFO]: CONSOLE issued server command: /list\n[12:34:56 Server thread/INFO]: Ci sono 0 giocatori online su un massimo di 20.", // [EssentialsX]
			0,
			false,
		},

		// negative cases [plugins]
		{
			"[12:34:56 INFO]: [Essentials] CONSOLE issued server command: /list", // [EssentialsX]
			-1,
			true,
		},

		// negative cases [example]
		{
			"[12:34:56] [Server ERROR]: There are 0 out of maximum 20 players online",
			-1,
			true,
		},
		{
			"[12:34:56] [Server INFO]: Example where there are no numbers",
			-1,
			true,
		},
	}

	for _, tt := range tests {
		n, logMsh := searchListCom(tt.str)
		if logMsh != nil {
			if tt.expErr {
				continue
			}
			t.Error("function returned unexpected error")
		}

		if n != tt.expNum {
			t.Error("function returned unexpected number")
		}
	}
}
