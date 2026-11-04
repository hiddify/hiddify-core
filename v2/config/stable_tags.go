package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/sagernet/sing-box/option"
)

// Stable tags across profile updates.
//
// The share links of a profile are numbered by their position ("name § 59"). A subscription
// update that adds, removes or reorders a link renumbers every link after it, so every proxy
// looked changed: the hot reload replaced them all, their pings went back to cached, and the
// whole list was tested again, which also made keep top pick anew. StableProxyTags gives a proxy
// of the new profile the tag it had in the previous one when its options are the same (the tag
// and the detour aside, the detour being the previous link of a chain, renamed with it), so the
// hot reload keeps its instance and its ping. A new proxy whose tag an old one took gets a free
// number. References follow the renaming: detours, group members, route and selector targets.

// StableProxyTags renames the profile's proxies in input to the tags they had in previous (the
// ProxyItems of the profile as last built, nil the first time), when their options are the same.
// It returns input's proxies with their tags as they are now, for the next time, and how many
// were renamed.
func StableProxyTags(ctx context.Context, input *option.Options, previous []ProxyItem) ([]ProxyItem, int, error) {
	if input == nil {
		return nil, 0, nil
	}
	raw, err := input.MarshalJSONContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	var tree map[string]any
	if err := json.Unmarshal(raw, &tree); err != nil {
		return nil, 0, err
	}
	items := ProxyItems(tree)
	rename := stableRenaming(previous, items)
	if len(rename) == 0 {
		return items, 0, nil
	}
	renameReferences(tree, rename)
	renamed, err := json.Marshal(tree)
	if err != nil {
		return nil, 0, err
	}
	var options option.Options
	if err := options.UnmarshalJSONContext(ctx, renamed); err != nil {
		return nil, 0, err
	}
	*input = options
	for i := range items {
		if to, ok := rename[items[i].Tag]; ok {
			items[i].Tag = to
		}
	}
	return items, len(rename), nil
}

// ProxyItem is a proxy of a profile: its tag and its fingerprint, a hash of its options but the
// tag and the detour. Keep top keeps its pick by the same fingerprint.
type ProxyItem struct {
	Tag         string
	Fingerprint string
}

// Fingerprints are the items by tag, as keep top keeps them.
func Fingerprints(items []ProxyItem) map[string]string {
	fingerprints := make(map[string]string, len(items))
	for _, item := range items {
		fingerprints[item.Tag] = item.Fingerprint
	}
	return fingerprints
}

// ProxyItems are the profile's proxies in tree, the profile's options as a JSON tree.
func ProxyItems(tree map[string]any) []ProxyItem {
	var items []ProxyItem
	for _, key := range []string{"outbounds", "endpoints"} {
		list, _ := tree[key].([]any)
		for _, entry := range list {
			object, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			tag, _ := object["tag"].(string)
			typ, _ := object["type"].(string)
			if tag == "" || !isUserProxy(tag, typ) {
				continue
			}
			copied := make(map[string]any, len(object))
			for k, v := range object {
				if k != "tag" && k != "detour" {
					copied[k] = v
				}
			}
			content, err := json.Marshal(copied) // encoding/json sorts the keys
			if err != nil {
				continue
			}
			sum := sha256.Sum256(content)
			items = append(items, ProxyItem{Tag: tag, Fingerprint: hex.EncodeToString(sum[:])})
		}
	}
	return items
}

// stableRenaming maps the new tags to the old ones of the same proxies, in order of appearance
// for repeated ones, and a new proxy whose tag an old one takes to a free one.
func stableRenaming(oldItems, newItems []ProxyItem) map[string]string {
	oldByFingerprint := make(map[string][]string)
	for _, item := range oldItems {
		oldByFingerprint[item.Fingerprint] = append(oldByFingerprint[item.Fingerprint], item.Tag)
	}
	rename := make(map[string]string)
	taken := make(map[string]bool)
	var unmatched []string
	for _, item := range newItems {
		if tags := oldByFingerprint[item.Fingerprint]; len(tags) > 0 {
			oldTag := tags[0]
			oldByFingerprint[item.Fingerprint] = tags[1:]
			taken[oldTag] = true
			if oldTag != item.Tag {
				rename[item.Tag] = oldTag
			}
			continue
		}
		unmatched = append(unmatched, item.Tag)
	}
	if len(rename) == 0 {
		return rename
	}
	// a new proxy keeps its tag unless a matched one took it
	highest := highestTagNumber(oldItems, newItems)
	for _, tag := range unmatched {
		if !taken[tag] {
			taken[tag] = true
			continue
		}
		base, _, _ := splitNumberedTag(tag)
		candidate := freeName(base, highest+1, func(name string) bool { return taken[name] })
		taken[candidate] = true
		rename[tag] = candidate
		_, highest, _ = splitNumberedTag(candidate)
	}
	return rename
}

// splitNumberedTag splits a numbered tag ("x § 12") into its name and number; numbered is false
// for a plain tag, whose name is the tag.
func splitNumberedTag(tag string) (name string, n int, numbered bool) {
	parts := strings.SplitN(tag, " § ", 2)
	if len(parts) < 2 {
		return tag, 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	return parts[0], n, err == nil
}

// highestTagNumber is the highest number of the numbered tags in lists.
func highestTagNumber(lists ...[]ProxyItem) int {
	highest := 0
	for _, items := range lists {
		for _, item := range items {
			if _, n, numbered := splitNumberedTag(item.Tag); numbered && n > highest {
				highest = n
			}
		}
	}
	return highest
}

// referenceKeys hold a tag: an outbound's detour, a group's members and default, a route
// rule's or the DNS's outbound, the route's final.
var referenceKeys = map[string]bool{"detour": true, "outbounds": true, "default": true, "outbound": true, "final": true}

// renameReferences renames the tags of the outbounds and endpoints, and every reference to them,
// in the whole tree. A renaming is applied at once, so a swap of two tags is right.
func renameReferences(tree map[string]any, rename map[string]string) {
	for _, key := range []string{"outbounds", "endpoints"} {
		list, _ := tree[key].([]any)
		for _, entry := range list {
			if object, ok := entry.(map[string]any); ok {
				if tag, ok := object["tag"].(string); ok {
					if to, renamed := rename[tag]; renamed {
						object["tag"] = to
					}
				}
			}
		}
	}
	walkJSON(tree, func(object map[string]any, key string, field any) bool {
		if !referenceKeys[key] {
			return true
		}
		if tag, ok := field.(string); ok {
			if to, renamed := rename[tag]; renamed {
				object[key] = to
			}
			return false
		}
		return !renameMembers(field, rename) // the top-level outbounds list, of objects, is walked
	})
}

// renameMembers renames the tags of a list of them in place (a group's members) and reports
// whether field was one; the top-level outbounds list, of objects, is not.
func renameMembers(field any, rename map[string]string) bool {
	list, ok := field.([]any)
	if !ok {
		return false
	}
	tags := false
	for i, item := range list {
		if tag, ok := item.(string); ok {
			tags = true
			if to, renamed := rename[tag]; renamed {
				list[i] = to
			}
		}
	}
	return tags
}
