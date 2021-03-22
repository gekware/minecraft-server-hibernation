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

	setIpPorts()

	data.LoadIcon(Config.Basic.ServerDirPath)

	errStr := checkConfig()
	if errStr != "" {
		log.Println("loadConfig: config error:", errStr)
		os.Exit(1)
	}
}

func setIpPorts() {
	ListenHost = "0.0.0.0"
	TargetHost = "127.0.0.1"

	serverPropertiesFilePath := filepath.Join(Config.Basic.ServerDirPath, "server.properties")
	data, err := ioutil.ReadFile(serverPropertiesFilePath)
	if err != nil {
		debugctrl.Logger("confctrl: setIpPorts:", err.Error())
	}

	dataStr := string(data)
	dataStr = strings.ReplaceAll(dataStr, "\r", "")
	TargetPort = strings.Split(strings.Split(dataStr, "server-port=")[1], "\n")[0]

	if TargetPort == Config.Advanced.ListenPort {
		log.Fatalln("TargetPort and ListenPort appear to be the same, please change one of them")
	}

	debugctrl.Logger("targeting server address:", TargetHost+":"+TargetPort)
}

// SaveConfig saves the config struct to the config file
func SaveConfig() {
	// write the struct config to json data
	configData, err := json.MarshalIndent(Config, "", "  ")
	if err != nil {
		debugctrl.Logger("forwardSync: could not marshal configuration")
		return
	}

	// write json data to config.json
	err = ioutil.WriteFile("config.json", configData, 0644)
	if err != nil {
		debugctrl.Logger("forwardSync: could not update config.json")
		return
	}

	debugctrl.Logger("saved to config.json")
}

// checks different parameters
func checkConfig() string {
	// check if serverFile/serverFolder exists
	// (if config.Basic.ServerFileName == "", then it will just check if the server folder exist)
	serverFileFolderPath := filepath.Join(Config.Basic.ServerDirPath, Config.Basic.ServerFileName)
	_, err := os.Stat(serverFileFolderPath)
	if os.IsNotExist(err) {
		return fmt.Sprintf("specified server file/folder does not exist: %s", serverFileFolderPath)
	}

	// check if java is installed
	_, err = exec.LookPath("java")
	if err != nil {
		return "java not installed!"
	}

	// if StopMinecraftServerForce is not set, set it equal to StopMinecraftServer
	if Config.Basic.StopMinecraftServerForce == "" {
		Config.Basic.StopMinecraftServerForce = Config.Basic.StopMinecraftServer
	}

	return ""
}
