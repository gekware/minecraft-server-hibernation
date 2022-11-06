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

	"msh/lib/errco"
	"msh/lib/model"
	"msh/lib/opsys"
	"msh/lib/servstats"
	"msh/lib/utility"
)

var (
	configFileName string = "msh-config.json" // configFileName is the config file name

	ConfigDefault *Configuration = &Configuration{} // ConfigDefault contains parameters of config in file
	ConfigRuntime *Configuration = &Configuration{} // ConfigRuntime contains parameters of config in runtime

	configDefaultSave bool = false // if true, the config will be saved after successful loading

	Javav string // Javav is the java version on the system. format: "java 16.0.1 2021-04-20"

	ServerIcon string = defaultServerIcon // ServerIcon contains the minecraft server icon

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

	errco.Logln(errco.LVL_3, "checking OS support...")

	// check if OS is supported.
	errMsh := opsys.OsSupported()
	if errMsh != nil {
		return errMsh.AddTrace("LoadConfig")
	}

	// ---------------- load config ---------------- //

	errco.Logln(errco.LVL_3, "loading config...")

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

	// ---------------- save config ---------------- //

	if configDefaultSave {
		errMsh := ConfigDefault.Save()
		if errMsh != nil {
			return errMsh.AddTrace("LoadConfig")
		}
	}

	return nil
}

// Save saves config to the config file.
// Then does the default config setup
func (c *Configuration) Save() *errco.Error {
	// encode the struct config
	configData, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return errco.NewErr(errco.ERROR_CONFIG_SAVE, errco.LVL_3, "Save", "could not marshal from config file")
	}

	// escape unicode characters ("\u003c" to "<" and "\u003e" to ">")
	configData, errMsh := utility.UnicodeEscape(configData)
	if errMsh != nil {
		errco.LogWarn(errMsh.AddTrace("Save"))
	}

	// write to config file
	err = ioutil.WriteFile(configFileName, configData, 0644)
	if err != nil {
		return errco.NewErr(errco.ERROR_CONFIG_SAVE, errco.LVL_3, "Save", "could not write to config file")
	}

	errco.Logln(errco.LVL_3, "saved default config to config file")

	return nil
}

// loadDefault loads config file to config variable
func (c *Configuration) loadDefault() *errco.Error {
	// get working directory
	cwdPath, err := os.Getwd()
	if err != nil {
		return errco.NewErr(errco.ERROR_CONFIG_LOAD, errco.LVL_1, "loadDefault", err.Error())
	}

	// read config file
	configFilePath := filepath.Join(cwdPath, configFileName)
	errco.Logln(errco.LVL_3, "reading config file: \"%s\"", configFilePath)
	configData, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return errco.NewErr(errco.ERROR_CONFIG_LOAD, errco.LVL_1, "loadDefault", err.Error())
	}

	// write data to config variable
	err = json.Unmarshal(configData, &c)
	if err != nil {
		return errco.NewErr(errco.ERROR_CONFIG_LOAD, errco.LVL_1, "loadDefault", err.Error())
	}

	// ------------------- setup ------------------- //

	// load mshid
	mi := MshID()
	if c.Configuration.Msh.ID != mi {
		errco.LogWarn(errco.NewErr(errco.ERROR_CONFIG_LOAD, errco.LVL_3, "loadDefault", "config msh id different from instance msh id, applying correction..."))
		c.Configuration.Msh.ID = mi
		configDefaultSave = true
	}

	// load ms version/protocol
	// (checkout version.json info: https://minecraft.fandom.com/wiki/Version.json)
	version, protocol, errMsh := c.getVersionInfo()
	if errMsh != nil {
		// just log warning since ms version/protocol are not vital for the connection with clients
		errco.LogWarn(errMsh.AddTrace("loadDefault"))
	} else if c.Server.Version != version || c.Server.Protocol != protocol {
		c.Server.Version = version
		c.Server.Protocol = protocol
		configDefaultSave = true
	}

	return nil
}

// loadRuntime initializes runtime config to default config.
// Then parses start arguments into runtime config, replaces placeholders and does the runtime config setup
func (c *Configuration) loadRuntime(confdef *Configuration) *errco.Error {
	// initialize config to base
	*c = *confdef

	// specify arguments
	flag.StringVar(&c.Server.Folder, "folder", c.Server.Folder, "Specify minecraft server folder path.")
	flag.StringVar(&c.Server.FileName, "file", c.Server.FileName, "Specify minecraft server file name.")
	flag.StringVar(&c.Server.Version, "version", c.Server.Version, "Specify minecraft server version.")
	flag.IntVar(&c.Server.Protocol, "protocol", c.Server.Protocol, "Specify minecraft server protocol.")

	flag.StringVar(&c.Commands.StartServerParam, "msparam", c.Commands.StartServerParam, "Specify start server parameters.")
	flag.IntVar(&c.Commands.StopServerAllowKill, "allowkill", c.Commands.StopServerAllowKill, "Specify after how many seconds the server should be killed (if stop command fails).")

	flag.IntVar(&c.Msh.Debug, "d", c.Msh.Debug, "Specify debug level.")
	flag.BoolVar(&c.Msh.AllowSuspend, "allowsuspend", c.Msh.AllowSuspend, "Specify if minecraft server process can be suspended.")
	flag.StringVar(&c.Msh.InfoHibernation, "infohibe", c.Msh.InfoHibernation, "Specify hibernation info.")
	flag.StringVar(&c.Msh.InfoStarting, "infostar", c.Msh.InfoStarting, "Specify starting info.")
	flag.BoolVar(&c.Msh.NotifyUpdate, "notifyupd", c.Msh.NotifyUpdate, "Specify if update notifications are allowed.")
	flag.BoolVar(&c.Msh.NotifyMessage, "notifymes", c.Msh.NotifyMessage, "Specify if message notifications are allowed.")
	flag.IntVar(&c.Msh.ListenPort, "port", c.Msh.ListenPort, "Specify msh port.")
	flag.Int64Var(&c.Msh.TimeBeforeStoppingEmptyServer, "timeout", c.Msh.TimeBeforeStoppingEmptyServer, "Specify time to wait before stopping minecraft server.")

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

	// after config variables are set, set debug level
	errco.Logln(errco.LVL_0, "setting log level to: %d", c.Msh.Debug)
	errco.DebugLvl = c.Msh.Debug

	// ------------------- setup ------------------- //

	// check if server folder/executeble exist
	serverFileFolderPath := filepath.Join(c.Server.Folder, c.Server.FileName)
	if _, err := os.Stat(serverFileFolderPath); os.IsNotExist(err) {
		// server folder/executeble does not exist

		servstats.Stats.SetMajorError(errco.NewErr(errco.ERROR_MINECRAFT_SERVER, errco.LVL_3, "loadRuntime", "specified minecraft server folder/file does not exist"))
		errco.LogMshErr(errco.NewErr(errco.ERROR_CONFIG_CHECK, errco.LVL_1, "loadRuntime", "specified server file/folder does not exist: "+serverFileFolderPath))

	} else {
		// server folder/executeble exist

		// check if eula.txt exists and is set to true
		eulaFilePath := filepath.Join(c.Server.Folder, "eula.txt")
		eulaData, err := ioutil.ReadFile(eulaFilePath)
		switch {
		case err != nil:
			// eula.txt does not exist

			errco.LogWarn(errco.NewErr(errco.ERROR_CONFIG_CHECK, errco.LVL_1, "loadRuntime", "could not read eula.txt file: "+eulaFilePath))

			// start server to generate eula.txt (and server.properties)
			errco.Logln(errco.LVL_3, "starting minecraft server to generate eula.txt file...")
			cSplit := strings.Split(c.Commands.StartServer, " ")
			cmd := exec.Command(cSplit[0], cSplit[1:]...)
			cmd.Dir = c.Server.Folder
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			fmt.Print(errco.COLOR_CYAN) // set color to server log color
			err = cmd.Run()
			fmt.Print(errco.COLOR_RESET) // reset color
			if err != nil {
				servstats.Stats.SetMajorError(errco.NewErr(errco.ERROR_MINECRAFT_SERVER, errco.LVL_3, "loadRuntime", "couldn't start minecraft server to generate eula.txt\n(are you using the correct java version?)"))
				errco.LogMshErr(errco.NewErr(errco.ERROR_TERMINAL_START, errco.LVL_1, "loadRuntime", "couldn't start minecraft server to generate eula.txt: ["+err.Error()+"]"))
			}
			fallthrough

		case !strings.Contains(strings.ReplaceAll(strings.ToLower(string(eulaData)), " ", ""), "eula=true"):
			// eula.txt exists but is not set to true

			servstats.Stats.SetMajorError(errco.NewErr(errco.ERROR_MINECRAFT_SERVER, errco.LVL_3, "loadRuntime", "please accept minecraft server eula.txt"))
			errco.LogMshErr(errco.NewErr(errco.ERROR_CONFIG_CHECK, errco.LVL_1, "loadRuntime", "please accept minecraft server eula.txt: "+eulaFilePath))

		default:
			// eula.txt exists and is set to true

			errco.Logln(errco.LVL_1, "eula.txt exist and is set to true")
		}
	}

	// check if java is installed and get java version
	_, err := exec.LookPath("java")
	if err != nil {
		servstats.Stats.SetMajorError(errco.NewErr(errco.ERROR_MINECRAFT_SERVER, errco.LVL_3, "loadRuntime", "java not installed"))
		errco.LogMshErr(errco.NewErr(errco.ERROR_CONFIG_CHECK, errco.LVL_1, "loadRuntime", "java not installed"))
	} else if out, err := exec.Command("java", "--version").Output(); err != nil {
		// non blocking error
		errco.LogWarn(errco.NewErr(errco.ERROR_CONFIG_CHECK, errco.LVL_1, "loadRuntime", "could not execute 'java -version' command"))
		Javav = "unknown"
	} else {
		Javav = strings.ReplaceAll(strings.Split(string(out), "\n")[0], "\r", "")
	}

	// initialize ip and ports for connection
	errMsh := c.loadIpPorts()
	if errMsh != nil {
		servstats.Stats.SetMajorError(errco.NewErr(errco.ERROR_MINECRAFT_SERVER, errco.LVL_3, "loadRuntime", "proxy setup failed, check msh logs"))
		errco.LogMshErr(errMsh.AddTrace("loadRuntime"))
	}
	errco.Logln(errco.LVL_3, "msh proxy setup: %s:%d --> %s:%d", ListenHost, ListenPort, TargetHost, TargetPort)

	// load server icon
	errMsh = c.loadIcon()
	if errMsh != nil {
		// it's enough to log a warning since the default icon is loaded by default
		errco.LogWarn(errMsh.AddTrace("loadRuntime"))
	}

	return nil
}
