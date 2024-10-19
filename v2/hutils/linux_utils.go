//go:build linux && !android

package hutils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

func IsAdmin() bool {
	return os.Getuid() == 0
}

func TunAllowed() bool {
	var hdr unix.CapUserHeader
	hdr.Version = unix.LINUX_CAPABILITY_VERSION_3
	hdr.Pid = 0 // 0 means current process

	var data unix.CapUserData
	if err := unix.Capget(&hdr, &data); err != nil {
		fmt.Print(err)
		return false //, fmt.Errorf("failed to get capabilities: %v", err)
	}
	return (data.Effective & (1 << unix.CAP_NET_ADMIN)) != 0
}

func ExecuteCmd(executablePath string, background bool, args ...string) (string, error) {
	cwd := filepath.Dir(executablePath)
	if appimage := os.Getenv("APPIMAGE"); appimage != "" {
		executablePath = appimage
		if !background {
			return "Fail", fmt.Errorf("appimage cannot have service")
		}
	}

	commands := [][]string{
		{"cocoasudo", "--prompt=Hiddify needs root for tunneling.", executablePath},
		{"gksu", executablePath},
		{"pkexec", executablePath},
		{"xterm", "-e", "sudo", executablePath, strings.Join(args, " ")},
		{"sudo", executablePath},
	}

	var err error
	var cmd *exec.Cmd
	for _, command := range commands {
		cmd = exec.Command(command[0], command[1:]...)
		cmd.Dir = cwd
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
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

	return "", fmt.Errorf("Error executing run as root shell command")
}
