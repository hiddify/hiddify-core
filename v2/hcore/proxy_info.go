package hcore

import (
	"strings"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/monitoring"
	G "github.com/sagernet/sing-box/protocol/group"
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
	out.Type = detour.Type()
	if group, isGroup := detour.(adapter.OutboundGroup); isGroup {
		out.IsGroup = true
		gnow := group.Now()
		out.GroupSelectedTag = &gnow
	}
	out.TagDisplay = TrimTagName(out.Tag)

	if tag := monitoring.RealTag(detour); tag != "" {
		dtag := TrimTagName(tag)
		out.GroupSelectedTagDisplay = &dtag
	}
	// realTag = adapter.OutboundTag(detour)

	// realTag = out.Tag

	// url_test_history := historyStorage.LoadURLTestHistory(realTag)

	if url_test_history != nil {
		out.UrlTestTime = timestamppb.New(url_test_history.Time)
		out.UrlTestDelay = int32(url_test_history.Delay)
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
	for _, it := range box.Outbound().Outbounds() {
		his, _ := hismap[it.Tag()]
		outbounds_converted[it.Tag()] = h.GetProxyInfo(his, it)
		if group, isGroup := it.(adapter.OutboundGroup); isGroup {
			iGroups = append(iGroups, group)
			// h.Box().Logger().Info("Outbound group found: ", group.Tag(), outbounds_converted[it.Tag()], fmt.Sprint("his", his))
		}
	}
	for _, it := range box.Endpoint().Endpoints() {
		his, _ := hismap[it.Tag()]
		outbounds_converted[it.Tag()] = h.GetProxyInfo(his, it)
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

	if ctx, urlTestHistory := h.Context(), h.UrlTestHistory(); ctx != nil && urlTestHistory != nil {
		monitor := monitoring.Get(h.Context())
		observer, err := monitor.GroupObserver(config.OutboundSelectTag)
		if err != nil {
			return err
		}
		urltestch, done, err := observer.Subscribe()
		defer observer.UnSubscribe(urltestch)
		if err != nil {
			return err
		}
		stream.Send(h.GetAllProxiesInfo(monitor.OutboundsHistory(config.OutboundSelectTag), onlyMain))
		debouncer := NewDebouncer(500 * time.Millisecond)
		defer debouncer.Stop()

		for {
			select {
			case <-urltestch:

				debouncer.Hit()

			case <-debouncer.C():
				if err := stream.Send(h.GetAllProxiesInfo(monitor.OutboundsHistory(config.OutboundSelectTag), onlyMain)); err != nil {
					return err
				}

			case <-stream.Context().Done():
				return nil
			case <-ctx.Done():
				return nil
			case <-done:
				return nil
			}
		}
	}
	return nil
}
