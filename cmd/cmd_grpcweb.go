package cmd

import (
	"fmt"
	"os"

	"github.com/hiddify/hiddify-core/cmd/grpcweb"
	"github.com/spf13/cobra"
)

var (
	grpcwebWorkdir     string
	grpcwebListen      string
	grpcwebGRPCAddr    string
	grpcwebConnectOnly bool
	grpcwebConfigPath  string
	grpcwebMixedPort   int
)

var commandGRPCWeb = &cobra.Command{
	Use:   "grpcweb",
	Short: "HTML UI + Core gRPC on one port (Start/Stop/proxies)",
	Long: `Tiny browser harness for hiddify-core.

Hosts insecure Core gRPC and the HTML UI on the same TCP port (h2c).
Open http://127.0.0.1:18088 — also dial that host:port with a gRPC client.

Works in Cursor Simple Browser.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		err := grpcweb.Run(grpcweb.Options{
			Workdir:     grpcwebWorkdir,
			Listen:      grpcwebListen,
			GRPCAddr:    grpcwebGRPCAddr,
			ConnectOnly: grpcwebConnectOnly,
			ConfigPath:  grpcwebConfigPath,
			MixedPort:   grpcwebMixedPort,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	commandGRPCWeb.Flags().StringVar(&grpcwebWorkdir, "workdir", "", "core working dir (default: temp)")
	commandGRPCWeb.Flags().StringVar(&grpcwebListen, "listen", "0.0.0.0:18088", "single bind for Core gRPC + HTTP UI (h2c)")
	commandGRPCWeb.Flags().StringVar(&grpcwebGRPCAddr, "grpc", "", "with --connect: remote Core gRPC address")
	commandGRPCWeb.Flags().BoolVar(&grpcwebConnectOnly, "connect", false, "do not start a core; only attach to --grpc (HTTP still on --listen)")
	commandGRPCWeb.Flags().StringVar(&grpcwebConfigPath, "config", "", "optional default config path shown in the UI")
	commandGRPCWeb.Flags().IntVar(&grpcwebMixedPort, "mixed-port", 12334, "hiddify mixed-port when starting a local core")
	mainCommand.AddCommand(commandGRPCWeb)
}
