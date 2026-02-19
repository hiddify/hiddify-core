//go:build darwin

package hutils

import "github.com/sagernet/sing-box/experimental/libbox"

func RedirectStderr(path string) error {
	return libbox.RedirectStderr(path)
}
