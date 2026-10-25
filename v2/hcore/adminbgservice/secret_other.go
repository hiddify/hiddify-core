//go:build !unix && !windows

package adminbgservice

func chownSecret(path, owner string) error {
	return nil
}
