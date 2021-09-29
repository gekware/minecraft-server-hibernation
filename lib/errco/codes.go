package errco

// ------------------- codes ------------------- //

const (
	SERVER_STATUS_OFFLINE  = 0x00000000
	SERVER_STATUS_STARTING = 0x00000001
	SERVER_STATUS_ONLINE   = 0x00000002
	SERVER_STATUS_STOPPING = 0x00000003

	VERSION_UPDATED           = 0x00010000 // check update result: msh updated
	VERSION_UPDATEAVAILABLE   = 0x00010001 // check update result: update available
	VERSION_UNOFFICIALVERSION = 0x00010002 // check update result: msh is running unofficial version

	CLIENT_REQ_UNKN = 0x00020000 // client request unknown
	CLIENT_REQ_INFO = 0x00020001 // client request server info
	CLIENT_REQ_JOIN = 0x00020002 // client request server join

	MESSAGE_FORMAT_TXT  = 0x00030001 // message to client should be built as TXT
	MESSAGE_FORMAT_INFO = 0x00030002 // message to client should be built as INFO
)

// ------------------- errors ------------------ //

const (
	VERSION_ERROR = 0x0001f000 // check update result: error

	CLIENT_REQ_ERROR = 0x0002f000 // client request result: error
)
