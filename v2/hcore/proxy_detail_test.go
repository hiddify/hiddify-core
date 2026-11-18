package hcore

import (
	"context"
	"net"
	"os"
	"testing"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	M "github.com/sagernet/sing/common/metadata"
	"github.com/stretchr/testify/require"
)

// The app shows a proxy's transport and security after its protocol, from the options of the
// config the engine runs; a proxy without them, or not in that config, keeps its display type.
func TestDisplayTypeWithDetail(t *testing.T) {
	t.Cleanup(func() { runningProxyDetails.Store(nil) })
	vless := &fakeProxy{outbound.NewAdapter(C.TypeVLESS, "v", nil, nil)}
	ss := &fakeProxy{outbound.NewAdapter(C.TypeShadowsocks, "s", nil, nil)}

	require.Equal(t, "VLESS", displayTypeWithDetail(vless), "before a config runs")
	rememberProxyDetails(&option.Options{Outbounds: []option.Outbound{
		{Type: C.TypeVLESS, Tag: "v", Options: &option.VLESSOutboundOptions{
			OutboundTLSOptionsContainer: option.OutboundTLSOptionsContainer{TLS: &option.OutboundTLSOptions{
				Enabled: true, Reality: &option.OutboundRealityOptions{Enabled: true},
			}},
			Transport: &option.V2RayTransportOptions{Type: C.V2RayTransportTypeXHTTP},
		}},
		{Type: C.TypeShadowsocks, Tag: "s", Options: &option.ShadowsocksOutboundOptions{}},
	}})
	require.Equal(t, "VLESS · XHTTP · REALITY", displayTypeWithDetail(vless))
	require.Equal(t, "Shadowsocks", displayTypeWithDetail(ss))
}

type fakeProxy struct{ outbound.Adapter }

func (*fakeProxy) DialContext(context.Context, string, M.Socksaddr) (net.Conn, error) {
	return nil, os.ErrInvalid
}

func (*fakeProxy) ListenPacket(context.Context, M.Socksaddr) (net.PacketConn, error) {
	return nil, os.ErrInvalid
}

var _ adapter.Outbound = (*fakeProxy)(nil)
