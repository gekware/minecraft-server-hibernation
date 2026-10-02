package config

import (
	"os"
	"testing"
)

func Test_loadDefault_connectTimeout(t *testing.T) {
	// Keep both the config and any generated msh.instance in a temporary directory.
	t.Chdir(t.TempDir())
	savedConfigDefaultSave := configDefaultSave
	t.Cleanup(func() { configDefaultSave = savedConfigDefaultSave })

	tests := []struct {
		name string
		json string
		want int
	}{
		{"missing setting", `{"Msh":{}}`, 60},
		{"custom setting", `{"Msh":{"ConnectTimeoutSeconds":120}}`, 120},
		{"explicit zero", `{"Msh":{"ConnectTimeoutSeconds":0}}`, 0},
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
			if got := c.Msh.ConnectTimeoutSeconds; got != test.want {
				t.Errorf("ConnectTimeoutSeconds = %d, want %d", got, test.want)
			}
		})
	}
}
