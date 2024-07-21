package v2

import pb "github.com/hiddify/hiddify-core/hiddifyrpc"

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

func convert2OldState(newStatus pb.CoreState) string {
	if newStatus == pb.CoreState_STOPPED {
		return Stopped
	}
	if newStatus == pb.CoreState_STARTED {
		return Started
	}
	if newStatus == pb.CoreState_STARTING {
		return Starting
	}
	if newStatus == pb.CoreState_STOPPING {
		return Stopping
	}
	return "Invalid"
}

type StatusMessage struct {
	Status  string  `json:"status"`
	Alert   *string `json:"alert"`
	Message *string `json:"message"`
}
