package admin_service_vpn

import (
	"github.com/sagernet/sing-box/option"

	ex "github.com/hiddify/hiddify-core/extension"
	tunnelservice "github.com/hiddify/hiddify-core/v2/hcore/tunnelservice"
	hutils "github.com/hiddify/hiddify-core/v2/hutils"
	C "github.com/sagernet/sing-box/constant"
)

type AdminServiceExtensionData struct {
	Count int `json:"count"` // Number of counts for the extension
}

var _ ex.Extension = (*AdminServiceExtension)(nil)

// AdminServiceExtension represents the core functionality of the extension
type AdminServiceExtension struct {
	ex.Base[AdminServiceExtensionData]
	tunInboundOptions *option.TunInboundOptions
	socksOptions      *option.SocksInboundOptions
}

func (b *AdminServiceExtension) OnMainServicePreStart(singconfig *option.Options) error {
	if hutils.TunAllowed() {
		return nil
	}
	newInbounds := make([]option.Inbound, 0, len(singconfig.Inbounds))

	for _, inb := range singconfig.Inbounds {
		if inb.Type == C.TypeTun {
			b.tunInboundOptions = &inb.TunOptions
		} else {
			if inb.Type == C.TypeSOCKS {
				b.socksOptions = &inb.SocksOptions
			}
			newInbounds = append(newInbounds, inb)
		}
	}

	singconfig.Inbounds = newInbounds
	return nil
}

func (b *AdminServiceExtension) OnMainServiceStart() error {
	if b.tunInboundOptions == nil || b.socksOptions == nil {
		return nil
	}
	username := ""
	password := ""
	if b.socksOptions.Users != nil && len(b.socksOptions.Users) > 0 {
		username = b.socksOptions.Users[0].Username
		password = b.socksOptions.Users[0].Password
	}
	tunnelservice.ActivateTunnelService(&tunnelservice.TunnelStartRequest{
		Ipv6:                   len(b.tunInboundOptions.Inet6Address) > 0,
		ServerPort:             int32(b.socksOptions.ListenPort),
		ServerUsername:         username,
		ServerPassword:         password,
		StrictRoute:            b.tunInboundOptions.StrictRoute,
		Stack:                  b.tunInboundOptions.Stack,
		EndpointIndependentNat: b.tunInboundOptions.EndpointIndependentNat,
	})
	return nil
}

func (b *AdminServiceExtension) OnMainServiceClose() error {
	if b.tunInboundOptions == nil || b.socksOptions == nil {
		return nil
	}
	return tunnelservice.DeactivateTunnelService()
}

func NewAdminServiceExtension() ex.Extension {
	return &AdminServiceExtension{}
}

func init() {
	ex.RegisterExtension(
		ex.ExtensionFactory{
			Id:          "github.com/hiddify/hiddify-core/extension/system/admin_service_vpn", // Package identifier
			Title:       "Admin Service",                                                      // Display title of the extension
			Description: "System Extension",                                                   // Brief description of the extension
			Builder:     NewAdminServiceExtension,                                             // Function to create a new instance
		},
	)
}
