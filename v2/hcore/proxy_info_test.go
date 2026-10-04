package hcore

import (
	"testing"

	"github.com/hiddify/hiddify-core/v2/config"
)

func TestShowEntryHopExtraSecurity(t *testing.T) {
	hop := &OutboundInfo{Tag: config.ChainExtraSecurityTag, TagDisplay: TrimTagName(config.ChainExtraSecurityTag), UrlTestDelay: 321}
	proxy := &OutboundInfo{Tag: "proxy-a", TagDisplay: "proxy-a", UrlTestDelay: 50}
	group := OutboundGroup{Tag: config.OutboundSelectTag, Selected: "proxy-a", Items: []*OutboundInfo{proxy}}

	showEntryHop(&group, map[string]*OutboundInfo{"proxy-a": proxy, config.ChainExtraSecurityTag: hop}, func(string) string { return "Psiphon" })

	if group.Selected != config.ChainExtraSecurityTag || group.Items[0] != hop {
		t.Fatalf("extra security hop must be the active entry, got selected %q first %q", group.Selected, group.Items[0].Tag)
	}
	if hop.TagDisplay != "Psiphon → proxy-a" {
		t.Fatalf("unexpected display %q", hop.TagDisplay)
	}
	if group.Items[0].UrlTestDelay != 321 {
		t.Fatal("the hop's own delay must be shown")
	}
}

func TestShowEntryHopWithoutHop(t *testing.T) {
	proxy := &OutboundInfo{Tag: "proxy-a", TagDisplay: "proxy-a"}
	group := OutboundGroup{Tag: config.OutboundSelectTag, Selected: "proxy-a", Items: []*OutboundInfo{proxy}}
	showEntryHop(&group, map[string]*OutboundInfo{"proxy-a": proxy}, func(string) string { return "Psiphon" })
	if group.Selected != "proxy-a" || len(group.Items) != 1 {
		t.Fatal("without an entry hop the selected proxy stays active")
	}
}
