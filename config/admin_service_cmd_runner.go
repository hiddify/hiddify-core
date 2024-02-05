//go:build !windows

package config

import (
	"fmt"
	"os"
	"os/exec"
)

func ExecuteCmd(executablePath, args string) (string, error) {
	if err := execCmdImp([]string{"gksu", executablePath, args}); err == nil {
		return "Ok", nil
	}
	if err := execCmdImp([]string{"pkexec", executablePath, args}); err == nil {
		return "Ok", nil
	}
	if err := execCmdImp([]string{"/bin/sh", "-c", "sudo " + executablePath + " " + args}); err == nil {
		return "Ok", nil
	}
	return "", fmt.Errorf("Error executing run as root shell command")

}

func execCmdImp(commands []string) error {
	cmd := exec.Command(commands[0], commands[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("Running command: %v", commands)
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		return err
	}
	return nil
}
