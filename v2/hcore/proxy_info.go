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
	if Box == nil {
		return nil
	}
	out := &OutboundInfo{
		Tag:  detour.Tag(),
		Type: detour.Type(),
	}
	url_test_history := Box.UrlTestHistory().LoadURLTestHistory(adapter.OutboundTag(detour))
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

func GetAllProxiesInfo(onlyGroupitems bool) *OutboundGroupList {
	if Box == nil {
		return nil
	}

	cacheFile := service.FromContext[adapter.CacheFile](Box.Context())
	outbounds := Box.GetInstance().Router().Outbounds()
	var iGroups []adapter.OutboundGroup
	for _, it := range outbounds {
		if group, isGroup := it.(adapter.OutboundGroup); isGroup {
			iGroups = append(iGroups, group)
		}
	}
	var groups OutboundGroupList
	for _, iGroup := range iGroups {
		var group OutboundGroup
		group.Tag = iGroup.Tag()
		group.Type = iGroup.Type()
		_, group.Selectable = iGroup.(*outbound.Selector)
		// group.Selected = iGroup.Now()
		selectedTag := iGroup.Now()
		if cacheFile != nil {
			if isExpand, loaded := cacheFile.LoadGroupExpand(group.Tag); loaded {
				group.IsExpand = isExpand
			}
		}

		for _, itemTag := range iGroup.All() {
			itemOutbound, isLoaded := Box.GetInstance().Router().Outbound(itemTag)
			if !isLoaded {
				continue
			}
			if onlyGroupitems && itemTag != selectedTag {
				continue
			}
			pinfo := GetProxyInfo(itemOutbound)
			pinfo.IsSelected = itemTag == selectedTag
			if pinfo.IsSelected {
				group.Selected = pinfo
			}
			group.Items = append(group.Items, pinfo)
			pinfo.IsVisible = !strings.Contains(itemTag, "§hide§")
			pinfo.TagDisplay = strings.Split(itemTag, "§")[0]
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
	urltestch, done, err := Box.UrlTestHistory().Observer().Subscribe()
	defer Box.UrlTestHistory().Observer().UnSubscribe(urltestch)
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
