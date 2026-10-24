//go:build !unix && !windows

package tunnelservice

func chownSecret(path, owner string) error {
	return nil
}
