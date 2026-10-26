package config

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	box "github.com/sagernet/sing-box"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
)

const (
	dnsStartupTimeout         = 5 * time.Second
	dnsStartupShutdownTimeout = time.Second
	dnsStartupMonitoringURL   = "%"
)

func TestBuildConfigStartsWithDirectDNSTransports(t *testing.T) {
	testCases := []struct {
		name             string
		directDNSAddress string
		enableFragment   bool
	}{
		{name: "bare address without fragment", directDNSAddress: "1.1.1.1"},
		{name: "bare address with fragment", directDNSAddress: "1.1.1.1", enableFragment: true},
		{name: "UDP address without fragment", directDNSAddress: "udp://1.1.1.1"},
		{name: "UDP address with fragment", directDNSAddress: "udp://1.1.1.1", enableFragment: true},
		{name: "HTTPS address without fragment", directDNSAddress: "https://1.1.1.1/dns-query"},
		{name: "HTTPS address with fragment", directDNSAddress: "https://1.1.1.1/dns-query", enableFragment: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			options := buildDNSStartupConfig(t, testCase.directDNSAddress, testCase.enableFragment)
			if err := startConfig(t, options); err != nil {
				t.Fatalf("start generated config: %v", err)
			}

			assertDNSDetour(t, options, DNSRemoteNoWarpTag, "")
			assertDNSDetour(t, options, DNSDirectTag, "")
			assertDNSDetour(t, options, DNSTricksDirectTag, "")
			assertNoDNSDetourTargetsEmptyDirectOutbound(t, options)
			assertTrickDNSFragment(t, options)
			assertDirectFragmentOutbound(t, options)
		})
	}
}

func TestRawDNSDetourToEmptyDirectOutboundIsRejected(t *testing.T) {
	options := buildDNSStartupConfig(t, "https://1.1.1.1/dns-query", false)
	for _, tag := range []string{DNSRemoteNoWarpTag, DNSDirectTag, DNSTricksDirectTag} {
		setDNSDetour(t, options, tag, "")
	}
	setDNSDetour(t, options, DNSDirectTag, OutboundDirectFragmentTag)

	err := startConfig(t, options)
	if err == nil {
		t.Fatal("start raw config: expected empty direct detour rejection")
	}
	if !strings.Contains(err.Error(), "detour to an empty direct outbound makes no sense") {
		t.Fatalf("start raw config: unexpected error: %v", err)
	}
}

func buildDNSStartupConfig(t *testing.T, directDNSAddress string, enableFragment bool) *option.Options {
	t.Helper()

	hiddifyOptions := DefaultHiddifyOptions()
	hiddifyOptions.DirectDnsAddress = directDNSAddress
	hiddifyOptions.TLSTricks.EnableFragment = enableFragment
	hiddifyOptions.EnableClashApi = false
	hiddifyOptions.EnableNTP = false
	hiddifyOptions.LogFile = ""

	input := &option.Options{
		Outbounds: []option.Outbound{
			{
				Type: C.TypeSOCKS,
				Tag:  "test-proxy",
				Options: &option.SOCKSOutboundOptions{
					ServerOptions: option.ServerOptions{
						Server:     "127.0.0.1",
						ServerPort: 9,
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(include.Context(context.Background()), dnsStartupTimeout)
	defer cancel()
	options, err := BuildConfig(ctx, hiddifyOptions, &ReadOptions{Options: input})
	if err != nil {
		t.Fatalf("BuildConfig: %v", err)
	}
	options.Inbounds = nil
	options.Log = &option.LogOptions{Disabled: true}
	// box.New always creates outbound monitoring. An invalid URL makes URLTest
	// return from url.Parse before it can call any outbound dialer.
	options.Experimental = &option.ExperimentalOptions{
		Monitoring: &option.MonitoringOptions{
			URLs: []string{dnsStartupMonitoringURL},
		},
	}
	return options
}

func startConfig(t *testing.T, options *option.Options) error {
	t.Helper()
	assertIsolatedDNSStartupRuntimeSurface(t, options)

	ctx, cancel := context.WithTimeout(include.Context(context.Background()), dnsStartupTimeout)
	instance, err := box.New(box.Options{
		Context: ctx,
		Options: *options,
	})
	if err != nil {
		cancel()
		t.Fatalf("box.New: %v", err)
	}

	startResult := make(chan error, 1)
	go func() {
		startResult <- instance.Start()
	}()

	select {
	case err = <-startResult:
		closeErr := instance.Close()
		cancel()
		if closeErr != nil && !(err != nil && errors.Is(closeErr, os.ErrClosed)) {
			t.Errorf("box.Close after box.Start returned: %v", closeErr)
		}
		return err
	case <-ctx.Done():
		timeoutErr := ctx.Err()
		cancel()
		shutdownTimer := time.NewTimer(dnsStartupShutdownTimeout)
		defer shutdownTimer.Stop()
		select {
		case startErr := <-startResult:
			closeErr := instance.Close()
			return fmt.Errorf("box.Start timed out after %s: %w (box.Close: %v; box.Start returned after shutdown: %v)", dnsStartupTimeout, timeoutErr, closeErr, startErr)
		case <-shutdownTimer.C:
			return fmt.Errorf("box.Start timed out after %s: %w (box.Start did not exit within %s; box.Close was not called to avoid concurrent close)", dnsStartupTimeout, timeoutErr, dnsStartupShutdownTimeout)
		}
	}
}

func assertIsolatedDNSStartupRuntimeSurface(t *testing.T, options *option.Options) {
	t.Helper()
	if len(options.Inbounds) != 0 {
		t.Fatalf("DNS startup test config has %d inbounds", len(options.Inbounds))
	}
	if len(options.Services) != 0 {
		t.Fatalf("DNS startup test config has %d additional services", len(options.Services))
	}
	if len(options.Endpoints) != 0 {
		t.Fatalf("DNS startup test config has %d endpoints", len(options.Endpoints))
	}
	if options.Route != nil && len(options.Route.RuleSet) != 0 {
		t.Fatalf("DNS startup test config has %d route rule sets", len(options.Route.RuleSet))
	}
	if options.NTP != nil && options.NTP.Enabled {
		t.Fatal("DNS startup test config has NTP enabled")
	}
	if options.Log == nil || !options.Log.Disabled || options.Log.Output != "" {
		t.Fatalf("DNS startup test log options are not disabled and output-free: %+v", options.Log)
	}
	if options.Experimental == nil || options.Experimental.Monitoring == nil {
		t.Fatal("DNS startup test config has no isolated monitoring options")
	}
	if options.Experimental.CacheFile != nil || options.Experimental.ClashAPI != nil || options.Experimental.V2RayAPI != nil || options.Experimental.Debug != nil {
		t.Fatalf("DNS startup test config has non-monitoring experimental services: %+v", options.Experimental)
	}
	monitoringURLs := options.Experimental.Monitoring.URLs
	if len(monitoringURLs) != 1 || monitoringURLs[0] != dnsStartupMonitoringURL {
		t.Fatalf("DNS startup test monitoring URLs = %q, want [%q]", monitoringURLs, dnsStartupMonitoringURL)
	}
	if _, err := url.Parse(monitoringURLs[0]); err == nil {
		t.Fatalf("DNS startup test monitoring URL %q can reach an outbound dialer", monitoringURLs[0])
	}
}

func assertDNSDetour(t *testing.T, options *option.Options, tag string, expected string) {
	t.Helper()
	actual := dnsDialerOptions(t, dnsServerByTag(t, options, tag)).Detour
	if actual != expected {
		t.Errorf("DNS server %q detour = %q, want %q", tag, actual, expected)
	}
}

func assertNoDNSDetourTargetsEmptyDirectOutbound(t *testing.T, options *option.Options) {
	t.Helper()

	emptyDirectTags := make(map[string]struct{})
	for _, outbound := range options.Outbounds {
		if outbound.Type != C.TypeDirect {
			continue
		}
		dialerOptions, ok := outbound.Options.(option.DialerOptionsWrapper)
		if !ok {
			t.Fatalf("direct outbound %q options %T do not expose dialer options", outbound.Tag, outbound.Options)
		}
		if reflect.DeepEqual(dialerOptions.TakeDialerOptions(), option.DialerOptions{}) {
			emptyDirectTags[outbound.Tag] = struct{}{}
		}
	}
	for _, tag := range []string{OutboundDirectTag, OutboundDirectFragmentTag} {
		if _, found := emptyDirectTags[tag]; !found {
			t.Errorf("direct outbound %q is not present and semantically empty", tag)
		}
	}

	if options.DNS == nil {
		t.Fatal("generated config has no DNS options")
	}
	for _, server := range options.DNS.Servers {
		dialerOptions, ok := server.Options.(option.DialerOptionsWrapper)
		if !ok {
			continue
		}
		detour := dialerOptions.TakeDialerOptions().Detour
		if _, targetsEmptyDirect := emptyDirectTags[detour]; targetsEmptyDirect {
			t.Errorf("DNS server %q detours through semantically empty direct outbound %q", server.Tag, detour)
		}
	}
}

func setDNSDetour(t *testing.T, options *option.Options, tag string, detour string) {
	t.Helper()
	server := dnsServerByTag(t, options, tag)
	dialerOptions := dnsDialerOptions(t, server)
	dialerOptions.Detour = detour
	server.Options.(option.DialerOptionsWrapper).ReplaceDialerOptions(dialerOptions)
}

func dnsDialerOptions(t *testing.T, server *option.DNSServerOptions) option.DialerOptions {
	t.Helper()
	dialerOptions, ok := server.Options.(option.DialerOptionsWrapper)
	if !ok {
		t.Fatalf("DNS server %q options %T do not expose dialer options", server.Tag, server.Options)
	}
	return dialerOptions.TakeDialerOptions()
}

func dnsServerByTag(t *testing.T, options *option.Options, tag string) *option.DNSServerOptions {
	t.Helper()
	if options.DNS == nil {
		t.Fatal("generated config has no DNS options")
	}
	for index := range options.DNS.Servers {
		if options.DNS.Servers[index].Tag == tag {
			return &options.DNS.Servers[index]
		}
	}
	t.Fatalf("generated config has no DNS server tagged %q", tag)
	return nil
}

func assertTrickDNSFragment(t *testing.T, options *option.Options) {
	t.Helper()
	server := dnsServerByTag(t, options, DNSTricksDirectTag)
	httpsOptions, ok := server.Options.(*option.RemoteHTTPSDNSServerOptions)
	if !ok {
		t.Fatalf("DNS server %q options = %T, want *option.RemoteHTTPSDNSServerOptions", server.Tag, server.Options)
	}
	if httpsOptions.TLS == nil || !httpsOptions.TLS.Enabled || !httpsOptions.TLS.Fragment || !httpsOptions.TLS.RecordFragment {
		t.Fatalf("DNS server %q lost TLS fragmentation: %+v", server.Tag, httpsOptions.TLS)
	}
	if actual := time.Duration(httpsOptions.TLS.FragmentFallbackDelay); actual != 300*time.Millisecond {
		t.Errorf("DNS server %q fragment fallback delay = %s, want 300ms", server.Tag, actual)
	}
}

func assertDirectFragmentOutbound(t *testing.T, options *option.Options) {
	t.Helper()
	for _, outbound := range options.Outbounds {
		if outbound.Tag == OutboundDirectFragmentTag {
			if outbound.Type != C.TypeDirect {
				t.Errorf("outbound %q type = %q, want %q", outbound.Tag, outbound.Type, C.TypeDirect)
			}
			return
		}
	}
	t.Fatalf("generated config has no outbound tagged %q", OutboundDirectFragmentTag)
}
