package constants

import "runtime/debug"

var Version = "unknown"

// commit is set at link time, 7 characters as git abbreviates it (the Makefile does):
//
//	-X github.com/hiddify/hiddify-core/v2/hcommon/constants.commit=<hash>
var commit = ""

// Commit is the hiddify-core commit this binary was built from: the one set at link time, else
// the one Go stamps when it builds inside the repository, else "unknown". A "-dirty" suffix
// says the tree had uncommitted changes.
func Commit() string {
	if commit != "" {
		return commit
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	revision, modified := "", false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	if revision == "" {
		return "unknown"
	}
	if len(revision) > 7 {
		revision = revision[:7]
	}
	if modified {
		revision += "-dirty"
	}
	return revision
}
