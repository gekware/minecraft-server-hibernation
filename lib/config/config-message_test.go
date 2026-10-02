package config

import (
	"os"
	"testing"
)

func Test_loadDefault_startingMessage(t *testing.T) {
	// Keep both the config and any generated msh.instance in a temporary directory.
	t.Chdir(t.TempDir())
	savedConfigDefaultSave := configDefaultSave
	t.Cleanup(func() { configDefaultSave = savedConfigDefaultSave })

	tests := []struct {
		name string
		json string
		want string
	}{
		{"missing setting", `{"Msh":{}}`, "Server start command issued. Please wait..."},
		{"custom setting", `{"Msh":{"MsgStarting":"Attendi, il server si sta avviando…"}}`, "Attendi, il server si sta avviando…"},
		{"explicit empty setting", `{"Msh":{"MsgStarting":""}}`, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := os.WriteFile(configFileName, []byte(test.json), 0600); err != nil {
				t.Fatal(err)
			}
			var c Configuration
			if logMsh := c.loadDefault(); logMsh != nil {
				t.Fatalf("loadDefault failed: %v", logMsh)
			}
			if got := c.Msh.MsgStarting; got != test.want {
				t.Errorf("MsgStarting = %q, want %q", got, test.want)
			}
		})
	}
}
