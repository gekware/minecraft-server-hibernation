package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"msh/lib/errco"
	"msh/lib/opsys"
)

// configFileName is the config file name
const configFileName string = "msh-config.yml"

var (
	// Config variables contain the configuration parameters for config file and runtime
	ConfigDefault configuration
	ConfigRuntime configuration

	// ServerIcon contains the minecraft server icon
	ServerIcon string

	// Listen and Target host/port used for proxy connection
	ListenHost string
	ListenPort string
	TargetHost string
	TargetPort string
)

// struct adapted to config file
type configuration struct {
	Server struct {
		Folder   string `yaml:"Folder"`
		FileName string `yaml:"FileName"`
		Protocol string `yaml:"Protocol"`
		Version  string `yaml:"Version"`
	} `yaml:"Server"`
	Commands struct {
		StartServer         string `yaml:"StartServer"`
		StartServerParam    string `yaml:"StartServerParam"`
		StopServer          string `yaml:"StopServer"`
		StopServerAllowKill int    `yaml:"StopServerAllowKill"`
	} `yaml:"Commands"`
	Msh struct {
		Debug                         int    `yaml:"Debug"`
		InfoHibernation               string `yaml:"InfoHibernation"`
		InfoStarting                  string `yaml:"InfoStarting"`
		NotifyUpdate                  bool   `yaml:"NotifyUpdate"`
		Port                          string `yaml:"Port"`
		TimeBeforeStoppingEmptyServer int64  `yaml:"TimeBeforeStoppingEmptyServer"`
	} `yaml:"Msh"`
}

// LoadConfig loads json data from config file into config
func LoadConfig() *errco.Error {
	errco.Logln(errco.LVL_D, "checking OS support...")
	// check if OS is supported.
	errMsh := opsys.OsSupported()
	if errMsh != nil {
		return errMsh.AddTrace("LoadConfig")
	}

	errco.Logln(errco.LVL_D, "loading config file...")
	// read config file
	configData, err := ioutil.ReadFile(configFileName)
	if err != nil {
		return errco.NewErr(errco.LOAD_CONFIG_ERROR, errco.LVL_B, "LoadConfig", err.Error())
	}

	// write read data into ConfigDefault
	err = yaml.Unmarshal(configData, &ConfigDefault)
	if err != nil {
		return errco.NewErr(errco.LOAD_CONFIG_ERROR, errco.LVL_B, "LoadConfig", err.Error())
	}

	// generate runtime config
	ConfigRuntime = generateConfigRuntime()

	// --------------- ConfigRuntime --------------- //
	// from now on only ConfigRuntime should be used //

	errMsh = checkConfigRuntime()
	if errMsh != nil {
		return errMsh.AddTrace("LoadConfig")
	}

	// as soon as the Config variable is set, set debug level
	// (up until now the default errco.DebugLvl is LVL_E)
	errco.DebugLvl = ConfigRuntime.Msh.Debug
	// LVL_A log level is used to always notice the user of the log level
	errco.Logln(errco.LVL_A, "log level set to: %d", errco.DebugLvl)

	// initialize ip and ports for connection
	ListenHost, ListenPort, TargetHost, TargetPort, errMsh = getIpPorts()
	if errMsh != nil {
		return errMsh.AddTrace("LoadConfig")
	}
	errco.Logln(errco.LVL_D, "msh proxy setup:\t%s:%s --> %s:%s", ListenHost, ListenPort, TargetHost, TargetPort)

	// set server icon
	ServerIcon, errMsh = loadIcon(ConfigRuntime.Server.Folder)
	if errMsh != nil {
		// it's enough to log it without returning
		// since the default icon is loaded by default
		errco.LogMshErr(errMsh.AddTrace("LoadConfig"))
	}

	return nil
}

// SaveConfigDefault saves ConfigDefault to the config file
func SaveConfigDefault() *errco.Error {
	// write the struct config to json data
	configData, err := json.MarshalIndent(ConfigDefault, "", "  ")
	if err != nil {
		return errco.NewErr(errco.SAVE_CONFIG_ERROR, errco.LVL_D, "SaveConfigDefault", "could not marshal from config file")
	}

	// write json data to config file
	err = ioutil.WriteFile(configFileName, configData, 0644)
	if err != nil {
		return errco.NewErr(errco.SAVE_CONFIG_ERROR, errco.LVL_D, "SaveConfigDefault", "could not write to config file")
	}

	errco.Logln(errco.LVL_B, "SaveConfigDefault: saved to config file")

	return nil
}

// generateConfigRuntime parses start arguments into ConfigRuntime and replaces placeholders
func generateConfigRuntime() configuration {
	// initialize with ConfigDefault
	ConfigRuntime = ConfigDefault

	// specify arguments
	flag.StringVar(&ConfigRuntime.Server.FileName, "f", ConfigRuntime.Server.FileName, "Specify server file name.")
	flag.StringVar(&ConfigRuntime.Server.Folder, "F", ConfigRuntime.Server.Folder, "Specify server folder path.")

	flag.StringVar(&ConfigRuntime.Commands.StartServerParam, "P", ConfigRuntime.Commands.StartServerParam, "Specify start server parameters.")

	flag.StringVar(&ConfigRuntime.Msh.Port, "p", ConfigRuntime.Msh.Port, "Specify msh port.")
	flag.StringVar(&ConfigRuntime.Msh.InfoHibernation, "h", ConfigRuntime.Msh.InfoHibernation, "Specify hibernation info.")
	flag.StringVar(&ConfigRuntime.Msh.InfoStarting, "s", ConfigRuntime.Msh.InfoStarting, "Specify starting info.")
	flag.IntVar(&ConfigRuntime.Msh.Debug, "d", ConfigRuntime.Msh.Debug, "Specify debug level.")

	// specify the usage when there is an error in the arguments
	flag.Usage = func() {
		// not using errco.Logln since log time is not needed
		fmt.Println("Usage of msh:")
		flag.PrintDefaults()
	}

	// parse arguments
	flag.Parse()

	// replace placeholders in ConfigRuntime StartServer command
	ConfigRuntime.Commands.StartServer = strings.ReplaceAll(ConfigRuntime.Commands.StartServer, "<Server.FileName>", ConfigRuntime.Server.FileName)
	ConfigRuntime.Commands.StartServer = strings.ReplaceAll(ConfigRuntime.Commands.StartServer, "<Commands.StartServerParam>", ConfigRuntime.Commands.StartServerParam)

	return ConfigRuntime
}

// checkConfigRuntime checks different parameters in ConfigRuntime
func checkConfigRuntime() *errco.Error {
	// check if serverFile/serverFolder exists
	// (if config.Basic.ServerFileName == "", then it will just check if the server folder exist)
	serverFileFolderPath := filepath.Join(ConfigRuntime.Server.Folder, ConfigRuntime.Server.FileName)
	_, err := os.Stat(serverFileFolderPath)
	if os.IsNotExist(err) {
		return errco.NewErr(errco.CHECK_CONFIG_ERROR, errco.LVL_B, "checkConfigRuntime", "specified server file/folder does not exist: "+serverFileFolderPath)
	}

	// check if java is installed
	_, err = exec.LookPath("java")
	if err != nil {
		return errco.NewErr(errco.CHECK_CONFIG_ERROR, errco.LVL_B, "checkConfigRuntime", "java not installed")
	}

	return nil
}

// getIpPorts reads server.properties server file and returns the correct ports
func getIpPorts() (string, string, string, string, *errco.Error) {
	serverPropertiesFilePath := filepath.Join(ConfigRuntime.Server.Folder, "server.properties")
	data, err := ioutil.ReadFile(serverPropertiesFilePath)
	if err != nil {
		return "", "", "", "", errco.NewErr(errco.LOAD_CONFIG_ERROR, errco.LVL_B, "setIpPorts", err.Error())
	}

	dataStr := string(data)
	dataStr = strings.ReplaceAll(dataStr, "\r", "")
	TargetPort = strings.Split(strings.Split(dataStr, "server-port=")[1], "\n")[0]

	if TargetPort == ConfigRuntime.Msh.Port {
		return "", "", "", "", errco.NewErr(errco.LOAD_CONFIG_ERROR, errco.LVL_B, "setIpPorts", "TargetPort and ListenPort appear to be the same, please change one of them")
	}

	// return ListenHost, TargetHost, TargetPort, nil
	return "0.0.0.0", ConfigRuntime.Msh.Port, "127.0.0.1", TargetPort, nil
}
