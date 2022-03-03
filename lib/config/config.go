package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"msh/lib/errco"
	"msh/lib/model"
	"msh/lib/opsys"
	"msh/lib/utility"
)

// configFileName is the config file name
const configFileName string = "msh-config.json"

type Configuration struct {
	model.Configuration
}

var (
	// ConfigDefault contains the configuration parameters of config in file
	ConfigDefault *Configuration = &Configuration{}
	// ConfigRuntime contains the configuration parameters of config during runtime.
	// (can be altered during runtime without affecting the config file)
	ConfigRuntime *Configuration = &Configuration{}

	// Javav is the java version on the system.
	// format: "16.0.1"
	Javav string

	// ServerIcon contains the minecraft server icon
	ServerIcon string

	ListenHost string = "0.0.0.0"   // ListenHost is the ip address for clients to connect to msh
	ListenPort int                  // ListenPort is the port for clients to connect to msh
	TargetHost string = "127.0.0.1" // TargetHost is the ip address for msh to connect to minecraft server
	TargetPort int                  // TargetPort is the port for msh to connect to minecraft server
)

// LoadConfig loads config file into ConfigDefault and ConfigRuntime
func LoadConfig() *errco.Error {
	// ---------------- OS support ----------------- //

	errco.Logln(errco.LVL_D, "checking OS support...")

	// check if OS is supported.
	errMsh := opsys.OsSupported()
	if errMsh != nil {
		return errMsh.AddTrace("LoadConfig")
	}

	// ---------------- load config ---------------- //

	errco.Logln(errco.LVL_D, "loading config...")

	// load config default
	errMsh = ConfigDefault.loadDefault()
	if errMsh != nil {
		return errMsh.AddTrace("LoadConfig")
	}

	// load config runtime
	errMsh = ConfigRuntime.loadRuntime(ConfigDefault)
	if errMsh != nil {
		return errMsh.AddTrace("LoadConfig")
	}

	errMsh = ConfigRuntime.check()
	if errMsh != nil {
		return errMsh.AddTrace("LoadConfig")
	}

	// --------------- config runtime -------------- //
	// from now on only ConfigRuntime should be used //

	// as soon as the config variables are set, set debug level
	// (up until now the default errco.DebugLvl is LVL_E)
	errco.Logln(errco.LVL_A, "setting log level to: %d", ConfigRuntime.Msh.Debug)
	errco.DebugLvl = ConfigRuntime.Msh.Debug

	// initialize ip and ports for connection
	ListenHost, ListenPort, TargetHost, TargetPort, errMsh = ConfigRuntime.getIpPorts()
	if errMsh != nil {
		return errMsh.AddTrace("LoadConfig")
	}

	errco.Logln(errco.LVL_D, "msh proxy setup: %s:%d --> %s:%d", ListenHost, ListenPort, TargetHost, TargetPort)

	// set server icon
	ServerIcon, errMsh = ConfigRuntime.loadIcon()
	if errMsh != nil {
		// it's enough to log it without returning
		// since the default icon is loaded by default
		errco.LogMshErr(errMsh.AddTrace("LoadConfig"))
	}

	return nil
}

// loadDefault loads config file to config variable
func (c *Configuration) loadDefault() *errco.Error {
	// read config file
	configData, err := ioutil.ReadFile(configFileName)
	if err != nil {
		return errco.NewErr(errco.ERROR_CONFIG_LOAD, errco.LVL_B, "loadDefault", err.Error())
	}

	// write data to config variable
	err = json.Unmarshal(configData, &c)
	if err != nil {
		return errco.NewErr(errco.ERROR_CONFIG_LOAD, errco.LVL_B, "loadDefault", err.Error())
	}

	return nil
}

// loadRuntime parses start arguments into config and replaces placeholders
func (c *Configuration) loadRuntime(base *Configuration) *errco.Error {
	// initialize config to base
	*c = *base

	// specify arguments
	flag.StringVar(&c.Server.FileName, "f", c.Server.FileName, "Specify server file name.")
	flag.StringVar(&c.Server.Folder, "F", c.Server.Folder, "Specify server folder path.")

	flag.StringVar(&c.Commands.StartServerParam, "P", c.Commands.StartServerParam, "Specify start server parameters.")

	flag.IntVar(&c.Msh.ListenPort, "p", c.Msh.ListenPort, "Specify msh port.")
	flag.StringVar(&c.Msh.InfoHibernation, "h", c.Msh.InfoHibernation, "Specify hibernation info.")
	flag.StringVar(&c.Msh.InfoStarting, "s", c.Msh.InfoStarting, "Specify starting info.")
	flag.IntVar(&c.Msh.Debug, "d", c.Msh.Debug, "Specify debug level.")

	// specify the usage when there is an error in the arguments
	flag.Usage = func() {
		// not using errco.Logln since log time is not needed
		fmt.Println("Usage of msh:")
		flag.PrintDefaults()
	}

	// parse arguments
	flag.Parse()

	// replace placeholders
	c.Commands.StartServer = strings.ReplaceAll(c.Commands.StartServer, "<Server.FileName>", c.Server.FileName)
	c.Commands.StartServer = strings.ReplaceAll(c.Commands.StartServer, "<Commands.StartServerParam>", c.Commands.StartServerParam)

	return nil
}

// Save saves config to the config file
func (c *Configuration) Save() *errco.Error {
	// encode the struct config
	configData, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return errco.NewErr(errco.ERROR_CONFIG_SAVE, errco.LVL_D, "save", "could not marshal from config file")
	}

	// write to config file
	err = ioutil.WriteFile(configFileName, configData, 0644)
	if err != nil {
		return errco.NewErr(errco.ERROR_CONFIG_SAVE, errco.LVL_D, "save", "could not write to config file")
	}

	errco.Logln(errco.LVL_D, "save: saved to config file")

	return nil
}

// check checks different parameters in config
func (c *Configuration) check() *errco.Error {
	// check if serverFile/serverFolder exists
	// (if config.Basic.ServerFileName == "", then it will just check if the server folder exist)
	serverFileFolderPath := filepath.Join(c.Server.Folder, c.Server.FileName)
	_, err := os.Stat(serverFileFolderPath)
	if os.IsNotExist(err) {
		return errco.NewErr(errco.ERROR_CONFIG_CHECK, errco.LVL_B, "check", "specified server file/folder does not exist: "+serverFileFolderPath)
	}

	// check if java is installed
	_, err = exec.LookPath("java")
	if err != nil {
		return errco.NewErr(errco.ERROR_CONFIG_CHECK, errco.LVL_B, "check", "java not installed")
	}

	// check java version
	if out, err := exec.Command("java", "--version").Output(); err != nil {
		// non blocking error
		errco.LogMshErr(errco.NewErr(errco.ERROR_CONFIG_CHECK, errco.LVL_D, "check", "could not execute 'java -version' command"))
		Javav = "unknown"
	} else if j, errMsh := utility.StrBetween(string(out), "java ", " "); errMsh != nil {
		// non blocking error
		errco.LogMshErr(errco.NewErr(errco.ERROR_CONFIG_CHECK, errco.LVL_D, "check", "could not extract java version"))
		Javav = "unknown"
	} else {
		Javav = j
	}

	return nil
}

// getIpPorts reads server.properties server file and returns the correct ports
func (c *Configuration) getIpPorts() (string, int, string, int, *errco.Error) {
	data, err := ioutil.ReadFile(filepath.Join(c.Server.Folder, "server.properties"))
	if err != nil {
		return "", -1, "", -1, errco.NewErr(errco.ERROR_CONFIG_LOAD, errco.LVL_B, "getIpPorts", err.Error())
	}

	dataStr := strings.ReplaceAll(string(data), "\r", "")

	TargetPortStr, errMsh := utility.StrBetween(dataStr, "server-port=", "\n")
	if errMsh != nil {
		return "", -1, "", -1, errMsh.AddTrace("getIpPorts")
	}

	TargetPort, err = strconv.Atoi(TargetPortStr)
	if err != nil {
		return "", -1, "", -1, errco.NewErr(errco.ERROR_CONVERSION, errco.LVL_D, "getIpPorts", err.Error())
	}

	if TargetPort == ConfigRuntime.Msh.ListenPort {
		return "", -1, "", -1, errco.NewErr(errco.ERROR_CONFIG_LOAD, errco.LVL_B, "getIpPorts", "TargetPort and ListenPort appear to be the same, please change one of them")
	}

	// return ListenHost, TargetHost, TargetPort, nil
	return ListenHost, ConfigRuntime.Msh.ListenPort, TargetHost, TargetPort, nil
}

// IsWhitelisted checks if the playerName or clientAddress is whitelisted
func IsWhitelisted(params ...string) *errco.Error {
	// check if whitelist is enabled
	// if empty then it is not enabled and no checks are needed
	if len(ConfigRuntime.Msh.Whitelist) == 0 {
		errco.Logln(errco.LVL_D, "whitelist not enabled")
		return nil
	}

	errco.Logln(errco.LVL_D, "checking whitelist for: %s", strings.Join(params, ", "))

	// check if playerName or clientAddress are in whitelist
	for _, p := range params {
		if utility.SliceContain(p, ConfigRuntime.Msh.Whitelist) {
			errco.Logln(errco.LVL_D, "playerName or clientAddress is whitelisted!")
			return nil
		}
	}

	// playerName or clientAddress not found in whitelist
	errco.Logln(errco.LVL_D, "playerName or clientAddress is not whitelisted!")
	return errco.NewErr(errco.ERROR_PLAYER_NOT_IN_WHITELIST, errco.LVL_B, "IsWhitelisted", "playerName or clientAddress is not whitelisted")
}
