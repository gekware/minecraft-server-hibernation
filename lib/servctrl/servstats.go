package servctrl

type serverStats struct {
	// ServerStatus represent the status of the minecraft server ("offline", "starting", "online", "stopping")
	Status string
	// Players keeps track of players connected to the server
	Players int
	// StopInstances keeps track of how many times stopEmptyMinecraftServer() has been called in the last {TimeBeforeStoppingEmptyServer} seconds
	StopInstances int
	// LoadProgress indicates the loading percentage while the server is starting
	LoadProgress string
}

// ServStats contains the info relative to server
var ServStats *serverStats

func init() {
	ServStats = &serverStats{
		Status:        "offline",
		Players:       0,
		StopInstances: 0,
		LoadProgress:  "0%",
	}
}
