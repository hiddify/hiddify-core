package main

import "C"
import (
	"encoding/json"

	"github.com/hiddify/libcore/bridge"
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

func stopAndAlert(alert string, err error) error {
	status = Stopped
	message := err.Error()

	msg, _ := json.Marshal(StatusMessage{Status: status, Alert: &alert, Message: &message})
	bridge.SendStringToPort(statusPropagationPort, string(msg))
	return err
}
