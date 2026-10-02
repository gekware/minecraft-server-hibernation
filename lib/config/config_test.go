package config

import (
	"flag"
	"math"
	"os"
	"strconv"
	"testing"
	"time"

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
		{"negative setting", `{"Msh":{"ConnectTimeoutSeconds":-1}}`, -1},
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

	maxTimeout := int64(math.MaxInt64) / int64(time.Second)
	if strconv.IntSize == 32 {
		maxTimeout = math.MaxInt32
	}
	type timeoutCase struct {
		name    string
		config  int
		args    []string
		want    int
		wantErr errco.LogCod
	}
	tests := []timeoutCase{
		{"default timeout", 60, nil, 60, errco.ERROR_INVALID_COMMAND},
		{"custom timeout", 120, nil, 120, errco.ERROR_INVALID_COMMAND},
		{"minimum timeout", 1, nil, 1, errco.ERROR_INVALID_COMMAND},
		{"CLI override", 120, []string{"-logintimeout", "300"}, 300, errco.ERROR_INVALID_COMMAND},
		{"CLI overrides invalid config", 0, []string{"-logintimeout", "120"}, 120, errco.ERROR_INVALID_COMMAND},
		{"zero config", 0, nil, 0, errco.ERROR_CONFIG_CHECK},
		{"negative config", -1, nil, -1, errco.ERROR_CONFIG_CHECK},
		{"zero CLI", 60, []string{"-logintimeout", "0"}, 0, errco.ERROR_CONFIG_CHECK},
		{"negative CLI", 60, []string{"-logintimeout", "-1"}, -1, errco.ERROR_CONFIG_CHECK},
		{"maximum timeout", int(maxTimeout), nil, int(maxTimeout), errco.ERROR_INVALID_COMMAND},
	}
	if strconv.IntSize == 64 {
		overflow := int(maxTimeout + 1)
		tests = append(tests,
			timeoutCase{"overflow config", overflow, nil, overflow, errco.ERROR_CONFIG_CHECK},
			timeoutCase{"overflow CLI", 60, []string{"-logintimeout", strconv.Itoa(overflow)}, overflow, errco.ERROR_CONFIG_CHECK},
		)
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
			if logMsh == nil || logMsh.Cod != test.wantErr {
				t.Fatalf("loadRuntime error = %v, want code %v", logMsh, test.wantErr)
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
