// Package daemon owns the lifecycle primitives for the always-running local
// control daemon. It acquires the state lock, serves the control.v1 API on a
// Unix socket, and delegates core networking to the hcore package.
package daemon

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// Runtime holds the acquired state lock and control listener.
type Runtime struct {
	listener net.Listener
	lock     *os.File
	socket   string
}

// Options controls Unix socket ownership and peer admission. An allowed UID of
// -1 retains the default of filesystem permissions only.
type Options struct {
	AllowedUID int
}

// Start acquires the daemon-owned state lock and creates the local control
// socket with default (permission-only) admission.
func Start(stateDir, socket string) (*Runtime, error) {
	return StartWithOptions(stateDir, socket, Options{AllowedUID: -1})
}

// StartWithOptions acquires the state lock and creates the control socket. A
// running daemon is never replaced, and a stale socket file is removed only
// after a live listener has been proven absent.
func StartWithOptions(stateDir, socket string, options Options) (*Runtime, error) {
	if stateDir == "" || socket == "" {
		return nil, errors.New("state directory and socket path are required")
	}
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return nil, fmt.Errorf("create state directory: %w", err)
	}
	lock, err := os.OpenFile(filepath.Join(stateDir, "daemon.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open daemon lock: %w", err)
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		return nil, fmt.Errorf("another hiddify daemon owns %s: %w", stateDir, err)
	}
	if err := os.MkdirAll(filepath.Dir(socket), 0o750); err != nil {
		lock.Close()
		return nil, fmt.Errorf("create socket directory: %w", err)
	}
	if err := removeStaleSocket(socket); err != nil {
		lock.Close()
		return nil, err
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		lock.Close()
		return nil, fmt.Errorf("listen on control socket: %w", err)
	}
	if options.AllowedUID >= 0 {
		if err := os.Chown(socket, options.AllowedUID, -1); err != nil {
			listener.Close()
			os.Remove(socket)
			lock.Close()
			return nil, fmt.Errorf("set control socket owner: %w", err)
		}
	}
	if err := os.Chmod(socket, socketMode(options.AllowedUID)); err != nil {
		listener.Close()
		os.Remove(socket)
		lock.Close()
		return nil, fmt.Errorf("set control socket mode: %w", err)
	}
	return &Runtime{listener: authorizeListener(listener, options.AllowedUID), lock: lock, socket: socket}, nil
}

// Close releases the listener, removes the socket file, and drops the lock.
func (r *Runtime) Close() error {
	if r.listener != nil {
		r.listener.Close()
	}
	if r.socket != "" {
		os.Remove(r.socket)
	}
	if r.lock != nil {
		r.lock.Close()
	}
	return nil
}

func removeStaleSocket(socket string) error {
	info, err := os.Stat(socket)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("refusing to remove non-socket path %s", socket)
	}
	connection, dialErr := net.DialTimeout("unix", socket, 200*time.Millisecond)
	if dialErr == nil {
		connection.Close()
		return fmt.Errorf("control socket %s is owned by a running daemon", socket)
	}
	return os.Remove(socket)
}
