//go:build !windows

package hcore

import "os/exec"

func prepareCliCommand(*exec.Cmd) {}

func explainCliError(_ string, err error) error { return err }
