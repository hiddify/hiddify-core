//go:build !windows

package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func ExecuteCmd(executablePath, args string, background bool) (string, error) {
	cwd := filepath.Dir(executablePath)
	if appimage := os.Getenv("APPIMAGE"); appimage != "" {
		executablePath = appimage + " HiddifyService"
		if !background {
			return nil, fmt.Errorf("Appimage can not have service")
		}
	}

	if err := execCmdImp([]string{"cocoasudo", "--prompt=Hiddify needs root for tunneling.", executablePath, args}, cwd, background); err == nil {
		return "Ok", nil
	}
	if err := execCmdImp([]string{"gksu", executablePath, args}, cwd, background); err == nil {
		return "Ok", nil
	}
	if err := execCmdImp([]string{"pkexec", executablePath, args}, cwd, background); err == nil {
		return "Ok", nil
	}
	if err := execCmdImp([]string{"xterm", "-e", "sudo " + executablePath + " " + args}, cwd, background); err == nil {
		return "Ok", nil
	}

	if err := execCmdImp([]string{"sudo", executablePath, args}, cwd, background); err == nil {
		return "Ok", nil
	}

	return "", fmt.Errorf("Error executing run as root shell command")

}

func execCmdImp(commands []string, cwd string, background bool) error {
	cmd := exec.Command(commands[0], commands[1:]...)
	cmd.Dir = cwd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("Running command: %v", commands)
	if background {
		if err := cmd.Start(); err != nil {
			fmt.Printf("Error: %v\n", err)
			return err
		}
	} else {
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error: %v\n", err)
			return err
		}
	}
	return nil
}
