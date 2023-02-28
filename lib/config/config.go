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

	"github.com/google/shlex"
)

var (
	configFileName string = "msh-config.json" // configFileName is the config file name

	ConfigDefault *Configuration = &Configuration{} // ConfigDefault contains parameters of config in file
	ConfigRuntime *Configuration = &Configuration{} // ConfigRuntime contains parameters of config in runtime

	configDefaultSave bool = false // if true, the config will be saved after successful loading

	JavaV string // Javav is the java version on the system. format: "java 16.0.1 2021-04-20"

	ServerIcon string = defaultServerIcon // ServerIcon contains the minecraft server icon

	MshHost       string = "0.0.0.0"   // MshHost		is the ip address for clients to connect to msh
	MshPort       int                  // MshPort		is the port for clients to connect to msh
	MshPortQuery  int                  // MshPortQuery	is the port for clients to perform stats query requests at msh
	ServHost      string = "127.0.0.1" // ServHost		is the ip address for msh to connect to minecraft server
	ServPort      int                  // ServPort		is the port for msh to connect to minecraft server
	ServPortQuery int                  // ServPortQuery	is the port for msh to perform stats query requests at minecraft server
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

		// reset config default save flag
		configDefaultSave = false
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

	return nil
}

// loadRuntime initializes runtime config to default config.
// Then parses start arguments into runtime config, replaces placeholders and does the runtime config setup
func (c *Configuration) loadRuntime(confdef *Configuration) *errco.MshLog {
	var logMsh *errco.MshLog

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
	flag.IntVar(&c.Msh.MshPort, "port", c.Msh.MshPort, "Specify msh port.")
	flag.Int64Var(&c.Msh.TimeBeforeStoppingEmptyServer, "timeout", c.Msh.TimeBeforeStoppingEmptyServer, "Specify time to wait before stopping minecraft server.")
	flag.BoolVar(&c.Msh.SuspendAllow, "suspendallow", c.Msh.SuspendAllow, "Specify if minecraft server process can be suspended.")
	flag.IntVar(&c.Msh.SuspendRefresh, "suspendrefresh", c.Msh.SuspendRefresh, "Specify how often the suspended minecraft server process must be refreshed.")
	flag.StringVar(&c.Msh.InfoHibernation, "infohibe", c.Msh.InfoHibernation, "Specify hibernation info.")
	flag.StringVar(&c.Msh.InfoStarting, "infostar", c.Msh.InfoStarting, "Specify starting info.")
	flag.BoolVar(&c.Msh.NotifyUpdate, "notifyupd", c.Msh.NotifyUpdate, "Specify if update notifications are allowed.")
	flag.BoolVar(&c.Msh.NotifyMessage, "notifymes", c.Msh.NotifyMessage, "Specify if message notifications are allowed.")
	// c.Msh.Whitelist (type []string, not worth to make it a flag)
	flag.BoolVar(&c.Msh.WhitelistImport, "wlimport", c.Msh.WhitelistImport, "Specify is minecraft server whitelist should be imported")

	// backward compatibility
	flag.IntVar(&c.Commands.StopServerAllowKill, "allowKill", c.Commands.StopServerAllowKill, "Specify after how many seconds the server should be killed (if stop command fails).") // msh pterodactyl egg
	flag.BoolVar(&c.Msh.SuspendAllow, "SuspendAllow", c.Msh.SuspendAllow, "Specify if minecraft server process can be suspended.")                                                   // msh pterodactyl egg
	flag.IntVar(&c.Msh.SuspendRefresh, "SuspendRefresh", c.Msh.SuspendRefresh, "Specify how often the suspended minecraft server process must be refreshed.")                        // msh pterodactyl egg

	// specify the usage when there is an error in the arguments
	flag.Usage = func() {
		// not using errco.NewLogln since log time is not needed
		fmt.Println("Usage of msh:")
		flag.PrintDefaults()
	}

	// join os provided args and split them again with shlex.
	// (this prevents badly splitted arguments on pterodactyl panel)
	// fixes #188
	args, err := shlex.Split(strings.Join(os.Args[1:], " "))
	if err != nil {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_1, errco.ERROR_PARSE, err.Error())
	}
	flag.CommandLine.Parse(args)

	// after config variables are set, set debug level
	errco.NewLogln(errco.TYPE_INF, errco.LVL_0, errco.ERROR_NIL, "setting log level to: %d", c.Msh.Debug)
	errco.DebugLvl = errco.LogLvl(c.Msh.Debug)

	// ---------------- setup check ---------------- //

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
			command, logMsh := c.BuildCommandStartServer()
			if logMsh != nil {
				return logMsh.AddTrace()
			}
			cmd := exec.Command(command[0], command[1:]...)
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
	_, err = exec.LookPath("java")
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

	// ---------------- setup load ----------------- //

	// load ports
	// MshHost	defined in global definition
	MshPort = c.Msh.MshPort
	MshPortQuery = c.Msh.MshPortQuery
	// ServHost	defined in global definition
	if ServPort, logMsh = c.ParsePropertiesInt("server-port"); logMsh != nil {
		logMsh.Log(true)
	} else if ServPort == c.Msh.MshPort {
		logMsh := errco.NewLogln(errco.TYPE_ERR, errco.LVL_1, errco.ERROR_CONFIG_LOAD, "ServPort and MshPort appear to be the same, please change one of them")
		servstats.Stats.SetMajorError(logMsh)
	}
	if ServPortQuery, logMsh = c.ParsePropertiesInt("query.port"); logMsh != nil {
		logMsh.Log(true)
	} else if ServPortQuery == c.Msh.MshPortQuery {
		logMsh := errco.NewLogln(errco.TYPE_ERR, errco.LVL_1, errco.ERROR_CONFIG_LOAD, "ServPortQuery and MshPortQuery appear to be the same, please change one of them")
		servstats.Stats.SetMajorError(logMsh)
	}

	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "msh connection  proxy setup: %s:%d --> %s:%d", MshHost, MshPort, ServHost, ServPort)
	errco.NewLogln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "msh stats query proxy setup: %s:%d --> %s:%d", MshHost, MshPortQuery, ServHost, ServPortQuery)

	// load ms version/protocol
	c.Server.Version, c.Server.Protocol, logMsh = c.getVersionInfo()
	if logMsh != nil {
		// just log it since ms version/protocol are not vital for the connection with clients
		logMsh.Log(true)
	} else if c.Server.Version == "" || c.Server.Protocol == -1 {
		// found ms version/protocol are invalid
		errco.NewLogln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_VERSION_LOAD, "version (%s) and protocol (%d) are invalid", c.Server.Version, c.Server.Protocol)
	} else if confdef.Server.Version != c.Server.Version || confdef.Server.Protocol != c.Server.Protocol {
		// replace found ms version/protocol in default config,
		confdef.Server.Version = c.Server.Version
		confdef.Server.Protocol = c.Server.Protocol
		configDefaultSave = true
	}

	// load server icon
	logMsh = c.loadIcon()
	if logMsh != nil {
		// log and continue (default icon is loaded by default)
		logMsh.Log(true)
	}

	return nil
}

// BuildCommandStartServer builds the start server command by replacing placeholders.
//
// If generated command has less than 2 arguments, it is considered invalid and error returned.
func (c *Configuration) BuildCommandStartServer() ([]string, *errco.MshLog) {
	var command = []string{}
	for _, ss := range strings.Fields(c.Commands.StartServer) {
		switch ss {
		case "<Server.FileName>":
			command = append(command, c.Server.FileName)
		case "<Commands.StartServerParam>":
			command = append(command, strings.Fields(c.Commands.StartServerParam)...)
		default:
			command = append(command, ss)
		}
	}

	if len(command) < 2 {
		return command, errco.NewLog(errco.TYPE_ERR, errco.LVL_1, errco.ERROR_INVALID_COMMAND, "generated command to start minecraft server is invalid")
	}

	return command, nil
}
