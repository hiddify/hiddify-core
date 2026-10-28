//go:build linux && !android

package hutils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sagernet/sing-box/experimental/libbox"
	"golang.org/x/sys/unix"
)

func RedirectStderr(path string) error {
	return libbox.RedirectStderr(path)
}

func IsAdmin() bool {
	return os.Getuid() == 0
}

func TunAllowed() bool {
	var hdr unix.CapUserHeader
	hdr.Version = unix.LINUX_CAPABILITY_VERSION_3
	hdr.Pid = 0

	// LINUX_CAPABILITY_VERSION_3 makes the kernel write two CapUserData, for caps 0-31 and 32-63
	var data [2]unix.CapUserData
	if err := unix.Capget(&hdr, &data[0]); err != nil {
		fmt.Print(err)
		return false
	}
	return (data[0].Effective & (1 << unix.CAP_NET_ADMIN)) != 0
}

// ExecuteCmd elevates once to install or start the tunnel service. Prefer pkexec (PolicyKit),
// which desktop environments and security tools recognise; avoid cocoasudo/gksu/xterm wrappers
// that look like unexpected privilege escalation.
func ExecuteCmd(executablePath string, background bool, args ...string) (string, error) {
	cwd := filepath.Dir(executablePath)
	if appimage := os.Getenv("APPIMAGE"); appimage != "" {
		executablePath = appimage
		if !background {
			return "Fail", fmt.Errorf("appimage cannot have service")
		}
	}

	commands := [][]string{
		append([]string{"pkexec", executablePath}, args...),
		append([]string{"sudo", "-n", executablePath}, args...),
		append([]string{"sudo", executablePath}, args...),
	}

	var err error
	for _, command := range commands {
		cmd := exec.Command(command[0], command[1:]...)
		cmd.Dir = cwd
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if owner := os.Getenv("USER"); owner != "" {
			cmd.Env = append(os.Environ(), "HIDDIFY_TUNNEL_OWNER="+owner)
		}
		fmt.Printf("Running command: %v\n", command)
		if background {
			err = cmd.Start()
		} else {
			err = cmd.Run()
		}
		if err == nil {
			return "Ok", nil
		}
	}

	return "", fmt.Errorf("error elevating tunnel service install: %w", err)
}

func Chmod(path string, mode os.FileMode) error {
	return os.Chmod(path, mode)
}
