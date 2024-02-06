package main

import "C"
import (
	"encoding/json"
	"fmt"

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

func stopAndAlert(alert string, err error) (resultErr error) {
	defer func() {
		if r := recover(); r != nil {
			resultErr = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	status = Stopped
	message := err.Error()
	fmt.Printf("Error: %s: %v\n", alert, err)
	msg, _ := json.Marshal(StatusMessage{Status: status, Alert: &alert, Message: &message})
	bridge.SendStringToPort(statusPropagationPort, string(msg))
	return nil
}
