package config

import (
	"archive/zip"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"msh/lib/errco"
	"msh/lib/model"
	"msh/lib/opsys"

	"github.com/denisbrodbeck/machineid"
)

var (
	configFileName string = "msh-config.json" // configFileName is the config file name

	ConfigDefault *Configuration = &Configuration{} // ConfigDefault contains parameters of config in file
	ConfigRuntime *Configuration = &Configuration{} // ConfigRuntime contains parameters of config in runtime.

	Javav string // Javav is the java version on the system. format: "java 16.0.1 2021-04-20"

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

	// set server and protocol version from server JAR file if not specified in config/arguments.
	// required for backward-compatibility and for minecraft versions without a version.info inside the JAR file.
	// (see https://minecraft.fandom.com/wiki/Version.json)
	errMsh = ConfigRuntime.extractVersionInfo()
	if errMsh != nil {
		return errMsh.AddTrace("LoadConfig")
	}

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
	// get msh executable path
	mshPath, err := os.Executable()
	if err != nil {
		return errco.NewErr(errco.ERROR_CONFIG_LOAD, errco.LVL_B, "ConfigDefaultFileRead", err.Error())
	}

	// read config file
	configData, err := ioutil.ReadFile(filepath.Join(filepath.Dir(mshPath), configFileName))
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

	if id, err := machineid.ProtectedID("msh"); err != nil {
		return errco.NewErr(errco.ERROR_CONFIG_LOAD, errco.LVL_D, "loadDefault", err.Error())
	} else if ex, err := os.Executable(); err != nil {
		return errco.NewErr(errco.ERROR_CONFIG_LOAD, errco.LVL_D, "loadDefault", err.Error())
	} else {
		hasher := sha1.New()
		hasher.Write([]byte(id + filepath.Dir(ex)))
		clientID := hex.EncodeToString(hasher.Sum(nil))
		if c.Msh.ID != clientID {
			c.Msh.ID = clientID
			errMsh := c.Save()
			if errMsh != nil {
				return errMsh.AddTrace("loadDefault")
			}
		}
	}

	return nil
}

// loadRuntime parses start arguments into config and replaces placeholders
func (c *Configuration) loadRuntime(base *Configuration) *errco.Error {
	// initialize config to base
	*c = *base

	// specify arguments
	flag.StringVar(&ConfigRuntime.Server.FileName, "file", ConfigRuntime.Server.FileName, "Specify server file name.")
	flag.StringVar(&ConfigRuntime.Server.Folder, "folder", ConfigRuntime.Server.Folder, "Specify server folder path.")

	flag.StringVar(&ConfigRuntime.Commands.StartServerParam, "msparam", ConfigRuntime.Commands.StartServerParam, "Specify start server parameters.")
	flag.IntVar(&ConfigRuntime.Commands.StopServerAllowKill, "allowKill", ConfigRuntime.Commands.StopServerAllowKill, "Specify after how many seconds the server should be killed (if stop command fails).")

	flag.IntVar(&ConfigRuntime.Msh.Debug, "d", ConfigRuntime.Msh.Debug, "Specify debug level.")
	flag.StringVar(&ConfigRuntime.Msh.InfoHibernation, "infohibe", ConfigRuntime.Msh.InfoHibernation, "Specify hibernation info.")
	flag.StringVar(&ConfigRuntime.Msh.InfoStarting, "infostar", ConfigRuntime.Msh.InfoStarting, "Specify starting info.")
	flag.IntVar(&ConfigRuntime.Msh.ListenPort, "port", ConfigRuntime.Msh.ListenPort, "Specify msh port.")
	flag.Int64Var(&ConfigRuntime.Msh.TimeBeforeStoppingEmptyServer, "timeout", ConfigRuntime.Msh.TimeBeforeStoppingEmptyServer, "Specify time to wait before stopping minecraft server.")

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
	} else {
		Javav = strings.ReplaceAll(strings.Split(string(out), "\n")[0], "\r", "")
	}

	return nil
}

// extractVersionInfo reads version.json from the server JAR file
// and sets the correct minecraft version and protocol in config
func (c *Configuration) extractVersionInfo() *errco.Error {
	if c.Server.Version != "" && c.Server.Protocol != 0 {
		errco.LogMshErr(errco.NewErr(errco.ERROR_VERSION_LOAD, errco.LVL_D, "extractVersionInfo", "minecraft server version and protocol already specified"))
		return nil
	}

	reader, err := zip.OpenReader(filepath.Join(c.Server.Folder, c.Server.FileName))
	if err != nil {
		return errco.NewErr(errco.ERROR_VERSION_LOAD, errco.LVL_D, "extractVersionInfo", err.Error())
	}
	defer reader.Close()

	for _, file := range reader.File {
		// search for version.json file
		if file.Name != "version.json" {
			continue
		}

		f, err := file.Open()
		if err != nil {
			return errco.NewErr(errco.ERROR_VERSION_LOAD, errco.LVL_D, "extractVersionInfo", err.Error())
		}
		defer f.Close()

		versionsBytes, err := ioutil.ReadAll(f)
		if err != nil {
			return errco.NewErr(errco.ERROR_VERSION_LOAD, errco.LVL_D, "extractVersionInfo", err.Error())
		}

		var info model.VersionInfo
		err = json.Unmarshal(versionsBytes, &info)
		if err != nil {
			return errco.NewErr(errco.ERROR_VERSION_LOAD, errco.LVL_D, "extractVersionInfo", err.Error())
		}

		if c.Server.Version == "" {
			c.Server.Version = info.Version
		}
		if c.Server.Protocol == 0 {
			c.Server.Protocol = info.Protocol
		}

		// update and save default config
		ConfigDefault.Server.Version = info.Version
		ConfigDefault.Server.Protocol = info.Protocol
		errMsh := ConfigDefault.Save()
		if errMsh != nil {
			return errMsh.AddTrace("extractVersionInfo")
		}
	}

	// just log the error since version and protocol are not vital for connection to clients
	// (and might still be extracted while retrieving server info)
	errco.LogMshErr(errco.NewErr(errco.ERROR_VERSION_LOAD, errco.LVL_D, "extractVersionInfo", "minecraft server version and protocol could not be extracted from version.json"))
	return nil
}
