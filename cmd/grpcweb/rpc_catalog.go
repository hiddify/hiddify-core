package grpcweb

import (
	"sync"
	"time"
)

// Core gRPC methods exposed by this harness (web ↔ grpcweb). Packet stats count
// HTTP/SSE payloads for each method — not VPN datapath traffic inside the core.
type rpcMeta struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // unary | server_stream
	Desc string `json:"desc"`
	Path string `json:"path"` // harness HTTP path
}

var coreRPCCatalog = []rpcMeta{
	{Name: "Start", Kind: "unary", Desc: "Start the core", Path: "/api/start"},
	{Name: "StartService", Kind: "unary", Desc: "Start service (same as Start)", Path: "/api/rpc"},
	{Name: "Stop", Kind: "unary", Desc: "Stop the core", Path: "/api/stop"},
	{Name: "Restart", Kind: "unary", Desc: "Restart the core", Path: "/api/restart"},
	{Name: "HotReload", Kind: "unary", Desc: "Hot reload config", Path: "/api/hotreload"},
	{Name: "CoreInfoListener", Kind: "server_stream", Desc: "Core state stream", Path: "/api/stream/core"},
	{Name: "OutboundsInfo", Kind: "server_stream", Desc: "All outbound groups", Path: "/api/stream/proxies"},
	{Name: "MainOutboundsInfo", Kind: "server_stream", Desc: "Main outbound groups", Path: "/api/stream/main"},
	{Name: "GetSystemInfo", Kind: "unary", Desc: "One-shot system info", Path: "/api/rpc"},
	{Name: "GetSystemInfoStream", Kind: "server_stream", Desc: "Live system info", Path: "/api/stream/system"},
	{Name: "Setup", Kind: "unary", Desc: "Setup working dirs / mode", Path: "/api/rpc"},
	{Name: "Parse", Kind: "unary", Desc: "Parse share links / config", Path: "/api/resolve"},
	{Name: "ChangeHiddifySettings", Kind: "unary", Desc: "Apply Hiddify options", Path: "/api/settings"},
	{Name: "SelectOutbound", Kind: "unary", Desc: "Select proxy in a group", Path: "/api/select"},
	{Name: "UrlTest", Kind: "unary", Desc: "URL-test a tag", Path: "/api/urltest"},
	{Name: "UrlTestActive", Kind: "unary", Desc: "URL-test active outbound", Path: "/api/urltest-active"},
	{Name: "GenerateWarpConfig", Kind: "unary", Desc: "Generate WARP account/config", Path: "/api/rpc"},
	{Name: "GetSystemProxyStatus", Kind: "unary", Desc: "System proxy available/enabled", Path: "/api/rpc"},
	{Name: "SetSystemProxyEnabled", Kind: "unary", Desc: "Enable/disable system proxy", Path: "/api/rpc"},
	{Name: "LogListener", Kind: "server_stream", Desc: "Core log stream", Path: "/api/stream/logs"},
	{Name: "Close", Kind: "unary", Desc: "Close / tear down mode", Path: "/api/rpc"},
	{Name: "GetLANIP", Kind: "unary", Desc: "LAN IP address", Path: "/api/rpc"},
	{Name: "NeedAdminBgService", Kind: "unary", Desc: "Whether admin_bgservice is required", Path: "/api/rpc"},
	{Name: "EnsureAdminBgService", Kind: "server_stream", Desc: "Install/start admin_bgservice (Windows/Linux)", Path: "/api/stream/ensure-admin"},
}

const rpcMinuteSlots = 60 // one slot per second for the last minute

type rpcSecBucket struct {
	packets int64
	bytes   int64
}

// rpcMeter counts web↔grpcweb payloads for one Core RPC.
type rpcMeter struct {
	mu            sync.Mutex
	totalPackets  int64
	totalBytes    int64
	lastSize      int
	winPackets    int64
	winBytes      int64
	winStart      time.Time
	packetsPerSec float64
	bytesPerSec   float64
	// rolling last-minute history (slot = unix second % 60)
	minute      [rpcMinuteSlots]rpcSecBucket
	minuteEpoch [rpcMinuteSlots]int64 // unix second stored in that slot; 0 = empty
}

func (m *rpcMeter) note(size int) {
	if size < 0 {
		size = 0
	}
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.winStart.IsZero() {
		m.winStart = now
	}
	m.totalPackets++
	m.totalBytes += int64(size)
	m.lastSize = size
	m.winPackets++
	m.winBytes += int64(size)
	m.addMinuteLocked(now, 1, int64(size))
	if elapsed := now.Sub(m.winStart); elapsed >= time.Second {
		sec := elapsed.Seconds()
		if sec > 0 {
			m.packetsPerSec = float64(m.winPackets) / sec
			m.bytesPerSec = float64(m.winBytes) / sec
		}
		m.winPackets = 0
		m.winBytes = 0
		m.winStart = now
	}
}

func (m *rpcMeter) addMinuteLocked(now time.Time, packets, bytes int64) {
	sec := now.Unix()
	i := int(sec % rpcMinuteSlots)
	if m.minuteEpoch[i] != sec {
		m.minute[i] = rpcSecBucket{}
		m.minuteEpoch[i] = sec
	}
	m.minute[i].packets += packets
	m.minute[i].bytes += bytes
}

func (m *rpcMeter) minuteTotalsLocked(now time.Time) (packets, bytes int64) {
	cutoff := now.Unix() - int64(rpcMinuteSlots) + 1
	for i := 0; i < rpcMinuteSlots; i++ {
		if m.minuteEpoch[i] < cutoff {
			continue
		}
		packets += m.minute[i].packets
		bytes += m.minute[i].bytes
	}
	return packets, bytes
}

func (m *rpcMeter) snapshot(name string) map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	pps, bps := m.packetsPerSec, m.bytesPerSec
	if !m.winStart.IsZero() {
		elapsed := now.Sub(m.winStart)
		if elapsed >= 200*time.Millisecond && elapsed < time.Second {
			sec := elapsed.Seconds()
			if sec > 0 {
				pps = float64(m.winPackets) / sec
				bps = float64(m.winBytes) / sec
			}
		} else if elapsed >= 2*time.Second && m.winPackets == 0 {
			pps, bps = 0, 0
		}
	}
	minPkts, minBytes := m.minuteTotalsLocked(now)
	return map[string]any{
		"name":            name,
		"packets":         m.totalPackets,
		"bytes":           m.totalBytes,
		"lastSize":        m.lastSize,
		"packetsPerSec":   pps,
		"bytesPerSec":     bps,
		"packetsLastMin":  minPkts,
		"bytesLastMin":    minBytes,
	}
}

type rpcStatsHub struct {
	mu     sync.Mutex
	meters map[string]*rpcMeter
}

func newRPCStatsHub() *rpcStatsHub {
	h := &rpcStatsHub{meters: make(map[string]*rpcMeter, len(coreRPCCatalog))}
	for _, m := range coreRPCCatalog {
		h.meters[m.Name] = &rpcMeter{}
	}
	return h
}

func (h *rpcStatsHub) note(method string, size int) {
	h.mu.Lock()
	m := h.meters[method]
	if m == nil {
		m = &rpcMeter{}
		h.meters[method] = m
	}
	h.mu.Unlock()
	m.note(size)
}

func (h *rpcStatsHub) snapshot() map[string]any {
	h.mu.Lock()
	defer h.mu.Unlock()
	methods := make([]map[string]any, 0, len(coreRPCCatalog))
	var totalPkts, totalBytes, totalMinPkts, totalMinBytes int64
	var totalPPS, totalBPS float64
	for _, meta := range coreRPCCatalog {
		m := h.meters[meta.Name]
		if m == nil {
			m = &rpcMeter{}
		}
		snap := m.snapshot(meta.Name)
		snap["kind"] = meta.Kind
		snap["desc"] = meta.Desc
		snap["path"] = meta.Path
		methods = append(methods, snap)
		totalPkts += snap["packets"].(int64)
		totalBytes += snap["bytes"].(int64)
		totalPPS += snap["packetsPerSec"].(float64)
		totalBPS += snap["bytesPerSec"].(float64)
		totalMinPkts += snap["packetsLastMin"].(int64)
		totalMinBytes += snap["bytesLastMin"].(int64)
	}
	return map[string]any{
		"total": map[string]any{
			"packets":         totalPkts,
			"bytes":           totalBytes,
			"packetsPerSec":   totalPPS,
			"bytesPerSec":     totalBPS,
			"packetsLastMin":  totalMinPkts,
			"bytesLastMin":    totalMinBytes,
		},
		"methods": methods,
		"catalog": coreRPCCatalog,
	}
}
