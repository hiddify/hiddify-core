package hcore

import (
	"context"
	"fmt"
	"sync"

	"github.com/sagernet/sing-box/option"

	"github.com/hiddify/hiddify-core/v2/config"
)

// The profile's proxies keep their tags across its updates (see config.StableProxyTags), so a
// hot reload after a subscription update keeps the unchanged proxies, their pings and keep top's
// pick, and tests only what changed.
var lastProfile struct {
	sync.Mutex
	path  string
	items []config.ProxyItem // the profile's proxies as last built, with their stable tags
}

// stableProfileTags renames input's proxies to the tags they had the last time this profile was
// built, when their options are the same, and remembers them for the next time. It returns the
// profile's proxies with their tags and fingerprints; nil when they could not be read, and then
// keep top picks anew.
func stableProfileTags(ctx context.Context, path string, input *option.Options) []config.ProxyItem {
	lastProfile.Lock()
	defer lastProfile.Unlock()
	var previous []config.ProxyItem
	if lastProfile.path == path {
		previous = lastProfile.items
	}
	items, renamed, err := config.StableProxyTags(ctx, input, previous)
	if err != nil {
		Log(LogLevel_WARNING, LogType_CORE, "stable tags: ", err.Error())
		return nil
	}
	if renamed > 0 {
		Log(LogLevel_INFO, LogType_CORE, fmt.Sprintf("stable tags: %d proxies keep the tags they had", renamed))
	}
	lastProfile.path, lastProfile.items = path, items
	return items
}
