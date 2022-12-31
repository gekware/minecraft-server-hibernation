package config

import (
	"encoding/json"
	"flag"
	"fmt"
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

	JavaV string // Javav is the java version on the system. format: "java 16.0.1 2021-04-20"

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
func LoadConfig() *errco.MshLog {
	// ---------------- OS support ----------------- //

	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "checking OS support...")

	// check if OS is supported.
	logMsh := opsys.OsSupported()
	if logMsh != nil {
		return logMsh.AddTrace()
	}

	// ---------------- load config ---------------- //

	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "loading config...")

	// load config default
	logMsh = ConfigDefault.loadDefault()
	if logMsh != nil {
		return logMsh.AddTrace()
	}

	// load config runtime
	logMsh = ConfigRuntime.loadRuntime(ConfigDefault)
	if logMsh != nil {
		return logMsh.AddTrace()
	}

	// ---------------- save config ---------------- //

	if configDefaultSave {
		logMsh := ConfigDefault.Save()
		if logMsh != nil {
			return logMsh.AddTrace()
		}
	}

	return nil
}

// Save saves config to the config file.
// Then does the default config setup
func (c *Configuration) Save() *errco.MshLog {
	// encode the struct config
	configData, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONFIG_SAVE, "could not marshal from config file")
	}

	// escape unicode characters ("\u003c" to "<" and "\u003e" to ">")
	configData, logMsh := utility.UnicodeEscape(configData)
	if logMsh != nil {
		logMsh.Log(true)
	}

	// write to config file
	err = os.WriteFile(configFileName, configData, 0644)
	if err != nil {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONFIG_SAVE, "could not write to config file")
	}

	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "saved default config to config file")

	return nil
}

// loadDefault loads config file to config variable
func (c *Configuration) loadDefault() *errco.MshLog {
	// get working directory
	cwdPath, err := os.Getwd()
	if err != nil {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_1, errco.ERROR_CONFIG_LOAD, err.Error())
	}

	// read config file
	configFilePath := filepath.Join(cwdPath, configFileName)
	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "reading config file: \"%s\"", configFilePath)
	configData, err := os.ReadFile(configFilePath)
	if err != nil {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_1, errco.ERROR_CONFIG_LOAD, err.Error())
	}

	// write data to config variable
	err = json.Unmarshal(configData, &c)
	if err != nil {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_1, errco.ERROR_CONFIG_LOAD, err.Error())
	}

	// ------------------- setup ------------------- //

	// load mshid
	mi := MshID()
	if c.Configuration.Msh.ID != mi {
		errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_CONFIG_LOAD, "config msh id different from instance msh id, applying correction...")
		c.Configuration.Msh.ID = mi
		configDefaultSave = true
	}

	// load ms version/protocol
	// (checkout version.json info: https://minecraft.fandom.com/wiki/Version.json)
	version, protocol, logMsh := c.getVersionInfo()
	if logMsh != nil {
		// just log it since ms version/protocol are not vital for the connection with clients
		logMsh.Log(true)
	} else if c.Server.Version != version || c.Server.Protocol != protocol {
		c.Server.Version = version
		c.Server.Protocol = protocol
		configDefaultSave = true
	}

	return nil
}

// loadRuntime initializes runtime config to default config.
// Then parses start arguments into runtime config, replaces placeholders and does the runtime config setup
func (c *Configuration) loadRuntime(confdef *Configuration) *errco.MshLog {
	// initialize config to base
	*c = *confdef

	// specify arguments
	flag.StringVar(&c.Server.Folder, "folder", c.Server.Folder, "Specify minecraft server folder path.")
	flag.StringVar(&c.Server.FileName, "file", c.Server.FileName, "Specify minecraft server file name.")
	flag.StringVar(&c.Server.Version, "version", c.Server.Version, "Specify minecraft server version.")
	flag.IntVar(&c.Server.Protocol, "protocol", c.Server.Protocol, "Specify minecraft server protocol.")

	// c.Commands.StartServer should not be set by a flag
	flag.StringVar(&c.Commands.StartServerParam, "msparam", c.Commands.StartServerParam, "Specify start server parameters.")
	// c.Commands.StopServer should not be set by a flag
	flag.IntVar(&c.Commands.StopServerAllowKill, "allowkill", c.Commands.StopServerAllowKill, "Specify after how many seconds the server should be killed (if stop command fails).")

	flag.IntVar(&c.Msh.Debug, "d", c.Msh.Debug, "Specify debug level.")
	// c.Msh.ID should not be set by a flag
	flag.IntVar(&c.Msh.ListenPort, "port", c.Msh.ListenPort, "Specify msh port.")
	flag.Int64Var(&c.Msh.TimeBeforeStoppingEmptyServer, "timeout", c.Msh.TimeBeforeStoppingEmptyServer, "Specify time to wait before stopping minecraft server.")
	flag.BoolVar(&c.Msh.SuspendAllow, "SuspendAllow", c.Msh.SuspendAllow, "Specify if minecraft server process can be suspended.")
	flag.IntVar(&c.Msh.SuspendRefresh, "SuspendRefresh", c.Msh.SuspendRefresh, "Specify how often the suspended minecraft server process must be refreshed.")
	flag.StringVar(&c.Msh.InfoHibernation, "infohibe", c.Msh.InfoHibernation, "Specify hibernation info.")
	flag.StringVar(&c.Msh.InfoStarting, "infostar", c.Msh.InfoStarting, "Specify starting info.")
	flag.BoolVar(&c.Msh.NotifyUpdate, "notifyupd", c.Msh.NotifyUpdate, "Specify if update notifications are allowed.")
	flag.BoolVar(&c.Msh.NotifyMessage, "notifymes", c.Msh.NotifyMessage, "Specify if message notifications are allowed.")
	// c.Msh.Whitelist (type []string, not worth to make it a flag)
	flag.BoolVar(&c.Msh.WhitelistImport, "wlimport", c.Msh.WhitelistImport, "Specify is minecraft server whitelist should be imported")

	// specify the usage when there is an error in the arguments
	flag.Usage = func() {
		// not using errco.NewLogln since log time is not needed
		fmt.Println("Usage of msh:")
		flag.PrintDefaults()
	}

	// parse arguments
	flag.Parse()

	// replace placeholders
	c.Commands.StartServer = strings.ReplaceAll(c.Commands.StartServer, "<Server.FileName>", c.Server.FileName)
	c.Commands.StartServer = strings.ReplaceAll(c.Commands.StartServer, "<Commands.StartServerParam>", c.Commands.StartServerParam)

	// after config variables are set, set debug level
	errco.NewLogln(errco.TYPE_INF, errco.LVL_0, errco.ERROR_NIL, "setting log level to: %d", c.Msh.Debug)
	errco.DebugLvl = errco.LogLvl(c.Msh.Debug)

	// ------------------- setup ------------------- //

	// check if server folder/executeble exist
	serverFileFolderPath := filepath.Join(c.Server.Folder, c.Server.FileName)
	if _, err := os.Stat(serverFileFolderPath); os.IsNotExist(err) {
		// server folder/executeble does not exist

		logMsh := errco.NewLogln(errco.TYPE_ERR, errco.LVL_1, errco.ERROR_MINECRAFT_SERVER, "specified minecraft server folder/file does not exist: %s", serverFileFolderPath)
		servstats.Stats.SetMajorError(logMsh)

	} else {
		// server folder/executeble exist

		// check if eula.txt exists and is set to true
		eulaFilePath := filepath.Join(c.Server.Folder, "eula.txt")
		eulaData, err := os.ReadFile(eulaFilePath)
		switch {
		case err != nil:
			// eula.txt does not exist

			errco.NewLogln(errco.TYPE_WAR, errco.LVL_1, errco.ERROR_CONFIG_CHECK, "could not read eula.txt file: %s", eulaFilePath)

			// start server to generate eula.txt (and server.properties)
			errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "starting minecraft server to generate eula.txt file...")
			cSplit := strings.Split(c.Commands.StartServer, " ")
			cmd := exec.Command(cSplit[0], cSplit[1:]...)
			cmd.Dir = c.Server.Folder
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			fmt.Print(errco.COLOR_CYAN) // set color to server log color
			err = cmd.Run()
			fmt.Print(errco.COLOR_RESET) // reset color
			if err != nil {
				logMsh := errco.NewLogln(errco.TYPE_ERR, errco.LVL_1, errco.ERROR_MINECRAFT_SERVER, "couldn't start minecraft server to generate eula.txt (%s)", err.Error())
				servstats.Stats.SetMajorError(logMsh)
			}
			fallthrough

		case !strings.Contains(strings.ReplaceAll(strings.ToLower(string(eulaData)), " ", ""), "eula=true"):
			// eula.txt exists but is not set to true

			logMsh := errco.NewLogln(errco.TYPE_ERR, errco.LVL_1, errco.ERROR_MINECRAFT_SERVER, "please accept minecraft server eula.txt: %s", eulaFilePath)
			servstats.Stats.SetMajorError(logMsh)

		default:
			// eula.txt exists and is set to true

			errco.NewLogln(errco.TYPE_INF, errco.LVL_1, errco.ERROR_NIL, "eula.txt exist and is set to true")
		}
	}

	// check if java is installed and get java version
	_, err := exec.LookPath("java")
	if err != nil {
		logMsh := errco.NewLogln(errco.TYPE_ERR, errco.LVL_1, errco.ERROR_MINECRAFT_SERVER, "java not installed")
		servstats.Stats.SetMajorError(logMsh)
	} else if out, err := exec.Command("java", "--version").Output(); err != nil {
		// non blocking error
		errco.NewLogln(errco.TYPE_WAR, errco.LVL_1, errco.ERROR_CONFIG_CHECK, "could not execute 'java -version' command")
		JavaV = "unknown"
	} else {
		JavaV = strings.ReplaceAll(strings.Split(string(out), "\n")[0], "\r", "")
	}

	// initialize ip and ports for connection
	logMsh := c.loadIpPorts()
	if logMsh != nil {
		logMsh.Log(true)
		servstats.Stats.SetMajorError(logMsh)
	} else {
		errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "msh proxy setup: %s:%d --> %s:%d", ListenHost, ListenPort, TargetHost, TargetPort)
	}

	// load server icon
	logMsh = c.loadIcon()
	if logMsh != nil {
		// log and continue (default icon is loaded by default)
		logMsh.Log(true)
	}

	return nil
}
