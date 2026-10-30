package config

import (
	"strconv"
	"strings"
)

// Some config generators repeat names: two proxies "X", one of them dialing through "X" (its own
// name, meaning the other one), and two groups "G" listing "X" once each. sing-box and Clash refuse
// repeated names, so the repeats are numbered ("X § 2", "X § 3", shown as "X" like the
// numbered tags of share links) and the references follow them:
//   - a detour to the item's own name is the item before it with that name (or after it, for the
//     first one);
//   - in a group's list, the k-th mention of a name is the k-th item with that name, and a name the
//     list mentions once is the item with the group's own number (the second "G" lists the second
//     "X").
//
// Any other reference is the first item with that name, which keeps its name.
type duplicateNames struct {
	renamed map[string][]string // a repeated name → the names of its items, in order
	ordinal []int               // the item's number among the items with its name, from 0
	names   []string            // the original names, in order
}

// numberDuplicates numbers the repeated names among names, which are the names of all items that
// references can point to, in order. It returns the new names.
func numberDuplicates(names []string) ([]string, *duplicateNames) {
	d := &duplicateNames{renamed: map[string][]string{}, ordinal: make([]int, len(names)), names: names}
	count := map[string]int{}
	for _, name := range names {
		count[name]++
	}
	seen := map[string]int{}
	result := make([]string, len(names))
	for i, name := range names {
		n := seen[name]
		seen[name]++
		d.ordinal[i] = n
		result[i] = name
		if name == "" || count[name] < 2 {
			continue
		}
		if n > 0 {
			result[i] = d.freeName(name, n+1, count)
		}
		d.renamed[name] = append(d.renamed[name], result[i])
	}
	return result, d
}

func (d *duplicateNames) freeName(name string, n int, taken map[string]int) string {
	for {
		candidate := strings.TrimSpace(name) + " § " + strconv.Itoa(n)
		if taken[candidate] == 0 {
			taken[candidate] = 1
			return candidate
		}
		n++
	}
}

func (d *duplicateNames) has() bool { return len(d.renamed) > 0 }

// detour is the reference ref of item i, a detour.
func (d *duplicateNames) detour(i int, ref string) string {
	items, repeated := d.renamed[ref]
	if !repeated {
		return ref
	}
	if ref == d.names[i] {
		if d.ordinal[i] > 0 {
			return items[d.ordinal[i]-1]
		}
		return items[1]
	}
	return items[0]
}

// members is the list of group item i.
func (d *duplicateNames) members(i int, list []string) []string {
	mentions := map[string]int{}
	for _, ref := range list {
		mentions[ref]++
	}
	seen := map[string]int{}
	result := make([]string, len(list))
	for k, ref := range list {
		result[k] = ref
		items, repeated := d.renamed[ref]
		if !repeated {
			continue
		}
		n := d.ordinal[i]
		if mentions[ref] > 1 {
			n = seen[ref]
			seen[ref]++
		}
		result[k] = items[min(n, len(items)-1)]
	}
	return result
}
