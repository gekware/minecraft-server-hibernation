package servctrl

type serverStats struct {
	// ServerStatus represent the status of the minecraft server ("offline", "starting", "online")
	Status string
	// Players keeps track of players connected to the server
	Players int
	// StopInstances keeps track of how many times stopEmptyMinecraftServer() has been called in the last {TimeBeforeStoppingEmptyServer} seconds
	StopInstances int
	// TimeLeftUntilUp keeps track of how many seconds are still needed to reach serverStatus == "online"
	TimeLeftUntilUp int
}

// ServStats contains the info relative to server
var ServStats *serverStats

func init() {
	ServStats = &serverStats{
		Status:        "offline",
		Players:       0,
		StopInstances: 0,
	}
}
