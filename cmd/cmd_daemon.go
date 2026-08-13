package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/hiddify/hiddify-core/v2/daemon"
	hcore "github.com/hiddify/hiddify-core/v2/hcore"
	"github.com/spf13/cobra"
)

var daemonStateDir string
var daemonSocket string
var daemonAllowedUID int

var commandDaemon = &cobra.Command{Use: "daemon", Short: "run the persistent local control daemon"}

var commandDaemonRun = &cobra.Command{
	Use:   "run",
	Short: "run without a terminal dependency",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		runtime, err := daemon.StartWithOptions(daemonStateDir, daemonSocket, daemon.Options{AllowedUID: daemonAllowedUID})
		if err != nil {
			return err
		}
		defer runtime.Close()
		if err := hcore.Setup(&hcore.SetupRequest{
			BasePath:   daemonStateDir,
			WorkingDir: daemonStateDir,
			TempDir:    filepath.Join(daemonStateDir, "tmp"),
			Mode:       hcore.SetupMode_OLD,
		}, nil); err != nil {
			return fmt.Errorf("initialize core: %w", err)
		}
		defer hcore.Stop()
		fmt.Fprintf(os.Stderr, "hiddify daemon listening on %s\n", daemonSocket)
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return runtime.ServeControl(ctx, daemon.NewControlServer())
	},
}

func init() {
	commandDaemonRun.Flags().StringVar(&daemonStateDir, "state-dir", "/var/lib/hiddify", "daemon-owned state directory")
	commandDaemonRun.Flags().StringVar(&daemonSocket, "socket", "/run/hiddify/control.sock", "local control socket")
	commandDaemonRun.Flags().IntVar(&daemonAllowedUID, "allowed-uid", -1, "designated local Unix user allowed to control the daemon (-1 uses socket permissions only)")
	commandDaemon.AddCommand(commandDaemonRun)
	mainCommand.AddCommand(commandDaemon)
}
