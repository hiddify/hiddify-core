//go:build linux && !android

package hcore

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// terminalPromptTimeout: how long the terminal may wait for the password.
var terminalPromptTimeout = 5 * time.Minute

// terminals that run a program given after their arguments, in the order tried after $TERMINAL and the
// desktop's own. The program is one script path, so terminals that take a command string work too.
var terminals = []struct {
	program string
	args    []string
	desktop string // XDG_CURRENT_DESKTOP that ships it
}{
	{"ptyxis", []string{"--new-window", "--"}, "GNOME"},
	{"kgx", []string{"--"}, "GNOME"},
	{"gnome-terminal", []string{"--"}, "GNOME"},
	{"konsole", []string{"-e"}, "KDE"},
	{"xfce4-terminal", []string{"-x"}, "XFCE"},
	{"mate-terminal", []string{"-x"}, "MATE"},
	{"qterminal", []string{"-e"}, "LXQT"},
	{"lxterminal", []string{"-e"}, "LXDE"},
	{"cosmic-term", []string{"-e"}, "COSMIC"},
	{"x-terminal-emulator", []string{"-e"}, ""},
	{"tilix", []string{"-e"}, ""},
	{"terminator", []string{"-x"}, ""},
	{"kitty", nil, ""},
	{"alacritty", []string{"-e"}, ""},
	{"foot", nil, ""},
	{"wezterm", []string{"start", "--"}, ""},
	{"urxvt", []string{"-e"}, ""},
	{"xterm", []string{"-e"}, ""},
}

// findTerminal returns a terminal and the arguments before the program to run in it, or "" when there
// is none or no desktop to show it on.
func findTerminal() (string, []string) {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return "", nil
	}
	if own := strings.Fields(os.Getenv("TERMINAL")); len(own) > 0 {
		if path, err := lookPath(own[0]); err == nil {
			for _, t := range terminals {
				if filepath.Base(path) == t.program {
					return path, t.args
				}
			}
			return path, []string{"-e"}
		}
	}
	desktop := strings.ToUpper(os.Getenv("XDG_CURRENT_DESKTOP"))
	for _, ownDesktop := range []bool{true, false} {
		for _, t := range terminals {
			if ownDesktop && (t.desktop == "" || !strings.Contains(desktop, t.desktop)) {
				continue
			}
			if path, err := lookPath(t.program); err == nil {
				return path, t.args
			}
		}
	}
	return "", nil
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// elevateInTerminal opens a terminal that says why, asks for the sudo password there and runs args as
// root. It returns errNoAdminPrompt when there is no terminal, errAdminPromptRefused when the password
// was not given or the window was closed.
func elevateInTerminal(args []string) error {
	terminal, terminalArgs := findTerminal()
	if terminal == "" {
		return errNoAdminPrompt
	}
	dir, err := os.MkdirTemp("", "hiddify-sudo-")
	if err != nil {
		return fmt.Errorf("could not ask for the administrator password: %w", err)
	}
	defer os.RemoveAll(dir)
	status := filepath.Join(dir, "status")
	if err := syscall.Mkfifo(status, 0o600); err != nil {
		return fmt.Errorf("could not ask for the administrator password: %w", err)
	}
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = shellQuote(arg)
	}
	errFile := filepath.Join(dir, "error")
	// the result goes on fd 3, the FIFO: its end (the window closed too) is what the core waits for,
	// as many terminals return at once instead of when their window closes
	script := fmt.Sprintf(`#!/bin/sh
exec 3>%[1]s
printf '\033[1m%%s\033[0m\n\n' 'Hiddify needs root access for VPN mode'
if ! sudo -v -p '[sudo] password for %%u: '; then
	echo refused >&3
	exit 1
fi
sudo -n -- %[2]s 2>%[3]s
code=$?
echo "$code" >&3
exec 3>&-
if [ "$code" != 0 ]; then
	cat %[3]s
	printf '\nPress Enter to close.'
	read -r _
fi
`, shellQuote(status), strings.Join(quoted, " "), shellQuote(errFile))
	scriptPath := filepath.Join(dir, "run")
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		return fmt.Errorf("could not ask for the administrator password: %w", err)
	}

	result := make(chan string, 1)
	go func() {
		// blocks until the script opens it, then reads until the script closes it
		f, err := os.OpenFile(status, os.O_RDONLY, 0)
		if err != nil {
			result <- ""
			return
		}
		defer f.Close()
		out, _ := io.ReadAll(f)
		result <- strings.TrimSpace(string(out))
	}()

	cmd := exec.Command(terminal, append(append([]string{}, terminalArgs...), scriptPath)...)
	if err := cmd.Start(); err != nil {
		return errNoAdminPrompt
	}
	terminalExited := make(chan error, 1)
	go func() { terminalExited <- cmd.Wait() }()

	var got string
	timeout := time.NewTimer(terminalPromptTimeout)
	defer timeout.Stop()
	select {
	case got = <-result:
	case err := <-terminalExited:
		// a terminal that returns at once (a running instance opens the window) still runs the script;
		// one that failed never opens it
		if err != nil {
			select {
			case got = <-result:
			case <-time.After(2 * time.Second):
				unblockFifo(status)
				return errNoAdminPrompt
			}
		} else {
			select {
			case got = <-result:
			case <-timeout.C:
				unblockFifo(status)
				return errAdminPromptRefused
			}
		}
	case <-timeout.C:
		unblockFifo(status)
		return errAdminPromptRefused
	}
	switch got {
	case "0":
		return nil
	case "", "refused":
		return errAdminPromptRefused
	}
	code, _ := strconv.Atoi(got)
	detail, _ := os.ReadFile(errFile)
	if d := strings.TrimSpace(string(detail)); d != "" {
		return fmt.Errorf("could not install the VPN service (exit status %d): %s", code, d)
	}
	return fmt.Errorf("could not install the VPN service (exit status %d)", code)
}

// unblockFifo lets the reader goroutine finish when the script never opened the FIFO.
func unblockFifo(path string) {
	if f, err := os.OpenFile(path, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
		_ = f.Close()
	}
}
