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
		ListenPort                    int      `json:"ListenPort"`
		TimeBeforeStoppingEmptyServer int64    `json:"TimeBeforeStoppingEmptyServer"`
		Whitelist                     []string `json:"Whitelist"`
	} `json:"Msh"`
}

type DataTxt struct {
	Text string `json:"text"`
}

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
		Mshv         string `json:"msh-v"`         // msh version
		ID           string `json:"id"`            // msh id
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
		Os       string `json:"os"`
		Platform string `json:"platform"`
		Javav    string `json:"java-v"`
		Stats    struct {
			CoresMsh int `json:"cores-msh"` // cores for msh
			Cores    int `json:"cores"`     // cores for system
			Mem      int `json:"mem"`       // memory dedicated to system
		} `json:"stats"`
	} `json:"machine"`
	Server struct {
		Uptime   int    `json:"uptime"`    // mc server uptime
		Minev    string `json:"mine-v"`    // mc server version
		MineProt int    `json:"mine-prot"` // mc server protocol
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
