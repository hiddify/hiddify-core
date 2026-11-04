package hcore

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	sbox "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/monitoring"
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

// connectedVia is the outbound a group sends every connection through, or "" when it picks one
// per connection (a balancer other than lowest-delay), so which one is in use is not known.
func connectedVia(group adapter.OutboundGroup) string {
	if b, ok := group.(*balancer.Balancer); ok && b.Strategy() != balancer.StrategyLowestDelay {
		return ""
	}
	return groupNow(group)
}

// unreachableDelay is the delay a failed URL test records.
const unreachableDelay = monitoring.TimeoutDelay

// bestGroupMember returns the member of a group with the lowest known delay. A balancer has no
// selection before its first pick, and its group tag would otherwise be taken for the outbound
// carrying traffic.
func bestGroupMember(group adapter.OutboundGroup, history adapter.URLTestHistoryStorage) string {
	best := ""
	var bestRank uint32
	for _, tag := range group.All() {
		if rank := memberRank(history, tag); rank > 0 && (best == "" || rank < bestRank) {
			best, bestRank = tag, rank
		}
	}
	return best
}

// memberRank orders members by ping as the balancers do (balancer.Rank: a real ping that works
// before a cached one), and 0 for none or a failed one, which does not count.
func memberRank(history adapter.URLTestHistoryStorage, tag string) uint32 {
	his := memberHistory(history, tag)
	if his == nil || his.Delay == 0 || his.Delay >= unreachableDelay {
		return 0
	}
	return uint32(balancer.Rank(his))
}

func memberHistory(history adapter.URLTestHistoryStorage, tag string) *adapter.URLTestHistory {
	if history == nil {
		return nil
	}
	return history.LoadURLTestHistory(tag)
}

// spreadMember is the member a balancer that spreads connections stands for, in the app: its
// fastest by ping (memberRank: a real ping before a cached one), kept as the results come. Another
// member takes the place of one with a real ping only when faster by more than the balancers'
// tolerance, or when it stops answering; of one with a cached ping, when better at all; while a
// regular test cycle runs, only when it stops answering. Taking the fastest at every result moved
// the row between members a few milliseconds apart, so the ping the app showed jumped up and down
// with every test.
func spreadMember(group adapter.OutboundGroup, history adapter.URLTestHistoryStorage) string {
	best := bestGroupMember(group, history)
	shown.access.Lock()
	defer shown.access.Unlock()
	current := shown.members[group.Tag()]
	if current != "" && slices.Contains(group.All(), current) && keepsShown(history, current, best) {
		return current
	}
	if best != "" {
		shown.members[group.Tag()] = best
	}
	return best
}

// keepsShown says whether the member shown stays for best, the fastest now (see spreadMember).
func keepsShown(history adapter.URLTestHistoryStorage, current, best string) bool {
	rank := memberRank(history, current)
	if rank == 0 {
		return false // it stopped answering
	}
	if periodicCycleRunning() || best == "" || best == current {
		return true
	}
	// the balancers' rule: the tolerance keeps a real ping from jitter, a cached one is left for
	// any better
	threshold := balancer.SelectedThreshold(memberHistory(history, current), config.BalancerTolerance, config.BalancerToleranceRatio)
	return memberRank(history, best)+threshold >= rank
}

// shown is the member each spreading balancer stands for now (spreadMember).
var shown = struct {
	access  sync.Mutex
	members map[string]string
}{members: map[string]string{}}

// periodicCycleRunning reports a regular test cycle of the schedule in progress, during which the
// balancer holds its members (not the first cycle after start, nor one after a network change).
func periodicCycleRunning() bool {
	if static == nil {
		return false
	}
	if ctx := static.Context(); ctx != nil {
		if monitor := monitoring.Get(ctx); monitor != nil {
			return monitor.PeriodicCycleRunning()
		}
	}
	return false
}

// resolveActiveOutbound walks down the selected chain to the outbound that actually carries
// traffic, so re-testing the active proxy tests one config instead of every config in its group. A
// balancer that spreads connections (round robin, ...) uses each member in turn, so the one it
// picked last says nothing: it stands for its fastest member, by ping.
func resolveActiveOutbound(box *sbox.Box, tag string, history adapter.URLTestHistoryStorage) string {
	for depth := 0; depth < 8; depth++ {
		outbound, loaded := box.Outbound().Outbound(tag)
		if !loaded {
			return tag
		}
		group, isGroup := outbound.(adapter.OutboundGroup)
		if !isGroup {
			return tag
		}
		next := connectedVia(group)
		if next == "" {
			next = spreadMember(group, history)
		}
		if next == "" || next == tag {
			return tag
		}
		tag = next
	}
	return tag
}

func (h *HiddifyInstance) GetProxyInfo(url_test_history *adapter.URLTestHistory, detour adapter.Outbound) *OutboundInfo {
	// historyStorage := h.UrlTestHistory()
	// if historyStorage == nil {
	// 	return nil
	// }

	out := &OutboundInfo{}
	// realTag := ""

	out.Tag = detour.Tag()
	out.Type = adapter.DisplayType(detour) // the type and its details, and why it failed; the status is out.Status
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
		// a ping cached from the previous run is not shown and decides nothing; it only orders
		// the list, and the tests, until the real one arrives
		out.SortDelay = out.UrlTestDelay
		if url_test_history.IsFromCache {
			out.UrlTestDelay = 0
		} else {
			out.Error = url_test_history.Error // why its last URL test failed
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
				FromCache:   url_test_history.IpInfoFromCache,
			}
		}

	}
	// not ready, so not tested: why it is not ready says more than an earlier test did
	if err := adapter.StatusError(detour); err != nil {
		out.Error = err.Error()
	}
	if deps := detour.Dependencies(); len(deps) == 1 {
		out.Detour = deps[0]
	}
	speedTag := h.speedTestTagOf(detour)
	out.UploadSpeed, out.UploadTesting, out.UploadSpeedCached = speedInfo(upload, speedTag)
	out.DownloadSpeed, out.DownloadTesting, out.DownloadSpeedCached = speedInfo(download, speedTag)
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
		// a balancer spreading connections shows the ping of the member it stands for, its fastest,
		// as its speeds do (speedTestTagOf), not of the one it picked last
		if group, isGroup := it.(adapter.OutboundGroup); isGroup && connectedVia(group) == "" {
			if member := resolveActiveOutbound(box, it.Tag(), h.UrlTestHistory()); member != it.Tag() {
				if memberHis, ok := hismap[member]; ok && memberHis != nil {
					his = memberHis
				}
			}
		}
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
		selectedTag := groupNow(iGroup)
		// a balancer spreading connections (round robin, ...) has no one member in use: the one it
		// picks next turns with every connection and can be one whose ping failed, which the home
		// screen took for the user's pick being down. It stands for the member it ranks first, as
		// its ping and speeds do (spreadMember).
		if connectedVia(iGroup) == "" {
			if member := spreadMember(iGroup, h.UrlTestHistory()); member != "" {
				selectedTag = member
			}
		}
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
			group.Items = append(group.Items, pinfo)
			pinfo.IsVisible = !strings.Contains(itemTag, "§hide§")

		}
		if len(group.Items) == 0 {
			continue
		}

		groups.Items = append(groups.Items, &group)

	}
	if onlyGroupitems {
		groups.ChainHop = outbounds_converted[chainHop(box)] // nil with the chain off
	}

	return &groups
}

func isChainTag(tag string) bool {
	return config.IsChainExtraSecurityTag(tag) || config.IsChainUnblockerTag(tag)
}

// chainHop is the tag of the extra security or unblocker hop the traffic goes through now, or ""
// with the chain off. The selectable mode puts its hops under a selector and a lowest-delay
// balancer; then it follows their selection down to the hop.
func chainHop(box *sbox.Box) string {
	var tags []string
	members := make(map[string]bool) // what the selectable mode's selector and balancer hold
	for _, out := range box.Outbound().Outbounds() {
		if !isChainTag(out.Tag()) {
			continue
		}
		tags = append(tags, out.Tag())
		if group, isGroup := out.(adapter.OutboundGroup); isGroup {
			for _, member := range group.All() {
				members[member] = true
			}
		}
	}
	for _, endpoint := range box.Endpoint().Endpoints() {
		if isChainTag(endpoint.Tag()) {
			tags = append(tags, endpoint.Tag())
		}
	}
	for _, tag := range tags {
		if members[tag] {
			continue
		}
		for range 3 {
			out, _ := box.Outbound().Outbound(tag)
			group, isGroup := out.(adapter.OutboundGroup)
			if !isGroup {
				return tag
			}
			tag = groupNow(group)
		}
	}
	return ""
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
	// The app is reading the proxies, so it wants their pings: that is activity for the
	// monitoring, which otherwise stops testing once the core has routed nothing for its idle
	// time (three URL test intervals), and the rows it had not reached stayed waiting for good.
	monitor.Touch()
	keepActive := time.NewTicker(time.Minute)
	defer keepActive.Stop()

	if err := stream.Send(h.GetAllProxiesInfo(monitor.OutboundsHistory(""), onlyMain)); err != nil {
		return err
	}

	urltestch, err := h.subscribeAllProxies(stream.Context(), monitor)
	if err != nil {
		return err
	}
	defer monitor.UnsubscribeGroup("", urltestch)
	speedch, unsubscribeSpeedTests := subscribeSpeedTests()
	defer unsubscribeSpeedTests()

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
		case <-keepActive.C:
			monitor.Touch()
		case <-speedch:
			// a speed test began or ended: shown with the next send
			if timer == nil {
				timer = time.NewTimer(debounceWindow)
				timerCh = timer.C
			}
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
