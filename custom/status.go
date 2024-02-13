package main

import "C"
import (
	"encoding/json"
	"fmt"

	"github.com/hiddify/libcore/bridge"
	"github.com/hiddify/libcore/config"
)

var statusPropagationPort int64
var status = Stopped

type StatusMessage struct {
	Status  string  `json:"status"`
	Alert   *string `json:"alert"`
	Message *string `json:"message"`
}

func propagateStatus(newStatus string) {
	status = newStatus

	msg, _ := json.Marshal(StatusMessage{Status: status})
	bridge.SendStringToPort(statusPropagationPort, string(msg))
}

func stopAndAlert(alert string, err error) (resultErr error) {
	defer func() {
		if r := recover(); r != nil {
			resultErr = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	fmt.Printf("Error: %s: %v\n", alert, err)
	propagateStatus(Stopped)
	config.DeactivateTunnelService()
	if commandServer != nil {
		commandServer.SetService(nil)
	}
	if box != nil {
		box.Close()
		box = nil
	}
	if commandServer != nil {
		commandServer.Close()
	}
	return nil
}
