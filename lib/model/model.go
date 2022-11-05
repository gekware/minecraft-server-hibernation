package model

// struct adapted to config file
type Configuration struct {
	Server struct {
		Folder   string `json:"Folder"`
		FileName string `json:"FileName"`
		Version  string `json:"Version"`
		Protocol int    `json:"Protocol"`
	} `json:"Server"`
	Commands struct {
		StartServer         string `json:"StartServer"`
		StartServerParam    string `json:"StartServerParam"`
		StopServer          string `json:"StopServer"`
		StopServerAllowKill int    `json:"StopServerAllowKill"`
	} `json:"Commands"`
	Msh struct {
		ID                            string   `json:"ID"`
		Debug                         int      `json:"Debug"`
		AllowSuspend                  bool     `json:"AllowSuspend"` // specify if msh should suspend java server process
		InfoHibernation               string   `json:"InfoHibernation"`
		InfoStarting                  string   `json:"InfoStarting"`
		NotifyUpdate                  bool     `json:"NotifyUpdate"`
		NotifyMessage                 bool     `json:"NotifyMessage"`
		ListenPort                    int      `json:"ListenPort"`
		TimeBeforeStoppingEmptyServer int64    `json:"TimeBeforeStoppingEmptyServer"`
		Whitelist                     []string `json:"Whitelist"`
	} `json:"Msh"`
}

// struct for message format txt
type DataTxt struct {
	Text string `json:"text"`
}

// struct for message format info
type DataInfo struct {
	Description struct {
		Text string `json:"text"`
	} `json:"description"`
	Players struct {
		Max    int `json:"max"`
		Online int `json:"online"`
	} `json:"players"`
	Version struct {
		Name     string `json:"name"`
		Protocol int    `json:"protocol"`
	} `json:"version"`
	Favicon string `json:"favicon"`
}

type Api2Req struct {
	Protv int `json:"prot-v"` // msh protocol version
	Msh   struct {
		ID           string `json:"id"`            // msh id
		Mshv         string `json:"msh-v"`         // msh version
		Uptime       int    `json:"uptime"`        // msh uptime
		AllowSuspend bool   `json:"allow-suspend"` // specify if msh hibernates ms by suspending process
		Sgm          struct {
			Seconds     int     `json:"seconds"`         // segment duration in seconds
			SecondsHibe int     `json:"seconds-hibe"`    // segment seconds in which ms server was hibernating
			CpuUsage    float64 `json:"cpu-usage"`       // segment cpu usage by msh
			MemUsage    float64 `json:"mem-usage"`       // segment memory usage by msh
			PlayerSec   int     `json:"play-second-sum"` // segment player seconds (total seconds spent playing)
			PreTerm     bool    `json:"preterm"`         // segment forcefully ended preterm
		} `json:"sgm"`
	} `json:"msh"`
	Machine struct {
		Os        string `json:"os"`
		Arch      string `json:"arch"`
		Javav     string `json:"java-v"`
		CpuModel  string `json:"cpu-model"`  // cpu model
		CpuVendor string `json:"cpu-vendor"` // cpu vendor
		CoresMsh  int    `json:"cores-msh"`  // cores for msh
		CoresSys  int    `json:"cores-sys"`  // cores for system
		Mem       int    `json:"mem"`        // system memory
	} `json:"machine"`
	Server struct {
		Uptime int    `json:"uptime"`  // mc server uptime
		Msv    string `json:"ms-v"`    // mc server version
		MsProt int    `json:"ms-prot"` // mc server protocol
	} `json:"server"`
}

type Api2Res struct {
	Result string `json:"result"`
	Dev    struct {
		Version string `json:"version"`
		Date    string `json:"date"`
		Commit  string `json:"commit"`
	} `json:"dev"`
	Official struct {
		Version string `json:"version"`
		Date    string `json:"date"`
		Commit  string `json:"commit"`
	} `json:"official"`
	Deprecated struct {
		Version string `json:"version"`
		Date    string `json:"date"`
		Commit  string `json:"commit"`
	} `json:"deprecated"`
	Messages []string `json:"messages"`
}

// struct for in game raw message
type GameRawMessage struct {
	Text  string `json:"text"`
	Color string `json:"color"`
	Bold  bool   `json:"bold"`
}

// struct for version.json of server JAR
type VersionInfo struct {
	Version  string `json:"release_target"`
	Protocol int    `json:"protocol_version"`
}

// struct for msh instance file version
type MshInstanceV struct {
	V int `json:"V"`
}

// struct for msh instance file (V0)
type MshInstanceV0 struct {
	V        int    `json:"V"`
	CFlag    string `json:"CFlag"`
	MId      string `json:"MId"`
	HostName string `json:"HostName"`
	FId      uint64 `json:"FId"`
	MshId    string `json:"MshId"`
	CheckSum string `json:"CheckSum"`
}
