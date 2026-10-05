package hcore

import (
	"strings"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/monitoring"
	C "github.com/sagernet/sing-box/constant"
	G "github.com/sagernet/sing-box/protocol/group"
	"github.com/sagernet/sing-box/protocol/group/balancer"
	E "github.com/sagernet/sing/common/exceptions"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/service"
	"google.golang.org/grpc"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// groupNow returns the tag currently selected by an outbound group, or "" if none.
func groupNow(group adapter.OutboundGroup) string {
	if selected := group.Selected(N.NetworkTCP); selected != nil {
		return selected.Tag()
	}
	return ""
}

func outboundDisplayType(outbound adapter.Outbound) string {
	if displayer, ok := outbound.(interface{ DisplayType() string }); ok {
		return displayer.DisplayType()
	}
	return C.ProxyDisplayName(outbound.Type())
}

func (h *HiddifyInstance) GetProxyInfo(url_test_history *adapter.URLTestHistory, detour adapter.Outbound) *OutboundInfo {
	// historyStorage := h.UrlTestHistory()
	// if historyStorage == nil {
	// 	return nil
	// }

	out := &OutboundInfo{}
	// realTag := ""

	out.Tag = detour.Tag()
	out.Type = outboundDisplayType(detour)
	if group, isGroup := detour.(adapter.OutboundGroup); isGroup {
		out.IsGroup = true
		gnow := groupNow(group)
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
	if deps := detour.Dependencies(); len(deps) == 1 {
		out.Detour = deps[0]
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
	for _, it := range outbounds_converted {
		if it.Detour == "" {
			continue
		}
		if det, ok := outbounds_converted[it.Detour]; ok {
			it.TagDisplay += " → " + det.TagDisplay
			it.Type += " → " + det.Type
		}
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
		selectedTag := groupNow(iGroup)
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
			if onlyGroupitems && pinfo.GroupSelectedTagDisplay != nil && pinfo.TagDisplay != *pinfo.GroupSelectedTagDisplay {
				pinfo.TagDisplay = pinfo.TagDisplay + " → " + *pinfo.GroupSelectedTagDisplay
			}
			group.Items = append(group.Items, pinfo)
			pinfo.IsVisible = !strings.Contains(itemTag, "§hide§")

		}
		if len(group.Items) == 0 {
			continue
		}

		groups.Items = append(groups.Items, &group)

		if onlyGroupitems && group.Tag == config.OutboundSelectTag {
			showEntryHop(&group, outbounds_converted, func(tag string) string {
				if out, ok := box.Outbound().Outbound(tag); ok {
					return C.ProxyDisplayName(out.Type())
				}
				if ep, ok := box.Endpoint().Get(tag); ok {
					return C.ProxyDisplayName(ep.Type())
				}
				return TrimTagName(tag)
			})
		}

	}

	return &groups
}

// showEntryHop makes the hop that traffic enters before the selected proxy (extra security, or
// legacy WARP) the active entry of the main group, since its delay and IP are what the user gets.
// It is labelled "<hop protocol> → <selected config>", e.g. "Psiphon → my-server".
func showEntryHop(group *OutboundGroup, infos map[string]*OutboundInfo, protocolName func(tag string) string) {
	for _, entryTag := range []string{config.ChainExtraSecurityTag, config.WARPConfigTag} {
		entry, ok := infos[entryTag]
		if !ok {
			continue
		}
		entry.TagDisplay = protocolName(entryTag)
		if selected, ok := infos[group.Selected]; ok {
			entry.TagDisplay += " → " + selected.TagDisplay
		}
		group.Selected = entry.Tag
		group.Items = append([]*OutboundInfo{entry}, group.Items...)
		return
	}
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

		stream.Send(h.GetAllProxiesInfo(monitor.OutboundsHistory(""), onlyMain))

		urltestch, err := monitor.SubscribeGroup("")
		if err != nil {
			Log(LogLevel_ERROR, LogType_CORE, "failed to send outbounds info: ", err)
			// return err
		}
		defer monitor.UnsubscribeGroup("", urltestch)

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
				if err := stream.Send(h.GetAllProxiesInfo(monitor.OutboundsHistory(""), onlyMain)); err != nil {
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
