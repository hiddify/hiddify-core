package hcore

import (
	"strings"

	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/outbound"
	"github.com/sagernet/sing/service"
	"google.golang.org/grpc"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

func GetProxyInfo(detour adapter.Outbound) *OutboundInfo {
	if static.Box == nil {
		return nil
	}
	out := &OutboundInfo{
		Tag:  detour.Tag(),
		Type: detour.Type(),
	}
	url_test_history := static.Box.UrlTestHistory().LoadURLTestHistory(adapter.OutboundTag(detour))
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
		if _, isGroup := detour.(adapter.OutboundGroup); isGroup {
			out.IsGroup = true
		}

	}

	return out
}

func GetAllProxiesInfo(onlyGroupitems bool) *OutboundGroupList {
	if static.Box == nil {
		return nil
	}

	cacheFile := service.FromContext[adapter.CacheFile](static.Box.Context())
	outbounds := static.Box.GetInstance().Router().Outbounds()
	outbounds_converted := make(map[string]*OutboundInfo, 0)
	var iGroups []adapter.OutboundGroup
	for _, it := range outbounds {
		if group, isGroup := it.(adapter.OutboundGroup); isGroup {
			iGroups = append(iGroups, group)
		}

		outbounds_converted[it.Tag()] = GetProxyInfo(it)
	}

	var groups OutboundGroupList
	for _, iGroup := range iGroups {
		var group OutboundGroup
		group.Tag = iGroup.Tag()
		group.Type = iGroup.Type()
		_, group.Selectable = iGroup.(*outbound.Selector)
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
			pinfo.TagDisplay = strings.Trim(strings.Split(itemTag, "§")[0], " ")
		}
		if len(group.Items) == 0 {
			continue
		}
		groups.Items = append(groups.Items, &group)
	}

	return &groups
}

func (s *CoreService) OutboundsInfo(req *hcommon.Empty, stream grpc.ServerStreamingServer[OutboundGroupList]) error {
	return AllProxiesInfoStream(stream, false)
}

func (s *CoreService) MainOutboundsInfo(req *hcommon.Empty, stream grpc.ServerStreamingServer[OutboundGroupList]) error {
	return AllProxiesInfoStream(stream, true)
}

func AllProxiesInfoStream(stream grpc.ServerStreamingServer[OutboundGroupList], onlyMain bool) error {
	urltestch, done, err := static.Box.UrlTestHistory().Observer().Subscribe()
	defer static.Box.UrlTestHistory().Observer().UnSubscribe(urltestch)
	if err != nil {
		return err
	}
	stream.Send(GetAllProxiesInfo(onlyMain))
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
			stream.Send(GetAllProxiesInfo(onlyMain))
			// case <-time.After(500 * time.Millisecond):
		}
	}
}
