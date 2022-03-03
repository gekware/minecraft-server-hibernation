package config

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"msh/lib/errco"
	"msh/lib/model"
	"msh/lib/opsys"
	"msh/lib/utility"
)

var (
	configFileName string = "msh-config.json" // configFileName is the config file name

	ConfigDefault *Configuration = &Configuration{} // ConfigDefault contains parameters of config in file
	ConfigRuntime *Configuration = &Configuration{} // ConfigRuntime contains parameters of config in runtime.

	Javav string // Javav is the java version on the system. format: "16.0.1"

	ServerIcon string // ServerIcon contains the minecraft server icon

	ListenHost string = "0.0.0.0"   // ListenHost is the ip address for clients to connect to msh
	ListenPort int                  // ListenPort is the port for clients to connect to msh
	TargetHost string = "127.0.0.1" // TargetHost is the ip address for msh to connect to minecraft server
	TargetPort int                  // TargetPort is the port for msh to connect to minecraft server
)

type Configuration struct {
	model.Configuration
}

// LoadConfig loads config file into default/runtime config.
// should be the first function to be called by main.
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

	// --------------- config runtime -------------- //
	//  from now only config runtime should be used  //

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

// Save saves config to the config file
func (c *Configuration) Save() *errco.Error {
	// encode the struct config
	configData, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return errco.NewErr(errco.ERROR_CONFIG_SAVE, errco.LVL_D, "Save", "could not marshal from config file")
	}

	// write to config file
	err = ioutil.WriteFile(configFileName, configData, 0644)
	if err != nil {
		return errco.NewErr(errco.ERROR_CONFIG_SAVE, errco.LVL_D, "Save", "could not write to config file")
	}

	errco.Logln(errco.LVL_D, "Save: saved to config file")

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

	// ------------------- checks ------------------ //

	// check that msh id is healthy
	// if not generate a new one and save to config
	if strings.Contains(c.Msh.ID, "feefe75") {
		errco.Logln(errco.LVL_D, "loadDefault: generating new msh id...")

		rand.Seed(time.Now().UnixNano())
		b := make([]byte, 16)
		for strings.Contains(c.Msh.ID, "feefe75") {
			_, err := rand.Read(b)
			if err != nil {
				return errco.NewErr(errco.ERROR_CONFIG_LOAD, errco.LVL_B, "loadDefault", err.Error())
			}
			c.Msh.ID = hex.EncodeToString(b)
		}

		errco.Logln(errco.LVL_D, "loadDefault: new msh id: %s", c.Msh.ID)
		c.Save()
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

	// ------------------- checks ------------------ //

	// check if serverFile/serverFolder exists
	serverFileFolderPath := filepath.Join(c.Server.Folder, c.Server.FileName)
	_, err := os.Stat(serverFileFolderPath)
	if os.IsNotExist(err) {
		return errco.NewErr(errco.ERROR_CONFIG_CHECK, errco.LVL_B, "check", "specified server file/folder does not exist: "+serverFileFolderPath)
	}

	// check if java is installed and get java version
	_, err = exec.LookPath("java")
	if err != nil {
		return errco.NewErr(errco.ERROR_CONFIG_CHECK, errco.LVL_B, "check", "java not installed")
	} else if out, err := exec.Command("java", "--version").Output(); err != nil {
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
