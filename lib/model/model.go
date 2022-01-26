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
