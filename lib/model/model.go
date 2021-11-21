package model

// struct adapted to config file
type Configuration struct {
	Server struct {
		Folder   string `yaml:"Folder"`
		FileName string `yaml:"FileName"`
		Version  string `yaml:"Version"`
		Protocol int    `yaml:"Protocol"`
	} `yaml:"Server"`
	Commands struct {
		StartServer         string `yaml:"StartServer"`
		StartServerParam    string `yaml:"StartServerParam"`
		StopServer          string `yaml:"StopServer"`
		StopServerAllowKill int    `yaml:"StopServerAllowKill"`
	} `yaml:"Commands"`
	Msh struct {
		Debug                         int    `yaml:"Debug"`
		InfoHibernation               string `yaml:"InfoHibernation"`
		InfoStarting                  string `yaml:"InfoStarting"`
		NotifyUpdate                  bool   `yaml:"NotifyUpdate"`
		ListenPort                    int    `yaml:"ListenPort"`
		TimeBeforeStoppingEmptyServer int64  `yaml:"TimeBeforeStoppingEmptyServer"`
	} `yaml:"Msh"`
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
