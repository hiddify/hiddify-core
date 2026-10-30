package config

import (
	"slices"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

// Go's own TLS ClientHello is easy to tell from a browser's, so a TLS client that names neither a fingerprint
// (utls) nor an engine gets one of these, as Xray uses Chrome's fingerprint when none is set:
//   - the system's TLS (engine "apple" on macOS and iOS, "windows" on Windows 10 1809 or later), whose
//     ClientHello is the system's own, when the config uses nothing the system TLS lacks: disable_sni,
//     cipher_suites, curve_preferences, a client certificate, ECH, TLS fragment or record fragment, spoof, kTLS,
//     TLS tricks, or the dialer's TLS fragment or finalmask, which all shape the ClientHello themselves;
//   - otherwise uTLS with Chrome's fingerprint.
//
// Only TLS over a TCP stream is changed (VLESS, VMess, Trojan, AnyTLS, HTTP): QUIC (HTTP/3, the quic transport,
// Hysteria, TUIC) needs Go's TLS. "fingerprint": "unsafe" keeps Go's TLS, as in Xray, and REALITY has its own.

var tlsEngineProtocols = map[string]bool{
	C.TypeVLESS: true, C.TypeVMess: true, C.TypeTrojan: true, C.TypeAnyTLS: true, C.TypeHTTP: true,
}

// chooseTLSEngines applies the rules above to the outbounds, with the engine this system has ("" for none).
func chooseTLSEngines(outbounds []option.Outbound, systemEngine string) {
	for _, outbound := range outbounds {
		if !tlsEngineProtocols[outbound.Type] {
			continue
		}
		wrapper, hasTLS := outbound.Options.(option.OutboundTLSOptionsWrapper)
		if !hasTLS {
			continue
		}
		var dialer option.DialerOptions
		if dialerWrapper, hasDialer := outbound.Options.(option.DialerOptionsWrapper); hasDialer {
			dialer = dialerWrapper.TakeDialerOptions()
		}
		transport := outboundTransport(outbound)
		if transport != nil && transport.Type == C.V2RayTransportTypeQUIC {
			continue
		}
		chooseTLSEngine(wrapper.TakeOutboundTLSOptions(), dialer, systemEngine)
		if transport != nil && transport.Type == C.V2RayTransportTypeXHTTP && transport.XHTTPOptions.Download != nil {
			chooseTLSEngine(transport.XHTTPOptions.Download.TakeOutboundTLSOptions(), dialer, systemEngine)
		}
	}
}

func chooseTLSEngine(tls *option.OutboundTLSOptions, dialer option.DialerOptions, systemEngine string) {
	if tls == nil || !tls.Enabled || tls.Engine != "" || (tls.UTLS != nil && tls.UTLS.Enabled) ||
		(tls.Reality != nil && tls.Reality.Enabled) || slices.Contains(tls.ALPN, "h3") {
		return
	}
	if systemEngine != "" && systemTLSCan(tls, dialer) {
		tls.Engine = systemEngine
		return
	}
	tls.UTLS = &option.OutboundUTLSOptions{Enabled: true, Fingerprint: "chrome"}
}

// systemTLSCan reports whether the system's TLS can make this connection: sing-box refuses these fields for it
// (ValidateSystemTLSOptions), and the TLS tricks, the TLS fragment and finalmask change Go's or uTLS's
// ClientHello, which the system's TLS does not give out.
func systemTLSCan(tls *option.OutboundTLSOptions, dialer option.DialerOptions) bool {
	switch {
	case tls.DisableSNI, len(tls.CipherSuites) > 0, len(tls.CurvePreferences) > 0,
		len(tls.ClientCertificate) > 0, tls.ClientCertificatePath != "", len(tls.ClientKey) > 0, tls.ClientKeyPath != "",
		tls.Fragment, tls.RecordFragment, tls.KernelTx, tls.KernelRx, tls.Spoof != "", tls.SpoofMethod != "",
		tls.ECH != nil && tls.ECH.Enabled, tls.TLSFragment != nil && tls.TLSFragment.Enabled:
		return false
	case tls.TLSTricks != nil && (tls.TLSTricks.MixedCaseSNI || tls.TLSTricks.PaddingMode != "" || tls.TLSTricks.PaddingSNI != ""):
		return false
	case dialer.TLSFragment.Enabled, dialer.FinalMask != nil && (len(dialer.FinalMask.TCP) > 0 || len(dialer.FinalMask.UDP) > 0):
		return false
	}
	return true
}

func outboundTransport(outbound option.Outbound) *option.V2RayTransportOptions {
	switch options := outbound.Options.(type) {
	case *option.VLESSOutboundOptions:
		return options.Transport
	case *option.VMessOutboundOptions:
		return options.Transport
	case *option.TrojanOutboundOptions:
		return options.Transport
	}
	return nil
}
