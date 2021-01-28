package confctrl

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"msh/lib/data"
	"msh/lib/debugctrl"
)

// Config contains the configuration parameters
var Config configuration

// TimeLeftUntilUp keeps track of how many seconds are still needed to reach serverStatus == "online"
var TimeLeftUntilUp int

// struct adapted to config.json
type configuration struct {
	Basic    basic
	Advanced advanced
}
type basic struct {
	ServerDirPath                 string
	ServerFileName                string
	StartMinecraftServerLin       string
	StopMinecraftServerLin        string
	ForceStopMinecraftServerLin   string
	StartMinecraftServerWin       string
	StopMinecraftServerWin        string
	ForceStopMinecraftServerWin   string
	StartMinecraftServerMac       string
	StopMinecraftServerMac        string
	ForceStopMinecraftServerMac   string
	HibernationInfo               string
	StartingInfo                  string
	MinecraftServerStartupTime    int
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

	initVariables()
}

// checks different paramenters
func checkConfig() string {
	//------------- windows linux macos -------------//

	// check if OS is windows/linux
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		log.Print("checkConfig: error: OS not supported!")
		os.Exit(1)
	}

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

	//------------------- linux -------------------//
	if runtime.GOOS == "linux" {
		if strings.Contains(Config.Basic.StartMinecraftServerLin, "screen") {
			_, err = exec.LookPath("screen")
			if err != nil {
				return "screen not installed!"
			}
		}
	}

	//------------------- macos -------------------//
	if runtime.GOOS == "darwin" {
		if strings.Contains(Config.Basic.StartMinecraftServerWin, "screen") {
			_, err = exec.LookPath("screen")
			if err != nil {
				return "screen not installed!"
			}
		}
	}

	return ""
}

// initializes some variables
func initVariables() {
	TimeLeftUntilUp = Config.Basic.MinecraftServerStartupTime

	// if server-icon-frozen.png is in ServerDirPath folder then load this icon
	userIconPath := filepath.Join(Config.Basic.ServerDirPath, "server-icon-frozen.png")
	if _, err := os.Stat(userIconPath); !os.IsNotExist(err) {
		loadIcon(userIconPath)
	}

	// Set force command to normal stop command if undefined
	if Config.Basic.ForceStopMinecraftServerLin == "" {
		Config.Basic.ForceStopMinecraftServerLin = Config.Basic.StopMinecraftServerLin
	}
	if Config.Basic.ForceStopMinecraftServerMac == "" {
		Config.Basic.ForceStopMinecraftServerMac = Config.Basic.StopMinecraftServerMac
	}
	if Config.Basic.ForceStopMinecraftServerWin == "" {
		Config.Basic.ForceStopMinecraftServerWin = Config.Basic.StopMinecraftServerWin
	}
}

func loadIcon(userIconPath string) {
	// this function loads userIconPath image (base-64 encoded and compressed)
	// into serverIcon variable

	buff := &bytes.Buffer{}
	enc := &png.Encoder{CompressionLevel: -3} // -3: best compression

	// Using a decoder to read and then an encoder to compress the image data

	// Open file
	f, err := os.Open(userIconPath)
	if err != nil {
		debugctrl.Logger("loadIcon: error opening icon file:", err.Error())
		return
	}
	defer f.Close()

	// Decode
	pngIm, err := png.Decode(f)
	if err != nil {
		debugctrl.Logger("loadIcon: error decoding icon:", err.Error())
		return
	}

	// Encode if image is 64x64
	if pngIm.Bounds().Max == image.Pt(64, 64) {
		err = enc.Encode(buff, pngIm)
		if err != nil {
			debugctrl.Logger("loadIcon: error encoding icon:", err.Error())
			return
		}
		data.ServerIcon = base64.RawStdEncoding.EncodeToString(buff.Bytes())
	} else {
		log.Printf("loadIcon: incorrect server-icon-frozen.png size. Current size: %dx%d", pngIm.Bounds().Max.X, pngIm.Bounds().Max.Y)
	}
}
