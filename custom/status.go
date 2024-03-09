package main

import "C"
import (
	"encoding/json"
	"fmt"

	"github.com/hiddify/libcore/bridge"
	"github.com/hiddify/libcore/config"
	pb "github.com/hiddify/libcore/hiddifyrpc"
	v2 "github.com/hiddify/libcore/v2"
)

var statusPropagationPort int64

// var status = Stopped

type StatusMessage struct {
	Status  string  `json:"status"`
	Alert   *string `json:"alert"`
	Message *string `json:"message"`
}

func propagateStatus(newStatus pb.CoreState) {
	v2.CoreState = newStatus

	msg, _ := json.Marshal(StatusMessage{Status: convert2OldState(v2.CoreState)})
	bridge.SendStringToPort(statusPropagationPort, string(msg))
}

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

func stopAndAlert(alert string, err error) (resultErr error) {
	defer config.DeferPanicToError("stopAndAlert", func(err error) {
		resultErr = err
	})
	v2.CoreState = pb.CoreState_STOPPED
	message := err.Error()
	fmt.Printf("Error: %s: %s\n", alert, message)
	msg, _ := json.Marshal(StatusMessage{Status: convert2OldState(v2.CoreState), Alert: &alert, Message: &message})
	bridge.SendStringToPort(statusPropagationPort, string(msg))

	go config.DeactivateTunnelService()
	if commandServer != nil {
		commandServer.SetService(nil)
	}
	if v2.Box != nil {
		v2.Box.Close()
		v2.Box = nil
	}
	if commandServer != nil {
		commandServer.Close()
	}
	return nil
}
