package hcore

import (
	"github.com/sagernet/sing-box/experimental/libbox"
)

var _ libbox.CommandClientHandler = (*CommandClientHandler)(nil)

type CommandClientHandler struct{}

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
		groupItems := []*OutboundGroupItem{}
		for items.HasNext() {
			item := items.Next()
			groupItems = append(groupItems,
				&OutboundGroupItem{
					Tag:          item.Tag,
					Type:         item.Type,
					UrlTestTime:  item.URLTestTime,
					UrlTestDelay: item.URLTestDelay,
				},
			)
		}
		groups.Items = append(groups.Items, &OutboundGroup{Tag: group.Tag, Type: group.Type, Selected: group.Selected, Items: groupItems})
	}
	outboundsInfoObserver.Emit(&groups)
	mainOutboundsInfoObserver.Emit(&groups)
}

func (cch *CommandClientHandler) InitializeClashMode(modeList libbox.StringIterator, currentMode string) {
	Log(LogLevel_DEBUG, LogType_CORE, "initial clash mode: ", currentMode)
}

func (cch *CommandClientHandler) UpdateClashMode(newMode string) {
	Log(LogLevel_DEBUG, LogType_CORE, "update clash mode: ", newMode)
}
