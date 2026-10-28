package hcore

import (
	"context"
	"strings"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/monitoring"
	G "github.com/sagernet/sing-box/protocol/group"
	"github.com/sagernet/sing-box/protocol/group/balancer"
	E "github.com/sagernet/sing/common/exceptions"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/service"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// groupNow returns the tag currently selected by an outbound group, or "" if none.
func groupNow(group adapter.OutboundGroup) string {
	if selected := group.Selected(N.NetworkTCP); selected != nil {
		return selected.Tag()
	}
	return ""
}

func (h *HiddifyInstance) GetProxyInfo(url_test_history *adapter.URLTestHistory, detour adapter.Outbound) *OutboundInfo {
	// historyStorage := h.UrlTestHistory()
	// if historyStorage == nil {
	// 	return nil
	// }

	out := &OutboundInfo{}
	// realTag := ""

	out.Tag = detour.Tag()
	out.Type = adapter.DisplayType(detour) // the type, and "Connecting..." while not ready
	out.Status = Status(detour.Status())
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

	// the hops of the selectable chain mode: listed in its own group, not as entries
	chainMembers := make(map[string]bool)
	for _, iGroup := range iGroups {
		if isChainTag(iGroup.Tag()) {
			for _, member := range iGroup.All() {
				chainMembers[member] = true
			}
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
			if pinfo == nil {
				continue
			}
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
		if isChainTag(group.Tag) {
			showChainGroup(&group, outbounds_converted)
		}

		groups.Items = append(groups.Items, &group)

		if onlyGroupitems && group.Tag == config.OutboundSelectTag {
			showEntryHop(&group, outbounds_converted, chainMembers)
		}

	}

	appendChainHops(&groups, outbounds_converted, chainMembers)

	return &groups
}

// appendChainHops exposes active extra-security / unblocker hops (normally §hide§) so
// clients can show status, ping and real IP — named with the WARP mode (WireGuard / MASQUE / AWG).
func appendChainHops(groups *OutboundGroupList, infos map[string]*OutboundInfo, chainMembers map[string]bool) {
	var items []*OutboundInfo
	for tag, info := range infos {
		if info == nil || !isChainTag(tag) || chainMembers[tag] {
			continue
		}
		cloned := proto.Clone(info).(*OutboundInfo)
		cloned.IsVisible = true
		// Prefer DisplayType (keeps the connecting mark) over the static tag name.
		if hopType := chainHopType(tag, infos); hopType != "" {
			cloned.TagDisplay = hopType
		} else {
			cloned.TagDisplay = TrimTagName(tag)
		}
		items = append(items, cloned)
	}
	if len(items) == 0 {
		return
	}
	groups.Items = append(groups.Items, &OutboundGroup{
		Tag:        "chain-hops",
		Type:       "chain",
		Selectable: false,
		IsExpand:   true,
		Items:      items,
	})
}

// showEntryHop makes the hop that traffic enters before the selected proxy (extra security)
// the active entry of the main group, since its delay and IP are what the user gets.
// Labelled from DisplayType, e.g. "WARP · WG → my-server", or "WARP · WG → my-server ⏳
// Connecting..." while the hop is still starting.
func showEntryHop(group *OutboundGroup, infos map[string]*OutboundInfo, chainMembers map[string]bool) {
	for entryTag, entry := range infos {
		if entry == nil || !config.IsChainExtraSecurityTag(entryTag) || chainMembers[entryTag] {
			continue
		}
		base, connecting := adapter.SplitStatus(chainHopType(entryTag, infos))
		if base == "" {
			base = TrimTagName(entry.Tag)
		}
		entry.TagDisplay = base
		if selected, ok := infos[group.Selected]; ok {
			entry.TagDisplay = base + " → " + selected.TagDisplay
		}
		if connecting != "" {
			entry.TagDisplay += " " + connecting
		}
		entry.IsVisible = true
		group.Selected = entry.Tag
		group.Items = append([]*OutboundInfo{entry}, group.Items...)
		return
	}
}

func isChainTag(tag string) bool {
	return config.IsChainExtraSecurityTag(tag) || config.IsChainUnblockerTag(tag)
}

// chainHopType is the display type of the hop that tag sends through now: the hop itself, or
// for the selector of the selectable mode the hop it (or its lowest-delay balancer) selected.
func chainHopType(tag string, infos map[string]*OutboundInfo) string {
	for range 4 {
		info := infos[tag]
		if info == nil {
			return ""
		}
		if !info.IsGroup || info.GroupSelectedTag == nil || *info.GroupSelectedTag == "" {
			return info.Type
		}
		tag = *info.GroupSelectedTag
	}
	return ""
}

// showChainGroup makes the hops of the selectable chain mode visible in its selector, named by
// their display type (with the connecting mark), so the app can list them and move between them.
func showChainGroup(group *OutboundGroup, infos map[string]*OutboundInfo) {
	for i, item := range group.Items {
		cloned := proto.Clone(item).(*OutboundInfo)
		cloned.IsVisible = true
		if hopType := chainHopType(item.Tag, infos); hopType != "" {
			cloned.TagDisplay = hopType
			if cloned.IsGroup { // the lowest-delay balancer: "Lowest → WARP · AWG"
				cloned.TagDisplay = strings.TrimPrefix(TrimTagName(item.Tag), config.ChainExtraSecurityPrefix+" · ")
				cloned.TagDisplay = strings.TrimPrefix(cloned.TagDisplay, config.ChainUnblockerPrefix+" · ") + " → " + hopType
			}
		}
		group.Items[i] = cloned
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
	h.MakeSureContextIsNew(stream.Context())

	ctx := h.Context()
	if ctx == nil || h.UrlTestHistory() == nil {
		return E.New("hiddify service not found")
	}
	monitor := monitoring.Get(ctx)
	if monitor == nil {
		return E.New("outbound monitoring not ready")
	}

	if err := stream.Send(h.GetAllProxiesInfo(monitor.OutboundsHistory(""), onlyMain)); err != nil {
		return err
	}

	urltestch, err := h.subscribeAllProxies(stream.Context(), monitor)
	if err != nil {
		return err
	}
	defer monitor.UnsubscribeGroup("", urltestch)

	// Debounce URL-test / traffic bursts; selection events (GroupEvent.Priority) flush at once.
	const debounceWindow = 1000 * time.Millisecond
	var (
		timer   *time.Timer
		timerCh <-chan time.Time
	)
	stopTimer := func() {
		if timer == nil {
			return
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
	defer stopTimer()

	send := func() error {
		return stream.Send(h.GetAllProxiesInfo(monitor.OutboundsHistory(""), onlyMain))
	}

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-ctx.Done():
			return nil
		case ev, ok := <-urltestch:
			if !ok {
				return nil
			}
			priority := ev.Priority
			for drain := true; drain; {
				select {
				case ev2, ok2 := <-urltestch:
					if !ok2 {
						return nil
					}
					if ev2.Priority {
						priority = true
					}
				default:
					drain = false
				}
			}
			if priority {
				stopTimer()
				if err := send(); err != nil {
					Log(LogLevel_ERROR, LogType_CORE, "failed to send outbounds info: ", err)
					return err
				}
				continue
			}
			if timer == nil {
				timer = time.NewTimer(debounceWindow)
				timerCh = timer.C
			}
		case <-timerCh:
			stopTimer()
			if err := send(); err != nil {
				Log(LogLevel_ERROR, LogType_CORE, "failed to send outbounds info: ", err)
				return err
			}
		}
	}
}

// subscribeAllProxies waits until the catch-all monitoring group exists (box start race) then
// subscribes. A failed Subscribe used to leave a nil channel and block the stream forever.
func (h *HiddifyInstance) subscribeAllProxies(streamCtx context.Context, monitor *monitoring.OutboundMonitoring) (<-chan monitoring.GroupEvent, error) {
	for attempt := 0; attempt < 20; attempt++ {
		urltestch, err := monitor.SubscribeGroup("")
		if err == nil {
			return urltestch, nil
		}
		select {
		case <-streamCtx.Done():
			return nil, streamCtx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	return nil, E.New("outbound monitoring group not ready")
}
