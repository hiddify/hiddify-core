package v2

import (
	pb "github.com/hiddify/hiddify-core/hiddifyrpc"
)

var (
	systemInfoObserver        = NewObserver[*pb.SystemInfo](10)
	outboundsInfoObserver     = NewObserver[*pb.OutboundGroupList](10)
	mainOutboundsInfoObserver = NewObserver[*pb.OutboundGroupList](10)
)
