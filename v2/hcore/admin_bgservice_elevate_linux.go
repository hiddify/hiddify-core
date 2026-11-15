//go:build linux && !android

package hcore

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// lookPath finds a program in PATH; a variable so tests can take programs away.
var lookPath = exec.LookPath

// askpassDialogs show a password dialog and print the password, as sudo -A wants. sudo passes the
// prompt as the first argument; zenity has no place for it, so its title says what it is for.
var askpassDialogs = []struct {
	program string
	args    string
}{
	{"zenity", `--password --title="Hiddify: administrator password for VPN mode"`},
	{"kdialog", `--title=Hiddify --password="Administrator password for VPN mode"`},
	{"qarma", `--password --title="Hiddify: administrator password for VPN mode"`},
	{"ksshaskpass", `"$@"`},
	{"lxqt-openssh-askpass", `"$@"`},
	{"ssh-askpass", `"$@"`},
	{"/usr/libexec/openssh/gnome-ssh-askpass", `"$@"`},
	{"/usr/lib/openssh/gnome-ssh-askpass", `"$@"`},
	{"/usr/lib/ssh/x11-ssh-askpass", `"$@"`},
}

// elevateAdminBgInstall asks for the administrator password and waits for install to finish: through
// pkexec (PolicyKit), or, on a desktop without it, through sudo with a password dialog, passwordless
// sudo, or sudo in a terminal window.
func elevateAdminBgInstall(cliPath, owner string) error {
	if appimage := os.Getenv("APPIMAGE"); appimage != "" {
		return fmt.Errorf("VPN mode in the AppImage needs root; quit and run: sudo %s", appimage)
	}
	args := []string{cliPath, "admin_bgservice", "install", owner}
	if _, err := lookPath("pkexec"); err == nil {
		err := runElevated(exec.Command("pkexec", args...), owner, pkexecRefused)
		if !pkexecHasNoAgent(err) {
			return err
		}
		// no polkit agent to show the prompt (started from a terminal or over SSH, or a desktop
		// without one): sudo's ways of asking, where there are any, else what to do instead
		if err := elevateWithSudo(args, owner); !errors.Is(err, errNoAdminPrompt) {
			return err
		}
		return noAdminAgentError(cliPath, owner)
	}
	return elevateWithSudo(args, owner)
}

// elevateWithSudo runs args as root through sudo: with a password dialog, without a password, or in
// a terminal window. errNoAdminPrompt when none of them can ask.
func elevateWithSudo(args []string, owner string) error {
	if _, err := lookPath("sudo"); err != nil {
		return errNoAdminPrompt
	}
	askpass, cleanup, err := sudoAskpass()
	if err != nil {
		return err
	}
	if askpass == "" {
		// no dialog to type the password into: sudo without a password, else sudo in a terminal
		if runElevated(exec.Command("sudo", append([]string{"-n"}, args...)...), owner, nil) == nil {
			return nil
		}
		return elevateInTerminal(args)
	}
	defer cleanup()
	cmd := exec.Command("sudo", append([]string{"-A"}, args...)...)
	cmd.Env = append(os.Environ(), "SUDO_ASKPASS="+askpass)
	return runElevated(cmd, owner, sudoRefused)
}

// sudoAskpass returns a program for SUDO_ASKPASS: the user's own, or a small script around the first
// password dialog found. It returns "" when there is none.
func sudoAskpass() (path string, cleanup func(), err error) {
	cleanup = func() {}
	if own := os.Getenv("SUDO_ASKPASS"); own != "" {
		if _, err := os.Stat(own); err == nil {
			return own, cleanup, nil
		}
	}
	// a dialog needs a desktop to show on; without one it fails, and sudo takes that for a refusal
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return "", cleanup, nil
	}
	for _, dialog := range askpassDialogs {
		program, err := lookPath(dialog.program)
		if err != nil {
			continue
		}
		dir, err := os.MkdirTemp("", "hiddify-askpass-")
		if err != nil {
			return "", cleanup, fmt.Errorf("could not ask for the administrator password: %w", err)
		}
		path = filepath.Join(dir, "askpass")
		script := fmt.Sprintf("#!/bin/sh\nexec '%s' %s\n", strings.ReplaceAll(program, "'", `'\''`), dialog.args)
		if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
			_ = os.RemoveAll(dir)
			return "", cleanup, fmt.Errorf("could not ask for the administrator password: %w", err)
		}
		return path, func() { _ = os.RemoveAll(dir) }, nil
	}
	return "", cleanup, nil
}

// pkexec exits 126 when the prompt was dismissed, and 127 both when authorization failed and when
// the program could not run (a missing library, hiddify-app#2461): its stderr tells them apart.
func pkexecRefused(code int, _ string) bool { return code == 126 }

// pkexecHasNoAgent: pkexec found no polkit agent and could not ask on a terminal instead either
// ("Error creating textual authentication agent: ... /dev/tty"), so nobody was asked (exit 127).
func pkexecHasNoAgent(err error) bool {
	return err != nil && strings.Contains(err.Error(), "textual authentication agent")
}

// noAdminAgentError says how to install the VPN service when nothing can ask for the password.
func noAdminAgentError(cliPath, owner string) error {
	return fmt.Errorf("%w: no authentication agent is running, as when Hiddify is started from a terminal or over SSH. "+
		"Start Hiddify from the applications menu, or run once: sudo %s admin_bgservice install %s",
		errNoAdminAgent, shellWord(cliPath), shellWord(owner))
}

// shellWord quotes s for a shell only where it needs it, so the command stays easy to read.
func shellWord(s string) string {
	if s != "" && !strings.ContainsAny(s, " \t'\"$`\\;&|<>()*?[]{}~!#") {
		return s
	}
	return shellQuote(s)
}

// sudo exits 1 both when the dialog was cancelled or the password was wrong, and when the program
// failed; only its own messages tell them apart.
func sudoRefused(code int, stderr string) bool {
	if code != 1 {
		return false
	}
	for _, refused := range []string{"no password was provided", "incorrect password", "a password is required"} {
		if strings.Contains(stderr, refused) {
			return true
		}
	}
	return false
}

func runElevated(cmd *exec.Cmd, owner string, refused func(code int, stderr string) bool) error {
	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
	if owner != "" {
		if cmd.Env == nil {
			cmd.Env = os.Environ()
		}
		cmd.Env = append(cmd.Env, "HIDDIFY_TUNNEL_OWNER="+owner)
	}
	err := cmd.Run()
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if refused != nil && errors.As(err, &exitErr) && refused(exitErr.ExitCode(), stderr.String()) {
		return errAdminPromptRefused
	}
	if detail := strings.TrimSpace(stderr.String()); detail != "" {
		return fmt.Errorf("could not install the VPN service (%w): %s", err, detail)
	}
	return fmt.Errorf("could not install the VPN service (%w)", err)
}
