package config

import (
	"context"
	"fmt"
	"testing"

	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
	"github.com/stretchr/testify/require"
)

func parseOptions(t *testing.T, ctx context.Context, content string) *option.Options {
	t.Helper()
	options, err := json.UnmarshalExtendedContext[option.Options](ctx, []byte(content))
	require.NoError(t, err)
	return &options
}

// previousItems are the proxy items of a profile as last built, what StableProxyTags is given.
func previousItems(t *testing.T, ctx context.Context, options *option.Options) []ProxyItem {
	t.Helper()
	items, _, err := StableProxyTags(ctx, options, nil)
	require.NoError(t, err)
	return items
}

func itemTags(items []ProxyItem) []string {
	tags := make([]string, 0, len(items))
	for _, item := range items {
		tags = append(tags, item.Tag)
	}
	return tags
}

// ss is a shadowsocks outbound of a profile, as JSON; ssVia dials through detour.
func ss(tag, server string, port int) string {
	return fmt.Sprintf(`{"type": "shadowsocks", "tag": %q, "server": %q, "server_port": %d, "method": "aes-128-gcm", "password": "p"}`, tag, server, port)
}

func ssVia(tag, server string, port int, detour string) string {
	return ss(tag, server, port)[:len(ss(tag, server, port))-1] + fmt.Sprintf(`, "detour": %q}`, detour)
}

func tagsOf(options *option.Options) []string {
	var tags []string
	for _, out := range options.Outbounds {
		tags = append(tags, out.Tag)
	}
	return tags
}

// A subscription update that adds a link renumbers the links after it. The proxies that did not
// change keep the tags they had, so the hot reload keeps them and their pings, and the references
// to them follow.
func TestStableProxyTagsKeepTheOldTagsOfUnchangedProxies(t *testing.T) {
	ctx := libbox.BaseContext(nil)
	previous := parseOptions(t, ctx, `{"outbounds": [`+ss("a § 0", "a.example.com", 443)+", "+ss("b § 1", "b.example.com", 443)+", "+ss("c § 2", "c.example.com", 443)+`, {"type": "selector", "tag": "g", "outbounds": ["a § 0", "b § 1", "c § 2"], "default": "b § 1"}]}`)
	// n is new and first, so every old link moved one up; c changed its port
	input := parseOptions(t, ctx, `{"outbounds": [`+ss("n § 0", "n.example.com", 443)+", "+ss("a § 1", "a.example.com", 443)+", "+ss("b § 2", "b.example.com", 443)+", "+ss("c § 3", "c.example.com", 8443)+`, {"type": "selector", "tag": "g", "outbounds": ["n § 0", "a § 1", "b § 2", "c § 3"], "default": "b § 2"}], "route": {"rules": [{"domain": ["x.example.com"], "outbound": "a § 1"}], "final": "b § 2"}}`)
	items, renamed, err := StableProxyTags(ctx, input, previousItems(t, ctx, previous))
	require.NoError(t, err)
	require.Equal(t, 2, renamed, "a and b take their old tags; n and the changed c keep theirs")
	require.Equal(t, []string{"n § 0", "a § 0", "b § 1", "c § 3"}, itemTags(items), "the items for the next time carry the tags as they are now")
	require.Equal(t, []string{"n § 0", "a § 0", "b § 1", "c § 3", "g"}, tagsOf(input))
	selector := input.Outbounds[4].Options.(*option.SelectorOutboundOptions)
	require.Equal(t, []string{"n § 0", "a § 0", "b § 1", "c § 3"}, selector.Outbounds)
	require.Equal(t, "b § 1", selector.Default)
	require.Equal(t, "a § 0", input.Route.Rules[0].DefaultOptions.RouteOptions.Outbound)
	require.Equal(t, "b § 1", input.Route.Final)

	// nothing changed: nothing renamed
	again := parseOptions(t, ctx, `{"outbounds": [`+ss("a § 0", "a.example.com", 443)+`]}`)
	_, renamed, err = StableProxyTags(ctx, again, previousItems(t, ctx, parseOptions(t, ctx, `{"outbounds": [`+ss("a § 0", "a.example.com", 443)+`]}`)))
	require.NoError(t, err)
	require.Equal(t, 0, renamed)
}

// A chain's links dial through the previous link: the detour is not part of the fingerprint, and
// it is renamed with the link it names.
func TestStableProxyTagsFollowTheDetourOfAChain(t *testing.T) {
	ctx := libbox.BaseContext(nil)
	previous := parseOptions(t, ctx, `{"outbounds": [`+ss("first § 0", "1.example.com", 443)+", "+ssVia("second § 0", "2.example.com", 443, "first § 0")+`]}`)
	input := parseOptions(t, ctx, `{"outbounds": [`+ss("new § 0", "0.example.com", 443)+", "+ss("first § 1", "1.example.com", 443)+", "+ssVia("second § 1", "2.example.com", 443, "first § 1")+`]}`)
	_, _, err := StableProxyTags(ctx, input, previousItems(t, ctx, previous))
	require.NoError(t, err)
	require.Equal(t, []string{"new § 0", "first § 0", "second § 0"}, tagsOf(input))
	require.Equal(t, "first § 0", input.Outbounds[2].Options.(*option.ShadowsocksOutboundOptions).DialerOptions.Detour)
}

// A new proxy that got the tag an unchanged one had takes a free number instead.
func TestStableProxyTagsGiveANewProxyAFreeNumber(t *testing.T) {
	ctx := libbox.BaseContext(nil)
	previous := parseOptions(t, ctx, `{"outbounds": [`+ss("x § 0", "a.example.com", 443)+`]}`)
	input := parseOptions(t, ctx, `{"outbounds": [`+ss("x § 0", "new.example.com", 443)+", "+ss("x § 1", "a.example.com", 443)+`]}`)
	_, renamed, err := StableProxyTags(ctx, input, previousItems(t, ctx, previous))
	require.NoError(t, err)
	require.Equal(t, 2, renamed)
	require.Equal(t, []string{"x § 2", "x § 0"}, tagsOf(input))
}
