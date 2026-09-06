package servctrl

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"msh/lib/utility"
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

func Test_extractServInfo(t *testing.T) {
	// mountResponse builds a minecraft server info response around a json payload:
	// [ packet length (VarInt) | packet id (VarInt, 0) | json length (VarInt) | json ]
	mountResponse := func(jsonData []byte) []byte {
		data := append([]byte{0}, utility.EncodeVarInt(len(jsonData))...)
		data = append(data, jsonData...)
		return append(utility.EncodeVarInt(len(data)), data...)
	}

	// json payloads whose length puts the header at 3, 5 and 7 bytes:
	// only the 5 bytes case worked when the header size was hardcoded
	tests := []struct {
		title    string
		jsonData []byte
	}{
		{
			"short description, no favicon (3 bytes header)",
			[]byte(`{"description":{"text":""},"players":{"max":20,"online":0},"version":{"name":"1.19.2","protocol":760}}`),
		},
		{
			"regular response (5 bytes header)",
			[]byte(`{"description":{"text":"` + strings.Repeat("a", 1000) + `"}}`),
		},
		{
			"big favicon (7 bytes header)",
			[]byte(`{"description":{"text":"` + strings.Repeat("a", 20000) + `"}}`),
		},
	}

	for _, test := range tests {
		fmt.Printf("testing \"%s\"\n", test.title)

		extracted, logMsh := extractServInfo(mountResponse(test.jsonData))
		if logMsh != nil {
			t.Errorf("\t\"%s\": %s\n", test.title, fmt.Sprintf(logMsh.Mex, logMsh.Arg...))
			continue
		}

		if !bytes.Equal(extracted, test.jsonData) {
			t.Errorf("\t\"%s\": extracted json is different from expected\n\textracted: %s\n", test.title, string(extracted))
		}
	}

	// negative cases
	negative := []struct {
		title string
		data  []byte
	}{
		{"empty response", []byte{}},
		{"truncated header", []byte{128}},
		{"json shorter than declared", append([]byte{10, 0, 8}, []byte("abc")...)},
		{"packet id is not 0", []byte{3, 9, 1, 65}},
	}

	for _, test := range negative {
		if _, logMsh := extractServInfo(test.data); logMsh == nil {
			t.Errorf("\t\"%s\": expected an error, got none\n", test.title)
		}
	}
}
