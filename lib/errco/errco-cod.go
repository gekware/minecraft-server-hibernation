package errco

/*
0xxxxxfxxx: error

0x0000xxxx: server control package
0x0001xxxx: program manager package
0x0002xxxx: server connection package
0x0003xxxx: config package
0x0004xxxx: operative system package
0x0005xxxx: utility package
0x0006xxxx: main
0x0007xxxx: input package
0x0008xxxx: errco package
0x0009xxxx: servstats package
*/

// ------------------- codes ------------------- //

const (
	// server control package

	SERVER_STATUS_OFFLINE  = 0x00000000
	SERVER_STATUS_STARTING = 0x00000001
	SERVER_STATUS_ONLINE   = 0x00000002
	SERVER_STATUS_STOPPING = 0x00000003

	// program manager package

	VERSION_DEP = 0x00010000 // check update result: msh is running deprecated version
	VERSION_UPD = 0x00010001 // check update result: update available
	VERSION_OK  = 0x00010002 // check update result: msh updated
	VERSION_DEV = 0x00010003 // check update result: msh is running dev version
	VERSION_UNO = 0x00010004 // check update result: msh is running unofficial version

	// server connection package

	CLIENT_REQ_UNKN     = 0x00020000 // client request unknown
	CLIENT_REQ_INFO     = 0x00020001 // client request server info
	CLIENT_REQ_JOIN     = 0x00020002 // client request server join
	MESSAGE_FORMAT_TXT  = 0x00020003 // message to client should be built as TXT
	MESSAGE_FORMAT_INFO = 0x00020004 // message to client should be built as INFO
)

// -------------------- log -------------------- //

const (
	// log levels

	LVL_0 LogLvl = 0 // NONE: no log
	LVL_1 LogLvl = 1 // BASE: basic log
	LVL_2 LogLvl = 2 // SERV: mincraft server log
	LVL_3 LogLvl = 3 // DEVE: developement log
	LVL_4 LogLvl = 4 // BYTE: connection bytes log

	// log types

	TYPE_INF LogTyp = "info"
	TYPE_SER LogTyp = "serv"
	TYPE_BYT LogTyp = "byte"
	TYPE_WAR LogTyp = "warn"
	TYPE_ERR LogTyp = "error"
)

// ------------------- errors ------------------ //

const (
	ERROR_NIL LogCod = 0xffffffff // no error

	// server control package

	ERROR_TERMINAL_NOT_ACTIVE      LogCod = 0x0000f000 // server terminal is not active
	ERROR_TERMINAL_ALREADY_ACTIVE  LogCod = 0x0000f001 // server terminal is already active
	ERROR_TERMINAL_START           LogCod = 0x0000f002 // error while starting server terminal
	ERROR_SERVER_NOT_ONLINE        LogCod = 0x0000f100 // minecraft server is not online
	ERROR_SERVER_NOT_EMPTY         LogCod = 0x0000f101 // minecraft server is not empty
	ERROR_SERVER_MUST_WAIT         LogCod = 0x0000f102 // msh issued ms stop ahead of specified wait time
	ERROR_SERVER_UNEXP_OUTPUT      LogCod = 0x0000f103 // minecraft server output does not adhere to expected log format
	ERROR_SERVER_KILL              LogCod = 0x0000f104 // error while killing minecraft server process
	ERROR_SERVER_IS_WARM           LogCod = 0x0000f106 // minecraft server is already warm
	ERROR_SERVER_IS_FROZEN         LogCod = 0x0000f107 // minecraft server is already frozen
	ERROR_SERVER_OFFLINE_SUSPENDED LogCod = 0x0000f108 // minecraft server is offline but not suspended
	ERROR_SERVER_SUSPENDED         LogCod = 0x0000f109 // minecraft server is suspended
	ERROR_PIPE_INPUT_WRITE         LogCod = 0x0000f200 // error while writing to terminal input
	ERROR_PIPE_LOAD                LogCod = 0x0000f201 // error while loading pipe
	ERROR_CONVERSION               LogCod = 0x0000f300 // error while converting variable

	// program manager package

	ERROR_VERSION         LogCod = 0x0001f000 // check update error
	ERROR_VERSION_INVALID LogCod = 0x0001f001 // version format is invalid
	ERROR_GET_CORES       LogCod = 0x0001f100 // error getting system cores count
	ERROR_GET_CPU_INFO    LogCod = 0x0001f101 // error getting cpu info
	ERROR_GET_MEMORY      LogCod = 0x0001f102 // error getting system memory info

	// server connection package

	ERROR_REQ_FLAG_BUILD      LogCod = 0x0002f000 // error while building request flag
	ERROR_CLIENT_REQ          LogCod = 0x0002f100 // client request error
	ERROR_CLIENT_SOCKET_READ  LogCod = 0x0002f101 // error while reading client socket
	ERROR_SERVER_DIAL         LogCod = 0x0002f200 // error while dialing ms server
	ERROR_SERVER_REQUEST_INFO LogCod = 0x0002f201 // error while msh server info request
	ERROR_JSON_MARSHAL        LogCod = 0x0002f300 // error while exporting struct to json bytes
	ERROR_JSON_UNMARSHAL      LogCod = 0x0002f301 // error while importing struct from json bytes

	// config package

	ERROR_CONFIG_LOAD             LogCod = 0x0003f000 // error while loading config
	ERROR_CONFIG_SAVE             LogCod = 0x0003f001 // error while saving config to file
	ERROR_CONFIG_CHECK            LogCod = 0x0003f002 // error while checking config
	ERROR_CONFIG_MSHID            LogCod = 0x0003f003 // error while managing msh id
	ERROR_ICON_LOAD               LogCod = 0x0003f100 // error while loading icon
	ERROR_VERSION_LOAD            LogCod = 0x0003f101 // error while loading version.json from server JAR
	ERROR_PLAYER_NOT_IN_WHITELIST LogCod = 0x0003f200 // player is not in whitelist

	// operative system package

	ERROR_OS_NOT_SUPPORTED        LogCod = 0x0004f000 // error OS not supported
	ERROR_PROCESS_OPEN            LogCod = 0x0004f100 // error while opening process
	ERROR_PROCESS_SIGNAL          LogCod = 0x0004f101 // error while sending signal to process
	ERROR_PROCESS_SUSPEND_CALL    LogCod = 0x0004f200 // error while executing suspend call to process handle
	ERROR_PROCESS_RESUME_CALL     LogCod = 0x0004f201 // error while executing resume call to process handle
	ERROR_PROCESS_SYSTEM_SNAPSHOT LogCod = 0x0004f300 // error while building system processes snapshot
	ERROR_PROCESS_ENTRY           LogCod = 0x0004f301 // error while setting first process entry in snapshot
	ERROR_PROCESS_NOT_FOUND       LogCod = 0x0004f400 // error process pid was not found

	// utility package

	ERROR_ANALYSIS LogCod = 0x0005f000 // error while analyzing data

	// main

	ERROR_CLIENT_LISTEN LogCod = 0x0006f000 // error while listening for new clients
	ERROR_CLIENT_ACCEPT LogCod = 0x0006f001 // error while accepting new client

	// input package

	ERROR_COMMAND_INPUT     LogCod = 0x0007f000 // general error while reading command input
	ERROR_COMMAND_UNKNOWN   LogCod = 0x0007f001 // command is unknown
	ERROR_INPUT_READ        LogCod = 0x0007f100 // error while reading input)
	ERROR_INPUT_UNAVAILABLE LogCod = 0x0007f101 // stdin is not available

	// errco package
	ERROR_COLOR_ENABLE LogCod = 0x0008f000 // error while trying to enable colors on terminal

	// servstats package
	ERROR_MINECRAFT_SERVER LogCod = 0x0009f000 // major error while starting minecraft server (will be communicated to clients trying to join)
)
