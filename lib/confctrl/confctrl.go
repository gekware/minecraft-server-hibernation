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

// struct adapted to config.json
type configuration struct {
	Basic    basic
	Advanced advanced
}
type basic struct {
	ServerDirPath                 string
	ServerFileName                string
	StartMinecraftServer          string
	StopMinecraftServer           string
	StopMinecraftServerForce      string
	HibernationInfo               string
	StartingInfo                  string
	TimeBeforeStoppingEmptyServer int
	CheckForUpdates               bool
}
type advanced struct {
	ListenHost     string
	ListenPort     string
	TargetHost     string
	TargetPort     string
	Debug          bool
	ServerVersion  string
	ServerProtocol string
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

	error := checkConfig()
	if error != "" {
		log.Println("loadConfig: config error:", error)
		os.Exit(1)
	}

	data.LoadIcon(Config.Basic.ServerDirPath)

	debugctrl.Debug = Config.Advanced.Debug
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

// checks different paramenters
func checkConfig() string {
	//------------- windows linux macos -------------//
	// check if serverFile/serverFolder exists
	// (if config.Basic.ServerFileName == "", then it will just check if the server folder exist)
	serverFileFolderPath := filepath.Join(Config.Basic.ServerDirPath, Config.Basic.ServerFileName)
	debugctrl.Logger("Checking for " + serverFileFolderPath)
	_, err := os.Stat(serverFileFolderPath)
	if os.IsNotExist(err) {
		return fmt.Sprintf("specified server file/folder does not exist: %s", serverFileFolderPath)
	}

	// check if java is installed
	_, err = exec.LookPath("java")
	if err != nil {
		return "java not installed!"
	}

	if strings.Contains(Config.Basic.StartMinecraftServer, "screen") {
		_, err = exec.LookPath("screen")
		if err != nil {
			return "screen not installed!"
		}
	}

	// if StopMinecraftServerForce is not set, set it equal to StopMinecraftServer
	if Config.Basic.StopMinecraftServerForce == "" {
		Config.Basic.StopMinecraftServerForce = Config.Basic.StopMinecraftServer
	}

	return ""
}
