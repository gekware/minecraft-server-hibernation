package errco

/*
0xxxxxfxxx: error

0x0000xxxx: server control package
0x0001xxxx: program manager package
0x0002xxxx: server connection package
0x0003xxxx: config package
0x0004xxxx: operative system
*/

// ------------------- codes ------------------- //

const (
	SERVER_STATUS_OFFLINE  = 0x00000000
	SERVER_STATUS_STARTING = 0x00000001
	SERVER_STATUS_ONLINE   = 0x00000002
	SERVER_STATUS_STOPPING = 0x00000003

	VERSION_UPDATED           = 0x00010000 // check update result: msh updated
	VERSION_UPDATEAVAILABLE   = 0x00010001 // check update result: update available
	VERSION_UNOFFICIALVERSION = 0x00010002 // check update result: msh is running unofficial version

	CLIENT_REQ_UNKN     = 0x00020000 // client request unknown
	CLIENT_REQ_INFO     = 0x00020001 // client request server info
	CLIENT_REQ_JOIN     = 0x00020002 // client request server join
	MESSAGE_FORMAT_TXT  = 0x00020003 // message to client should be built as TXT
	MESSAGE_FORMAT_INFO = 0x00020004 // message to client should be built as INFO
)

// ------------------- errors ------------------ //

const (
	VERSION_ERROR            = 0x0001f000 // check update error
	VERSION_COMPARISON_ERROR = 0x0001f001 // delta version calculation error

	CLIENT_REQ_ERROR = 0x0002f000 // client request error

	LOAD_CONFIG_ERROR  = 0x0003f000 // error while loading config
	SAVE_CONFIG_ERROR  = 0x0003f001 // error while saving config to file
	CHECK_CONFIG_ERROR = 0x0003f002 // error while checking config

	OS_NOT_SUPPORTED_ERROR = 0x0004f001 // OS not supported
)
