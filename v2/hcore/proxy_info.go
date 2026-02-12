package hcore

import (
	"strings"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/monitoring"
	G "github.com/sagernet/sing-box/protocol/group"
	"github.com/sagernet/sing-box/protocol/group/balancer"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/service"
	"google.golang.org/grpc"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

func (h *HiddifyInstance) GetProxyInfo(url_test_history *adapter.URLTestHistory, detour adapter.Outbound) *OutboundInfo {
	// historyStorage := h.UrlTestHistory()
	// if historyStorage == nil {
	// 	return nil
	// }

	out := &OutboundInfo{}
	// realTag := ""

	out.Tag = detour.Tag()
	out.Type = detour.DisplayType()
	if group, isGroup := detour.(adapter.OutboundGroup); isGroup {
		out.IsGroup = true
		gnow := group.Now()
		out.GroupSelectedTag = &gnow
	}
	out.TagDisplay = TrimTagName(out.Tag)

	if tag := monitoring.RealTag(detour); tag != "" {
		dtag := TrimTagName(tag)
		out.GroupSelectedTagDisplay = &dtag
		if balancer, ok := detour.(*balancer.Balancer); ok {
			if stg := balancer.Strategy(); stg != "lowest-delay" {
				out.GroupSelectedTagDisplay = &stg
			}
		}
	}
	// realTag = adapter.OutboundTag(detour)

	// realTag = out.Tag

	// url_test_history := historyStorage.LoadURLTestHistory(realTag)
	if trafficManager := h.TrafficManager(); trafficManager != nil {
		up, down := trafficManager.OutboundUsage(out.Tag)
		out.Upload = up
		out.Download = down

	}
	if url_test_history != nil {
		out.UrlTestTime = timestamppb.New(url_test_history.Time)
		out.UrlTestDelay = int32(url_test_history.Delay)
		if url_test_history.IsFromCache {
			out.UrlTestDelay = 0
		}
		if url_test_history.IpInfo != nil {
			out.Ipinfo = &IpInfo{
				Ip:          url_test_history.IpInfo.IP,
				CountryCode: url_test_history.IpInfo.CountryCode,
				Region:      url_test_history.IpInfo.Region,
				City:        url_test_history.IpInfo.City,
				Asn:         int32(url_test_history.IpInfo.ASN),
				Org:         url_test_history.IpInfo.Org,
				Latitude:    url_test_history.IpInfo.Latitude,
				Longitude:   url_test_history.IpInfo.Longitude,
				PostalCode:  url_test_history.IpInfo.PostalCode,
			}
		}

	}

	return out
}

func (h *HiddifyInstance) GetAllProxiesInfo(hismap map[string]*adapter.URLTestHistory, onlyGroupitems bool) *OutboundGroupList {
	ctx, box := h.Context(), h.Box()
	if ctx == nil || box == nil {
		return nil
	}
	cacheFile := service.FromContext[adapter.CacheFile](ctx)

	outbounds_converted := make(map[string]*OutboundInfo, 0)
	var iGroups []adapter.OutboundGroup
	for _, it := range box.Endpoint().Endpoints() {
		his, _ := hismap[it.Tag()]
		outbounds_converted[it.Tag()] = h.GetProxyInfo(his, it)
	}
	for _, it := range box.Outbound().Outbounds() {
		his, _ := hismap[it.Tag()]
		outbounds_converted[it.Tag()] = h.GetProxyInfo(his, it)

	}
	for _, it := range box.Outbound().Outbounds() {
		if group, isGroup := it.(adapter.OutboundGroup); isGroup {
			iGroups = append(iGroups, group)
			// up := 0
			// down := 0
			// for _, itemTag := range group.All() {
			// 	if pinfo, ok := outbounds_converted[itemTag]; ok {
			// 		up += int(pinfo.Upload)
			// 		down += int(pinfo.Download)
			// 	}
			// }
			// outbounds_converted[it.Tag()].Upload += int64(up)
			// outbounds_converted[it.Tag()].Download += int64(down)
		}
	}

	var groups OutboundGroupList
	for _, iGroup := range iGroups {
		var group OutboundGroup
		group.Tag = iGroup.Tag()
		group.Type = iGroup.Type()
		_, group.Selectable = iGroup.(*G.Selector)
		selectedTag := iGroup.Now()
		group.Selected = selectedTag

		// outbounds_converted[iGroup.Tag()].GroupSelectedOutbound = &group.Selected
		if cacheFile != nil {
			if isExpand, loaded := cacheFile.LoadGroupExpand(group.Tag); loaded {
				group.IsExpand = isExpand
			}
		}

		for _, itemTag := range iGroup.All() {
			if onlyGroupitems && itemTag != selectedTag {
				continue
			}
			pinfo := outbounds_converted[itemTag]
			pinfo.IsSelected = itemTag == selectedTag

			group.Items = append(group.Items, pinfo)
			pinfo.IsVisible = !strings.Contains(itemTag, "§hide§")

		}
		if len(group.Items) == 0 {
			continue
		}

		groups.Items = append(groups.Items, &group)
	}

	return &groups
}

func TrimTagName(tag string) string {
	return strings.Trim(strings.Split(tag, "§")[0], " ")
}

func (s *CoreService) OutboundsInfo(req *hcommon.Empty, stream grpc.ServerStreamingServer[OutboundGroupList]) error {
	return static.AllProxiesInfoStream(stream, false)
}

func (s *CoreService) MainOutboundsInfo(req *hcommon.Empty, stream grpc.ServerStreamingServer[OutboundGroupList]) error {
	return static.AllProxiesInfoStream(stream, true)
}

func (h *HiddifyInstance) AllProxiesInfoStream(stream grpc.ServerStreamingServer[OutboundGroupList], onlyMain bool) error {
	// stream.Send(&OutboundGroupList{})
	h.MakeSureContextIsNew(stream.Context())

	if ctx, urlTestHistory := h.Context(), h.UrlTestHistory(); ctx != nil && urlTestHistory != nil {
		monitor := monitoring.Get(ctx)

		stream.Send(h.GetAllProxiesInfo(monitor.OutboundsHistory(config.OutboundSelectTag), onlyMain))

		urltestch, err := monitor.SubscribeGroup(config.OutboundSelectTag)
		if err != nil {
			Log(LogLevel_ERROR, LogType_CORE, "failed to send outbounds info: ", err)
			// return err
		}
		defer monitor.UnsubscribeGroup(config.OutboundSelectTag, urltestch)

		// timer2 := time.NewTicker(10 * time.Second)
		// defer timer2.Stop()
		debounceWindow := 1000 * time.Millisecond
		var (
			timer   *time.Timer
			timerCh <-chan time.Time
		)
		defer func() {
			if timer != nil {
				timer.Stop()
			}
		}()
		for {
			select {
			case <-stream.Context().Done():
				return nil
			case <-ctx.Done():
				return nil
			case _, ok := <-urltestch:
				if !ok {
					return nil
				}
				if timer == nil {
					timer = time.NewTimer(debounceWindow)
					timerCh = timer.C
				}
			case <-timerCh:
				if err := stream.Send(h.GetAllProxiesInfo(monitor.OutboundsHistory(config.OutboundSelectTag), onlyMain)); err != nil {
					Log(LogLevel_ERROR, LogType_CORE, "failed to send outbounds info: ", err)
					// return err
				}
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer = nil
				timerCh = nil

			}
		}
	}

	return E.New("hiddify service not found")
}
