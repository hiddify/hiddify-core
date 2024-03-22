package v2

import (
	pb "github.com/hiddify/hiddify-core/hiddifyrpc"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
)

type CommandClientHandler struct {
	port   int64
	logger log.Logger
}

func (cch *CommandClientHandler) Connected() {
	cch.logger.Debug("CONNECTED")
}

func (cch *CommandClientHandler) Disconnected(message string) {
	cch.logger.Debug("DISCONNECTED: ", message)
}

func (cch *CommandClientHandler) ClearLog() {
	cch.logger.Debug("clear log")
}

func (cch *CommandClientHandler) WriteLog(message string) {
	cch.logger.Debug("log: ", message)
}

func (cch *CommandClientHandler) WriteStatus(message *libbox.StatusMessage) {
	systemInfoObserver.Emit(pb.SystemInfo{
		ConnectionsIn:  message.ConnectionsIn,
		ConnectionsOut: message.ConnectionsOut,
		Uplink:         message.Uplink,
		Downlink:       message.Downlink,
		UplinkTotal:    message.UplinkTotal,
		DownlinkTotal:  message.DownlinkTotal,
		Memory:         message.Memory,
		Goroutines:     message.Goroutines,
	})
	cch.logger.Debug("Memory: ", libbox.FormatBytes(message.Memory), ", Goroutines: ", message.Goroutines)

}

func (cch *CommandClientHandler) WriteGroups(message libbox.OutboundGroupIterator) {
	if message == nil {
		return
	}
	groups := pb.OutboundGroupList{}
	for message.HasNext() {
		group := message.Next()
		items := group.GetItems()
		groupItems := []*pb.OutboundGroupItem{}
		for items.HasNext() {
			item := items.Next()
			groupItems = append(groupItems,
				&pb.OutboundGroupItem{
					Tag:          item.Tag,
					Type:         item.Type,
					UrlTestTime:  item.URLTestTime,
					UrlTestDelay: item.URLTestDelay,
				},
			)
		}
		groups.Items = append(groups.Items, &pb.OutboundGroup{Tag: group.Tag, Type: group.Type, Selected: group.Selected, Items: groupItems})
	}
	outboundsInfoObserver.Emit(groups)
	mainOutboundsInfoObserver.Emit(groups)
}

func (cch *CommandClientHandler) InitializeClashMode(modeList libbox.StringIterator, currentMode string) {
	cch.logger.Debug("initial clash mode: ", currentMode)
}

func (cch *CommandClientHandler) UpdateClashMode(newMode string) {
	cch.logger.Debug("update clash mode: ", newMode)
}
