package main

import (
	"encoding/json"
	"fmt"

	"github.com/hiddify/libcore/bridge"
	"github.com/sagernet/sing-box/experimental/libbox"
)

type CommandClientHandler struct {
	port int64
}

func (cch *CommandClientHandler) Connected() {
	fmt.Println("connected")
}

func (cch *CommandClientHandler) Disconnected(message string) {
	fmt.Printf("disconnected: %s\n", message)
}

func (cch *CommandClientHandler) WriteLog(message string) {
	fmt.Printf("new log: %s\n", message)
}

func (cch *CommandClientHandler) WriteStatus(message *libbox.StatusMessage) {
	msg, err := json.Marshal(
		map[string]int64{
			"connections-in":  int64(message.ConnectionsIn),
			"connections-out": int64(message.ConnectionsOut),
			"uplink":          message.Uplink,
			"downlink":        message.Downlink,
			"uplink-total":    message.UplinkTotal,
			"downlink-total":  message.DownlinkTotal,
		},
	)
	if err != nil {
		bridge.SendStringToPort(cch.port, fmt.Sprintf("error: %e", err))
	} else {
		bridge.SendStringToPort(cch.port, string(msg))
	}
}

func (cch *CommandClientHandler) WriteGroups(message libbox.OutboundGroupIterator) {}

func (cch *CommandClientHandler) InitializeClashMode(modeList libbox.StringIterator, currentMode string) {
}

func (cch *CommandClientHandler) UpdateClashMode(newMode string) {}
