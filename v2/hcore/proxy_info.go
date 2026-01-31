package hcore

import (
	"strings"
	"time"

	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/sagernet/sing-box/adapter"
	G "github.com/sagernet/sing-box/protocol/group"
	"github.com/sagernet/sing/service"
	"google.golang.org/grpc"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

func (h *HiddifyInstance) GetProxyInfo(detour adapter.Outbound, endpoint adapter.Endpoint) *OutboundInfo {
	historyStorage := h.UrlTestHistory()
	if historyStorage == nil {
		return nil
	}
	out := &OutboundInfo{}
	realTag := ""
	if detour != nil {
		out.Tag = detour.Tag()
		out.Type = detour.Type()
		if _, isGroup := detour.(adapter.OutboundGroup); isGroup {
			out.IsGroup = true
		}
		realTag = adapter.OutboundTag(detour)
	} else if endpoint != nil {
		out.Tag = endpoint.Tag()
		out.Type = endpoint.Type()
		realTag = out.Tag
	}
	url_test_history := historyStorage.LoadURLTestHistory(realTag)
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

func (h *HiddifyInstance) GetAllProxiesInfo(onlyGroupitems bool) *OutboundGroupList {
	instance := h.Instance()
	if instance == nil {
		return nil
	}
	box := h.Box()
	if box == nil {
		return nil
	}
	cacheFile := service.FromContext[adapter.CacheFile](instance.Context())

	outbounds_converted := make(map[string]*OutboundInfo, 0)
	var iGroups []adapter.OutboundGroup
	for _, it := range box.Outbound().Outbounds() {
		if group, isGroup := it.(adapter.OutboundGroup); isGroup {
			iGroups = append(iGroups, group)
		}

		outbounds_converted[it.Tag()] = h.GetProxyInfo(it, nil)
	}
	for _, it := range box.Endpoint().Endpoints() {
		outbounds_converted[it.Tag()] = h.GetProxyInfo(nil, it)
	}

	var groups OutboundGroupList
	for _, iGroup := range iGroups {
		var group OutboundGroup
		group.Tag = iGroup.Tag()
		group.Type = iGroup.Type()
		_, group.Selectable = iGroup.(*G.Selector)
		selectedTag := iGroup.Now()
		group.Selected = outbounds_converted[selectedTag]
		outbounds_converted[iGroup.Tag()].GroupSelectedOutbound = group.Selected
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
			pinfo.TagDisplay = TrimTagName(itemTag)
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
	if urlTestHistory := h.UrlTestHistory(); urlTestHistory != nil {
		urltestch, done, err := urlTestHistory.Observer().Subscribe()
		defer urlTestHistory.Observer().UnSubscribe(urltestch)
		if err != nil {
			return err
		}
		stream.Send(h.GetAllProxiesInfo(onlyMain))
		for {
			select {
			case <-stream.Context().Done():
				return nil
			case <-done:
				return nil
			case <-urltestch:
			debounce:
				for {
					select {
					case <-urltestch:
					default:
						break debounce
					}
				}
				stream.Send(h.GetAllProxiesInfo(onlyMain))
			case <-time.After(500 * time.Millisecond):
			}
		}
	}
	return nil
}
