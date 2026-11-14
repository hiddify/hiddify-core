package hcore

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"filippo.io/age"
	"github.com/sagernet/sing-box/experimental/libbox"
	E "github.com/sagernet/sing/common/exceptions"
)

// diagnosticsRecipient is the age recipient of diagnosticsPublicKey, nil when the build has none.
// A key that is not an age key is an error: no report is made rather than one in the clear.
func diagnosticsRecipient() (age.Recipient, error) {
	key := strings.TrimSpace(diagnosticsPublicKey)
	if key == "" {
		return nil, nil
	}
	recipient, err := age.ParseX25519Recipient(key)
	if err != nil {
		return nil, E.Cause(err, "the diagnostics public key of this build is not an age key")
	}
	return recipient, nil
}

// encryptDirectory zips directory, scrubbing its text as every report is (libbox.CreateZipArchive),
// encrypts the zip to recipient into destination, and removes directory: only the encrypted copy
// is left to go in the report.
func encryptDirectory(directory, destination string, recipient age.Recipient) error {
	plain := destination + ".zip"
	defer os.Remove(plain)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return E.Cause(err, "create the sensitive directory")
	}
	if err := libbox.CreateZipArchive(directory, plain, false); err != nil {
		return E.Cause(err, "zip the sensitive files")
	}
	if err := encryptFile(plain, destination, recipient); err != nil {
		os.Remove(destination)
		return E.Cause(err, "encrypt the sensitive files")
	}
	return os.RemoveAll(directory)
}

func encryptFile(source, destination string, recipient age.Recipient) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()
	writer, err := age.Encrypt(out, recipient)
	if err != nil {
		return err
	}
	if _, err := io.Copy(writer, in); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return out.Close()
}

// listFiles is every file under directory, relative to it and with forward slashes, sorted.
func listFiles(directory string) []string {
	var files []string
	filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			if relative, err := filepath.Rel(directory, path); err == nil {
				files = append(files, filepath.ToSlash(relative))
			}
		}
		return nil
	})
	slices.Sort(files)
	return files
}

// reportReadme tells whoever opens the report what it holds and what of it is encrypted.
func reportReadme(sensitiveFiles []string, encrypted bool) string {
	var list strings.Builder
	for _, name := range sensitiveFiles {
		list.WriteString("    " + name + "\n")
	}
	var b strings.Builder
	b.WriteString(`Hiddify diagnostics report
==========================

metadata.json
    Not encrypted. Generic information anyone with the report can read: the app version and
    build, the system, the connection state, the settings without passwords or keys, and the
    list of crash reports. No config, server, address or log.

goroutines/, crash_reports/ and app/
    Not encrypted. Stack traces: the previous run's goroutines, the core's crashes and the app's
    own exceptions. They show which code ran, not your config or servers.

`)
	if encrypted {
		b.WriteString(encryptedSensitive + `
    Encrypted on the device, to the Hiddify team's key, in the age format
    (https://age-encryption.org). Only the Hiddify team can open it. It is a zip of:

` + list.String() + `
    The Hiddify team opens it with:
        age -d -i hiddify-diagnostics.key ` + encryptedSensitive + ` > sensitive.zip

    To look inside it yourself, save the report again in the app: Report a problem,
    Save the file, Unencrypted.
`)
	} else {
		b.WriteString(sensitiveDirectory + `/
    NOT encrypted. The config with servers and keys removed, and the logs:

` + list.String() + `
    Addresses, links and secrets are removed from these files, but they can still tell which
    apps and sites you used. Send this file only to people you trust.
`)
	}
	return b.String()
}
