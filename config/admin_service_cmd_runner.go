//go:build !windows

package config

import (
	"fmt"
	"os"
	"os/exec"
)

func ExecuteCmd(executablePath, args string) (string, error) {
	err := execCmdImp([]string{"gksu", executablePath, args})
	if err == nil {
		return "Ok", nil
	}
	err := execCmdImp([]string{"pkexec", executablePath, args})
	if err == nil {
		return "Ok", nil
	}
	err := execCmdImp([]string{"/bin/sh", "-c", "sudo " + executablePath + " " + args})
	if err == nil {
		return "Ok", nil
	}
	return "", err

}

func execCmdImp(cmd []string) error {
	cmd := exec.Command(cmd[0], cmd[1:])
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("Running command: %v", cmd.String())
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		return err
	}
	return nil
}
