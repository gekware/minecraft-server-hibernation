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
		Debug                         int      `json:"Debug"`
		ID                            string   `json:"ID"`
		MshPort                       int      `json:"MshPort"`
		MshPortQuery                  int      `json:"MshPortQuery"`
		TimeBeforeStoppingEmptyServer int64    `json:"TimeBeforeStoppingEmptyServer"`
		SuspendAllow                  bool     `json:"SuspendAllow"`   // specify if msh should suspend java server process
		SuspendRefresh                int      `json:"SuspendRefresh"` // specify if msh should refresh java server process suspension and every how many seconds
		InfoHibernation               string   `json:"InfoHibernation"`
		InfoStarting                  string   `json:"InfoStarting"`
		NotifyUpdate                  bool     `json:"NotifyUpdate"`
		NotifyMessage                 bool     `json:"NotifyMessage"`
		Whitelist                     []string `json:"Whitelist"`
		WhitelistImport               bool     `json:"WhitelistImport"`
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
	ProtV int `json:"prot-v"` // msh protocol version
	Msh   struct {
		V            string `json:"msh-v"`         // msh version
		ID           string `json:"id"`            // msh id
		Uptime       int    `json:"uptime"`        // msh uptime
		SuspendAllow bool   `json:"allow-suspend"` // specify if msh hibernates ms by suspending process
		Sgm          struct {
			PreTerm  bool    `json:"preterm"`         // segment ended before expiration
			Dur      int     `json:"seconds"`         // segment duration in seconds
			HibeDur  int     `json:"seconds-hibe"`    // segment seconds in which ms server was hibernating
			PlaySec  int     `json:"play-second-sum"` // segment play seconds (sum of seconds spent playing)
			UsageCpu float64 `json:"cpu-usage"`       // segment usage of cpu by msh
			UsageMem float64 `json:"mem-usage"`       // segment usage of memory by msh
		} `json:"sgm"`
	} `json:"msh"`
	Machine struct {
		Os        string `json:"os"`
		Arch      string `json:"arch"`
		JavaV     string `json:"java-v"`
		CpuModel  string `json:"cpu-model"`  // cpu model
		CpuVendor string `json:"cpu-vendor"` // cpu vendor
		CoresMsh  int    `json:"cores-msh"`  // cores for msh
		CoresSys  int    `json:"cores-sys"`  // cores for system
		Mem       int64  `json:"mem"`        // system memory
	} `json:"machine"`
	Server struct {
		Uptime int    `json:"uptime"`  // mc server uptime
		V      string `json:"ms-v"`    // mc server version
		Prot   int    `json:"ms-prot"` // mc server protocol
	} `json:"server"`
}

type Api2Res struct {
	Result string `json:"result"`
	Dev    struct {
		V      string `json:"version"`
		Date   string `json:"date"`
		Commit string `json:"commit"`
	} `json:"dev"`
	Official struct {
		V      string `json:"version"`
		Date   string `json:"date"`
		Commit string `json:"commit"`
	} `json:"official"`
	Deprecated struct {
		V      string `json:"version"`
		Date   string `json:"date"`
		Commit string `json:"commit"`
	} `json:"deprecated"`
	Messages []string `json:"messages"`
}

// struct for in game raw message
type GameRawMessage struct {
	Text  string `json:"text"`
	Color string `json:"color"`
	Bold  bool   `json:"bold"`
}

// struct for version.json of server JAR.
// use 2 version json definitions as it might change depending on ms version.
type VersionInfo struct {
	Version1 string `json:"release_target"`
	Version2 string `json:"name"`
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

// struct for minecraft server whitelist file
type MSWhitelist struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}
