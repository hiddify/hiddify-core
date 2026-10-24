package admin_service_vpn

import (
	"github.com/sagernet/sing-box/option"

	ex "github.com/hiddify/hiddify-core/extension"
)

// AdminServiceExtension is kept registered for older configs but no longer starts the
// TUN-to-SOCKS bridge. Windows and Linux VPN mode now run the full core inside
// HiddifyTunnelService (same model as macOS), driven from the app over authenticated gRPC.
// The old bridge looked like a local traffic interceptor and raised antivirus false positives.
type AdminServiceExtensionData struct {
	Count int `json:"count"`
}

var _ ex.Extension = (*AdminServiceExtension)(nil)

type AdminServiceExtension struct {
	ex.Base[AdminServiceExtensionData]
}

func (b *AdminServiceExtension) OnMainServicePreStart(singconfig *option.Options) error {
	return nil
}

func (b *AdminServiceExtension) OnMainServiceStart() error {
	return nil
}

func (b *AdminServiceExtension) OnMainServiceClose() error {
	return nil
}

func NewAdminServiceExtension() ex.Extension {
	return &AdminServiceExtension{}
}

func init() {
	ex.RegisterExtension(
		ex.ExtensionFactory{
			Id:          "github.com/hiddify/hiddify-core/extension/system/admin_service_vpn",
			Title:       "Admin Service",
			Description: "Deprecated: VPN runs in HiddifyTunnelService",
			Builder:     NewAdminServiceExtension,
		},
	)
}
