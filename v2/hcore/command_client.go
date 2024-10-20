package hcore

import (
	"github.com/sagernet/sing-box/experimental/libbox"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

var _ libbox.CommandClientHandler = (*CommandClientHandler)(nil)

const (
	CommandCli = "core"
)

type CommandClientHandler struct {
	command int32
}

func (cch *CommandClientHandler) Connected() {
	Log(LogLevel_DEBUG, LogType_CORE, "CONNECTED")
}

func (cch *CommandClientHandler) Disconnected(message string) {
	Log(LogLevel_DEBUG, LogType_CORE, "DISCONNECTED: ", message)
}

func (cch *CommandClientHandler) ClearLog() {
	Log(LogLevel_DEBUG, LogType_CORE, "clear log")
}

func (cch *CommandClientHandler) WriteLog(message string) {
	Log(LogLevel_DEBUG, LogType_CORE, "log: ", message)
}

func (cch *CommandClientHandler) WriteStatus(message *libbox.StatusMessage) {
	systemInfoObserver.Emit(&SystemInfo{
		ConnectionsIn:  message.ConnectionsIn,
		ConnectionsOut: message.ConnectionsOut,
		Uplink:         message.Uplink,
		Downlink:       message.Downlink,
		UplinkTotal:    message.UplinkTotal,
		DownlinkTotal:  message.DownlinkTotal,
		Memory:         message.Memory,
		Goroutines:     message.Goroutines,
	})
	Log(LogLevel_DEBUG, LogType_CORE, "Memory: ", libbox.FormatBytes(message.Memory), ", Goroutines: ", message.Goroutines)
}

func (cch *CommandClientHandler) WriteGroups(message libbox.OutboundGroupIterator) {
	if message == nil {
		return
	}
	groups := OutboundGroupList{}
	for message.HasNext() {
		group := message.Next()
		items := group.GetItems()
		groupItems := []*OutboundInfo{}
		for items.HasNext() {
			item := items.Next()
			groupItems = append(groupItems,
				&OutboundInfo{
					Tag:          item.Tag,
					Type:         item.Type,
					UrlTestTime:  &timestamppb.Timestamp{Seconds: item.URLTestTime},
					UrlTestDelay: item.URLTestDelay,
				},
			)
		}
		groups.Items = append(groups.Items, &OutboundGroup{Tag: group.Tag, Type: group.Type, Selected: nil, Items: groupItems})
	}
	if cch.command == libbox.CommandGroupInfoOnly {
		mainOutboundsInfoObserver.Emit(&groups)
	} else {
		outboundsInfoObserver.Emit(&groups)
	}
}

func (cch *CommandClientHandler) InitializeClashMode(modeList libbox.StringIterator, currentMode string) {
	Log(LogLevel_DEBUG, LogType_CORE, "initial clash mode: ", currentMode)
}

func (cch *CommandClientHandler) UpdateClashMode(newMode string) {
	Log(LogLevel_DEBUG, LogType_CORE, "update clash mode: ", newMode)
}
