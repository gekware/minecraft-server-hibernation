package config

import (
	"flag"
	"os"
	"testing"

	"msh/lib/errco"
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

func Test_loadRuntime_connectTimeout(t *testing.T) {
	t.Chdir(t.TempDir())
	// Stop unrelated server setup at its command check, before any process runs.
	if err := os.WriteFile("server.jar", nil, 0600); err != nil {
		t.Fatal(err)
	}
	savedFlags, savedUsage, savedArgs := flag.CommandLine, flag.Usage, os.Args
	savedDebugLvl := errco.DebugLvl
	t.Cleanup(func() {
		flag.CommandLine, flag.Usage, os.Args = savedFlags, savedUsage, savedArgs
		errco.DebugLvl = savedDebugLvl
	})

	tests := []struct {
		name   string
		config int
		args   []string
		want   int
	}{
		{"default timeout", 60, nil, 60},
		{"custom timeout", 120, nil, 120},
		{"CLI override", 120, []string{"-logintimeout", "300"}, 300},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			flag.CommandLine = flag.NewFlagSet(t.Name(), flag.ContinueOnError)
			os.Args = append([]string{"msh"}, test.args...)
			var confdef, runtime Configuration
			confdef.Server.Folder = "."
			confdef.Server.FileName = "server.jar"
			confdef.Msh.ConnectTimeoutSeconds = test.config
			logMsh := runtime.loadRuntime(&confdef)
			if logMsh == nil || logMsh.Cod != errco.ERROR_INVALID_COMMAND {
				t.Fatalf("expected server command check, got %v", logMsh)
			}
			if got := runtime.Msh.ConnectTimeoutSeconds; got != test.want {
				t.Errorf("runtime timeout = %d, want %d", got, test.want)
			}
			if confdef.Msh.ConnectTimeoutSeconds != test.config {
				t.Error("runtime loading changed the file configuration timeout")
			}
		})
	}
}
