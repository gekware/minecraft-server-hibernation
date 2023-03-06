package errco

/*
0xxxfxxx: error

0x00xxxx: server control package
0x01xxxx: program manager package
0x02xxxx: server connection package
0x03xxxx: config package
0x04xxxx: operative system package
0x05xxxx: utility package
0x06xxxx: main
0x07xxxx: input package
0x08xxxx: errco package
0x09xxxx: servstats package
*/

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

// ------------------- codes ------------------- //

const (
	// server control package

	SERVER_STATUS_OFFLINE  = 0x000000
	SERVER_STATUS_STARTING = 0x000001
	SERVER_STATUS_ONLINE   = 0x000002
	SERVER_STATUS_STOPPING = 0x000003

	// program manager package

	VERSION_DEP = 0x010000 // check update result: msh is running deprecated version
	VERSION_UPD = 0x010001 // check update result: update available
	VERSION_OK  = 0x010002 // check update result: msh updated
	VERSION_DEV = 0x010003 // check update result: msh is running dev version
	VERSION_UNO = 0x010004 // check update result: msh is running unofficial version

	// server connection package

	CLIENT_REQ_UNKN     = 0x020000 // client request unknown
	CLIENT_REQ_INFO     = 0x020001 // client request server info
	CLIENT_REQ_JOIN     = 0x020002 // client request server join
	MESSAGE_FORMAT_TXT  = 0x020103 // message to client should be built as TXT
	MESSAGE_FORMAT_INFO = 0x020104 // message to client should be built as INFO
)

// ------------------- errors ------------------ //

const (
	// don't use 0xffffffff as it will overflow on arch 386
	ERROR_NIL LogCod = 0xffffff // no error

	// server control package

	ERROR_TERMINAL_NOT_ACTIVE      LogCod = 0x00f000 // server terminal is not active
	ERROR_TERMINAL_ACTIVE          LogCod = 0x00f001 // server terminal is active
	ERROR_TERMINAL_START           LogCod = 0x00f002 // server terminal error while starting
	ERROR_MSH_MUST_WAIT            LogCod = 0x00f100 // timeout time not reached to issue ms stop
	ERROR_SERVER_STATUS_UNKNOWN    LogCod = 0x00f200 // minecraft server status unknown
	ERROR_SERVER_NOT_ONLINE        LogCod = 0x00f201 // minecraft server is not online
	ERROR_SERVER_NOT_EMPTY         LogCod = 0x00f202 // minecraft server is not empty
	ERROR_SERVER_UNEXP_OUTPUT      LogCod = 0x00f203 // minecraft server output does not adhere to expected log format
	ERROR_SERVER_KILL              LogCod = 0x00f204 // minecraft server process kill error
	ERROR_SERVER_IS_WARM           LogCod = 0x00f205 // minecraft server is already warm
	ERROR_SERVER_IS_FROZEN         LogCod = 0x00f206 // minecraft server is already frozen
	ERROR_SERVER_SUSPENDED         LogCod = 0x00f207 // minecraft server is suspended
	ERROR_SERVER_NOT_SUSPENDED     LogCod = 0x00f208 // minecraft server is not suspended
	ERROR_SERVER_OFFLINE           LogCod = 0x00f209 // minecraft server is offline
	ERROR_SERVER_OFFLINE_SUSPENDED LogCod = 0x00f20a // minecraft server is offline but not suspended
	ERROR_SERVER_STOPPING          LogCod = 0x00f20b // minecraft server is stopping
	ERROR_SERVER_UNRESPONDING      LogCod = 0x00f20c // minecraft server is not responding
	ERROR_PIPE_INPUT_WRITE         LogCod = 0x00f300 // terminal input writing error
	ERROR_PIPE_LOAD                LogCod = 0x00f301 // terminal pipe load error
	ERROR_CONVERSION               LogCod = 0x00f400 // variable conversion error
	ERROR_WRONG_CONNECTION_COUNT   LogCod = 0x00f500 // connection count does not correspond to ms player count

	// program manager package

	ERROR_VERSION         LogCod = 0x01f000 // check update error
	ERROR_VERSION_INVALID LogCod = 0x01f001 // version format is invalid
	ERROR_GET_CORES       LogCod = 0x01f100 // error getting system cores count
	ERROR_GET_CPU_INFO    LogCod = 0x01f101 // error getting cpu info
	ERROR_GET_MEMORY      LogCod = 0x01f102 // error getting system memory info
	ERROR_BODY_READ       LogCod = 0x01f200 // error reading a body response

	// server connection package

	ERROR_REQ_FLAG_BUILD      LogCod = 0x02f000 // error while building request flag
	ERROR_CLIENT_REQ          LogCod = 0x02f100 // client request error
	ERROR_CLIENT_SOCKET_READ  LogCod = 0x02f101 // error while reading client socket
	ERROR_CONN_READ           LogCod = 0x02f102 // error while reading from client connection
	ERROR_CONN_WRITE          LogCod = 0x02f103 // error while writing to client connection
	ERROR_CONN_EOF            LogCod = 0x02f104 // read EOF from client connection
	ERROR_SERVER_DIAL         LogCod = 0x02f200 // error while dialing ms server
	ERROR_SERVER_REQUEST_INFO LogCod = 0x02f201 // error while msh server info request
	ERROR_JSON_MARSHAL        LogCod = 0x02f300 // error while exporting struct to json bytes
	ERROR_JSON_UNMARSHAL      LogCod = 0x02f301 // error while importing struct from json bytes
	ERROR_QUERY_CHALLENGE     LogCod = 0x02f401 // error caused by query challenge
	ERROR_QUERY_BAD_REQUEST   LogCod = 0x02f402 // error caused by query request
	ERROR_PING_PACKET_UNKNOWN LogCod = 0x02f500 // error ping packet received is unknown

	// config package

	ERROR_CONFIG_LOAD      LogCod = 0x03f000 // error while loading config
	ERROR_CONFIG_SAVE      LogCod = 0x03f001 // error while saving config to file
	ERROR_CONFIG_CHECK     LogCod = 0x03f002 // error while checking config
	ERROR_CONFIG_MSHID     LogCod = 0x03f003 // error while managing msh id
	ERROR_ICON_LOAD        LogCod = 0x03f100 // error while loading icon
	ERROR_VERSION_LOAD     LogCod = 0x03f101 // error while loading version.json from server JAR
	ERROR_WHITELIST_CHECK  LogCod = 0x03f200 // error while checking whitelist
	ERROR_TYPE_UNSUPPORTED LogCod = 0x03f300 // error interface{}.(type) not supported
	ERROR_INVALID_COMMAND  LogCod = 0x03f400 // error start ms command is invalid
	ERROR_PARSE            LogCod = 0x03f500 // error while parsing args

	// operative system package

	ERROR_OS_NOT_SUPPORTED        LogCod = 0x04f000 // error OS not supported
	ERROR_PROCESS_OPEN            LogCod = 0x04f100 // error while opening process
	ERROR_PROCESS_SIGNAL          LogCod = 0x04f101 // error while sending signal to process
	ERROR_PROCESS_SUSPEND_CALL    LogCod = 0x04f200 // error while executing suspend call to process handle
	ERROR_PROCESS_RESUME_CALL     LogCod = 0x04f201 // error while executing resume call to process handle
	ERROR_PROCESS_SYSTEM_SNAPSHOT LogCod = 0x04f300 // error while building system processes snapshot
	ERROR_PROCESS_ENTRY           LogCod = 0x04f301 // error while setting first process entry in snapshot
	ERROR_PROCESS_NOT_FOUND       LogCod = 0x04f400 // error process pid was not found
	ERROR_PROCESS_LIST            LogCod = 0x04f401 // error processes running not found
	ERROR_PROCESS_KILL            LogCod = 0x04f402 // error process kill
	ERROR_PROCESS_TIME            LogCod = 0x04f500 // error while retrieving process time

	// utility package

	ERROR_ANALYSIS LogCod = 0x05f000 // error while analyzing data

	// main

	ERROR_CLIENT_LISTEN LogCod = 0x06f000 // error while listening for new clients
	ERROR_CLIENT_ACCEPT LogCod = 0x06f001 // error while accepting new client

	// input package

	ERROR_COMMAND_INPUT   LogCod = 0x07f000 // general error while reading command input
	ERROR_COMMAND_UNKNOWN LogCod = 0x07f001 // command is unknown
	ERROR_INPUT           LogCod = 0x07f100 // error input
	ERROR_INPUT_READ      LogCod = 0x07f101 // error while reading input
	ERROR_INPUT_EOF       LogCod = 0x07f102 // read EOF from stdin

	// errco package
	ERROR_COLOR_ENABLE LogCod = 0x08f000 // error while trying to enable colors on terminal

	// servstats package
	ERROR_MINECRAFT_SERVER LogCod = 0x09f000 // major error while starting minecraft server (will be communicated to clients trying to join)
)
