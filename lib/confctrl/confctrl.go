package confctrl

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"msh/lib/data"
	"msh/lib/debugctrl"
)

// Config contains the configuration parameters
var Config configuration

var ListenHost string
var TargetHost string
var TargetPort string

// struct adapted to config.json
type configuration struct {
	Basic struct {
		ServerDirPath                 string `json:"ServerDirPath"`
		ServerFileName                string `json:"ServerFileName"`
		StartMinecraftServer          string `json:"StartMinecraftServer"`
		StopMinecraftServer           string `json:"StopMinecraftServer"`
		StopMinecraftServerForce      string `json:"StopMinecraftServerForce"`
		HibernationInfo               string `json:"HibernationInfo"`
		StartingInfo                  string `json:"StartingInfo"`
		TimeBeforeStoppingEmptyServer int    `json:"TimeBeforeStoppingEmptyServer"`
		CheckForUpdates               bool   `json:"CheckForUpdates"`
	} `json:"Basic"`
	Advanced struct {
		ListenPort     string `json:"ListenPort"`
		Debug          bool   `json:"Debug"`
		ServerVersion  string `json:"ServerVersion"`
		ServerProtocol string `json:"ServerProtocol"`
	} `json:"Advanced"`
}

// LoadConfig loads json data from config.json into config
func LoadConfig() {
	// read config.json
	configData, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Println("loadConfig:", err.Error())
		os.Exit(1)
	}
	// write read data into struct config
	err = json.Unmarshal(configData, &Config)
	if err != nil {
		log.Println("loadConfig:", err.Error())
		os.Exit(1)
	}

	// as soon the config is loaded, set debug level for debugctrl
	debugctrl.Debug = Config.Advanced.Debug

	err = setIpPorts()
	if err != nil {
		log.Fatalln("confctrl: loadConfig:", err.Error())
	}

	data.LoadIcon(Config.Basic.ServerDirPath)

	err = checkConfig()
	if err != nil {
		log.Fatalln("confctrl: loadConfig: checkConfig:", err.Error())
	}
}

// SaveConfig saves the config struct to the config file
func SaveConfig() error {
	// write the struct config to json data
	configData, err := json.MarshalIndent(Config, "", "  ")
	if err != nil {
		return fmt.Errorf("SaveConfig: could not marshal from config.json")
	}

	// write json data to config.json
	err = ioutil.WriteFile("config.json", configData, 0644)
	if err != nil {
		return fmt.Errorf("SaveConfig: could not write to config.json")
	}

	debugctrl.Logger("SaveConfig: saved to config.json")

	return nil
}

func setIpPorts() error {
	ListenHost = "0.0.0.0"
	TargetHost = "127.0.0.1"

	serverPropertiesFilePath := filepath.Join(Config.Basic.ServerDirPath, "server.properties")
	data, err := ioutil.ReadFile(serverPropertiesFilePath)
	if err != nil {
		return fmt.Errorf("setIpPorts: %v", err.Error())
	}

	dataStr := string(data)
	dataStr = strings.ReplaceAll(dataStr, "\r", "")
	TargetPort = strings.Split(strings.Split(dataStr, "server-port=")[1], "\n")[0]

	if TargetPort == Config.Advanced.ListenPort {
		return fmt.Errorf("setIpPorts: TargetPort and ListenPort appear to be the same, please change one of them")
	}

	debugctrl.Logger("targeting server address:", TargetHost+":"+TargetPort)

	return nil
}

// checks different parameters
func checkConfig() error {
	// check if serverFile/serverFolder exists
	// (if config.Basic.ServerFileName == "", then it will just check if the server folder exist)
	serverFileFolderPath := filepath.Join(Config.Basic.ServerDirPath, Config.Basic.ServerFileName)
	_, err := os.Stat(serverFileFolderPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("checkConfig: specified server file/folder does not exist: %s", serverFileFolderPath)
	}

	// check if java is installed
	_, err = exec.LookPath("java")
	if err != nil {
		return fmt.Errorf("checkConfig: java not installed")
	}

	// if StopMinecraftServerForce is not set, set it equal to StopMinecraftServer
	if Config.Basic.StopMinecraftServerForce == "" {
		Config.Basic.StopMinecraftServerForce = Config.Basic.StopMinecraftServer
	}

	return nil
}
