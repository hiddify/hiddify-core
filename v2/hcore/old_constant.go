package hcore

const (
	Stopped  = "Stopped"
	Starting = "Starting"
	Started  = "Started"
	Stopping = "Stopping"
)

const (
	EmptyConfiguration = "EmptyConfiguration"
	StartCommandServer = "StartCommandServer"
	CreateService      = "CreateService"
)

func convert2OldState(newStatus CoreStates) string {
	if newStatus == CoreStates_STOPPED {
		return Stopped
	}
	if newStatus == CoreStates_STARTED {
		return Started
	}
	if newStatus == CoreStates_STARTING {
		return Starting
	}
	if newStatus == CoreStates_STOPPING {
		return Stopping
	}
	return "Invalid"
}

type StatusMessage struct {
	Status  string  `json:"status"`
	Alert   *string `json:"alert"`
	Message *string `json:"message"`
}
