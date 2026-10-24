//go:build !unix

package tunnelservice

func chownSecret(path, owner string) error {
	return nil
}
