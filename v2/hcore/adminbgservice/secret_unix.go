//go:build unix

package adminbgservice

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
)

func chownSecret(path, owner string) error {
	uid, err := lookupOwnerUID(owner)
	if err != nil {
		return err
	}
	return os.Chown(path, uid, -1)
}

func lookupOwnerUID(owner string) (int, error) {
	if uid, err := strconv.Atoi(owner); err == nil {
		return uid, nil
	}
	account, err := user.Lookup(owner)
	if err != nil {
		return -1, fmt.Errorf("tunnel service owner %q: %w", owner, err)
	}
	uid, err := strconv.Atoi(account.Uid)
	if err != nil {
		return -1, err
	}
	return uid, nil
}
