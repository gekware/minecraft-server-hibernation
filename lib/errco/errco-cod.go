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

// ------------------- errors ------------------ //

const (
	// server control package

	ERROR_TERMINAL_NOT_ACTIVE = 0x0000f000 // server terminal is not active
	ERROR_TERMINAL_START      = 0x0000f001 // error while starting server terminal
	ERROR_SERVER_NOT_ONLINE   = 0x0000f100 // server is not online
	ERROR_SERVER_NOT_EMPTY    = 0x0000f101 // minecraft server is not empty
	ERROR_SERVER_MUST_WAIT    = 0x0000f102 // msh issued ms stop ahead of specified wait time
	ERROR_SERVER_UNEXP_OUTPUT = 0x0000f103 // server output does not adhere to expected log format
	ERROR_SERVER_KILL         = 0x0000f104 // error while killing server process
	ERROR_PIPE_INPUT_WRITE    = 0x0000f200 // error while writing to terminal input
	ERROR_PIPE_LOAD           = 0x0000f201 // error while loading pipe
	ERROR_CONVERSION          = 0x0000f300 // error while converting variable

	// program manager package

	ERROR_VERSION         = 0x0001f000 // check update error
	ERROR_VERSION_INVALID = 0x0001f001 // version format is invalid
	ERROR_GET_CORES       = 0x0001f100 // error getting system cores count
	ERROR_GET_CPU_INFO    = 0x0001f101 // error getting cpu info
	ERROR_GET_MEMORY      = 0x0001f102 // error getting system memory info

	// server connection package

	ERROR_REQ_FLAG_BUILD      = 0x0002f000 // error while building request flag
	ERROR_CLIENT_REQ          = 0x0002f100 // client request error
	ERROR_CLIENT_SOCKET_READ  = 0x0002f101 // error while reading client socket
	ERROR_SERVER_DIAL         = 0x0002f200 // error while dialing ms server
	ERROR_SERVER_REQUEST_INFO = 0x0002f201 // error while msh server info request
	ERROR_JSON_MARSHAL        = 0x0002f300 // error while exporting struct to json bytes
	ERROR_JSON_UNMARSHAL      = 0x0002f301 // error while importing struct from json bytes

	// config package

	ERROR_CONFIG_LOAD             = 0x0003f000 // error while loading config
	ERROR_CONFIG_SAVE             = 0x0003f001 // error while saving config to file
	ERROR_CONFIG_CHECK            = 0x0003f002 // error while checking config
	ERROR_ICON_LOAD               = 0x0003f100 // error while loading icon
	ERROR_VERSION_LOAD            = 0x0003f101 // error while loading version.json from server JAR
	ERROR_PLAYER_NOT_IN_WHITELIST = 0x0003f200 // player is not in whitelist

	// operative system package

	ERROR_OS_NOT_SUPPORTED = 0x0004f000 // OS not supported

	// utility package

	ERROR_ANALYSIS = 0x0005f000 // error while analyzing data

	// main

	ERROR_CLIENT_LISTEN = 0x0006f000 // error while listening for new clients
	ERROR_CLIENT_ACCEPT = 0x0006f001 // error while accepting new client

	// input package

	ERROR_COMMAND_INPUT     = 0x0007f000 // general error while reading command input
	ERROR_COMMAND_UNKNOWN   = 0x0007f001 // command is unknown
	ERROR_INPUT_READ        = 0x0007f100 // error while reading input)
	ERROR_INPUT_UNAVAILABLE = 0x0007f101 // stdin is not available
)
