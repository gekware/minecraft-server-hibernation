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

	"msh/lib/data"
	"msh/lib/logger"
)

// ConfigDefault contains the configuration parameters of the config file
var ConfigDefault configuration

// ConfigRuntime contains the configuration parameters during runtime
var ConfigRuntime configuration

var ListenHost string
var TargetHost string
var TargetPort string

// struct adapted to config.json
type configuration struct {
	Server struct {
		Folder   string `json:"Folder"`
		FileName string `json:"FileName"`
		Protocol string `json:"Protocol"`
		Version  string `json:"Version"`
	} `json:"Server"`
	Commands struct {
		StartServer         string `json:"StartServer"`
		StartServerParam    string `json:"StartServerParam"`
		StopServer          string `json:"StopServer"`
		StopServerAllowKill int    `json:"StopServerAllowKill"`
	} `json:"Commands"`
	Msh struct {
		Debug                         bool   `json:"Debug"`
		InfoHibernation               string `json:"InfoHibernation"`
		InfoStarting                  string `json:"InfoStarting"`
		NotifyUpdate                  bool   `json:"NotifyUpdate"`
		Port                          string `json:"Port"`
		TimeBeforeStoppingEmptyServer int64  `json:"TimeBeforeStoppingEmptyServer"`
	} `json:"Msh"`
}

// LoadConfig loads json data from config.json into config
func LoadConfig() error {
	// read config.json
	configData, err := ioutil.ReadFile("config.json")
	if err != nil {
		return fmt.Errorf("loadConfig: %v", err)
	}

	// write read data into struct config
	err = json.Unmarshal(configData, &ConfigDefault)
	if err != nil {
		return fmt.Errorf("loadConfig: %v", err)
	}

	// initialize runtime config
	ConfigRuntime = ConfigDefault

	setUpConfigRuntime()

	err = checkConfigRuntime()
	if err != nil {
		return fmt.Errorf("loadConfig: %v", err)
	}

	// as soon as the Config variable is set, set debug level
	logger.Debug = ConfigRuntime.Msh.Debug

	err = setIpPorts()
	if err != nil {
		return fmt.Errorf("loadConfig: %v", err)
	}

	err = data.LoadIcon(ConfigRuntime.Server.Folder)
	if err != nil {
		// it's enough to log it without returning
		// since the default icon is loaded by default
		logger.Logln("loadConfig:", err.Error())
	}

	return nil
}

// SaveConfigDefault saves ConfigDefault to the config file
func SaveConfigDefault() error {
	// write the struct config to json data
	configData, err := json.MarshalIndent(ConfigDefault, "", "  ")
	if err != nil {
		return fmt.Errorf("SaveConfig: could not marshal from config.json")
	}

	// write json data to config.json
	err = ioutil.WriteFile("config.json", configData, 0644)
	if err != nil {
		return fmt.Errorf("SaveConfig: could not write to config.json")
	}

	logger.Logln("SaveConfig: saved to config.json")

	return nil
}

// setUpConfigRuntime parses start arguments into ConfigRuntime and replaces placeholders
func setUpConfigRuntime() {
	// specify arguments
	flag.StringVar(&ConfigRuntime.Server.FileName, "f", ConfigRuntime.Server.FileName, "Specify server file name.")
	flag.StringVar(&ConfigRuntime.Server.Folder, "F", ConfigRuntime.Server.Folder, "Specify server folder path.")

	flag.StringVar(&ConfigRuntime.Commands.StartServerParam, "P", ConfigRuntime.Commands.StartServerParam, "Specify start server parameters.")

	flag.StringVar(&ConfigRuntime.Msh.Port, "p", ConfigRuntime.Msh.Port, "Specify msh port.")
	flag.StringVar(&ConfigRuntime.Msh.InfoHibernation, "h", ConfigRuntime.Msh.InfoHibernation, "Specify hibernation info.")
	flag.StringVar(&ConfigRuntime.Msh.InfoStarting, "s", ConfigRuntime.Msh.InfoStarting, "Specify starting info.")
	flag.BoolVar(&ConfigRuntime.Msh.Debug, "d", ConfigRuntime.Msh.Debug, "Set debug to true.")

	// specify the usage when there is an error in the arguments
	flag.Usage = func() {
		fmt.Printf("Usage of msh:\n")
		flag.PrintDefaults()
	}

	// parse arguments
	flag.Parse()

	// replace placeholders in StartServer command in ConfigRuntime
	ConfigRuntime.Commands.StartServer = strings.ReplaceAll(ConfigRuntime.Commands.StartServer, "<Server.FileName>", ConfigRuntime.Server.FileName)
	ConfigRuntime.Commands.StartServer = strings.ReplaceAll(ConfigRuntime.Commands.StartServer, "<Commands.StartServerParam>", ConfigRuntime.Commands.StartServerParam)
}

// checkConfigRuntime checks different parameters in ConfigRuntime
func checkConfigRuntime() error {
	// check if serverFile/serverFolder exists
	// (if config.Basic.ServerFileName == "", then it will just check if the server folder exist)
	serverFileFolderPath := filepath.Join(ConfigRuntime.Server.Folder, ConfigRuntime.Server.FileName)
	_, err := os.Stat(serverFileFolderPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("checkConfig: specified server file/folder does not exist: %s", serverFileFolderPath)
	}

	// check if java is installed
	_, err = exec.LookPath("java")
	if err != nil {
		return fmt.Errorf("checkConfig: java not installed")
	}

	return nil
}

// setIpPorts reads server.properties server file and sets the correct ports
func setIpPorts() error {
	ListenHost = "0.0.0.0"
	TargetHost = "127.0.0.1"

	serverPropertiesFilePath := filepath.Join(ConfigRuntime.Server.Folder, "server.properties")
	data, err := ioutil.ReadFile(serverPropertiesFilePath)
	if err != nil {
		return fmt.Errorf("setIpPorts: %v", err)
	}

	dataStr := string(data)
	dataStr = strings.ReplaceAll(dataStr, "\r", "")
	TargetPort = strings.Split(strings.Split(dataStr, "server-port=")[1], "\n")[0]

	if TargetPort == ConfigRuntime.Msh.Port {
		return fmt.Errorf("setIpPorts: TargetPort and ListenPort appear to be the same, please change one of them")
	}

	logger.Logln("targeting server address:", TargetHost+":"+TargetPort)

	return nil
}
