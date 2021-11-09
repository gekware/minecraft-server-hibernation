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

	VERSION_UPDATED           = 0x00010000 // check update result: msh updated
	VERSION_UPDATEAVAILABLE   = 0x00010001 // check update result: update available
	VERSION_UNOFFICIALVERSION = 0x00010002 // check update result: msh is running unofficial version

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

	TERMINAL_NOT_ACTIVE_ERROR = 0x0000f000 // server terminal is not active
	SERVER_NOT_ONLINE_ERROR   = 0x0000f001 // server terminal is not online
	INPUT_PIPE_WRITE_ERROR    = 0x0000f002 // error while writing to terminal input
	TERMINAL_START_ERROR      = 0x0000f003 // error while starting server terminal
	PIPE_LOAD_ERROR           = 0x0000f004 // error while loading pipe
	SERVER_NOT_EMPTY_ERROR    = 0x0000f005 // minecraft server is not empty
	SERVER_MUST_WAIT_ERROR    = 0x0000f006 // msh issued ms stop ahead of specified wait time
	CONVERSION_ERROR          = 0x0000f007 // error while converting variable
	SERVER_UNEXP_OUTPUT_ERROR = 0x0000f008 // server output does not adhere to expected log format
	SERVER_KILL_ERROR         = 0x0000f009 // error while killing server process

	// program manager package

	VERSION_ERROR            = 0x0001f000 // check update error
	VERSION_COMPARISON_ERROR = 0x0001f001 // delta version calculation error

	// server connection package

	CLIENT_REQ_ERROR         = 0x0002f000 // client request error
	BUILD_REQ_FLAG_ERROR     = 0x0002f001 // error while building request flag
	CLIENT_SOCKET_READ_ERROR = 0x0002f002 // error while reading client socket
	SERVER_DIAL_ERROR        = 0x0002f003 // error while dialing ms server
	JSON_MARSHAL_ERROR       = 0x0002f004 // error while building json object

	// config package

	LOAD_CONFIG_ERROR  = 0x0003f000 // error while loading config
	SAVE_CONFIG_ERROR  = 0x0003f001 // error while saving config to file
	CHECK_CONFIG_ERROR = 0x0003f002 // error while checking config
	LOAD_ICON_ERROR    = 0x0003f003 // error while loading icon

	// operative system package

	OS_NOT_SUPPORTED_ERROR = 0x0004f000 // OS not supported

	// utility package

	ANALYSIS_ERROR = 0x0005f001 // error while analyzing data

	// main

	CLIENT_LISTEN_ERROR = 0x0006f000 // error while listening for new clients
	CLIENT_ACCEPT_ERROR = 0x0006f001 // error while accepting new client

	// input package

	COMMAND_INPUT_ERROR   = 0x0007f000 // general error while reading command input
	COMMAND_UNKNOWN_ERROR = 0x0007f001 // command is unknown
	READ_INPUT_ERROR      = 0x0007f001 // error while reading input)
)
