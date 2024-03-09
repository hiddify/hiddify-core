//go:build !windows

package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ExecuteCmd(executablePath string, background bool, args ...string) (string, error) {
	cwd := filepath.Dir(executablePath)
	if appimage := os.Getenv("APPIMAGE"); appimage != "" {
		executablePath = appimage
		if !background {
			return "Fail", fmt.Errorf("Appimage cannot have service")
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
