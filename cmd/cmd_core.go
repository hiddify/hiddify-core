package cmd

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hiddify/hiddify-core/v2/hcore"
	"github.com/hiddify/hiddify-core/v2/hcore/hprocess"

	"github.com/spf13/cobra"
)

// The desktop app runs its core in a process of its own with this command, instead of loading the
// library into the UI process. The app reads one line from stdout: CoreReadyPrefix followed by the
// gRPC server certificate (base64 PEM, empty without TLS), or CoreErrorPrefix followed by the reason.
const (
	CoreReadyPrefix = "HIDDIFY_CORE_READY "
	CoreErrorPrefix = "HIDDIFY_CORE_ERROR "
	// CoreSecretEnv carries the setup secret, so it never shows in the process list.
	CoreSecretEnv = "HIDDIFY_CORE_SECRET"
	// coreStopTimeout bounds the graceful stop: the system proxy is restored by it.
	coreStopTimeout = 8 * time.Second
	// coreProcessName is what htop and top show for the core on Linux, where the app and the core
	// cannot be told apart by their icon as on Windows and macOS.
	coreProcessName = "hiddify-core"
)

var coreOptions struct {
	baseDir    string
	workingDir string
	tempDir    string
	listen     string
	mode       int32
	ownerPID   int32
	debug      bool
}

var commandCore = &cobra.Command{
	Use:    "core",
	Short:  "Run the app's core in its own process (started by the desktop app)",
	Hidden: true,
	Args:   cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(runCore())
	},
}

func init() {
	flags := commandCore.Flags()
	flags.StringVar(&coreOptions.baseDir, "base-dir", "", "base directory")
	flags.StringVar(&coreOptions.workingDir, "working-dir", "", "working directory")
	flags.StringVar(&coreOptions.tempDir, "temp-dir", "", "temporary directory")
	flags.StringVar(&coreOptions.listen, "listen", "127.0.0.1:17078", "gRPC listen address")
	flags.Int32Var(&coreOptions.mode, "mode", int32(hcore.SetupMode_GRPC_NORMAL), "setup mode")
	flags.Int32Var(&coreOptions.ownerPID, "owner-pid", 0, "the app's process: the core stops when it is gone")
	flags.BoolVar(&coreOptions.debug, "debug", false, "debug mode")
	mainCommand.AddCommand(commandCore)
}

func runCore() int {
	hprocess.SetName(coreProcessName, cliArgs)
	hprocess.RaisePriority()

	mode := hcore.SetupMode(coreOptions.mode)
	err := hcore.Setup(&hcore.SetupRequest{
		BasePath:   coreOptions.baseDir,
		WorkingDir: coreOptions.workingDir,
		TempDir:    coreOptions.tempDir,
		Listen:     coreOptions.listen,
		Secret:     os.Getenv(CoreSecretEnv),
		Debug:      coreOptions.debug,
		Mode:       mode,
	}, nil)
	if err != nil {
		writeCoreLine(CoreErrorPrefix + oneLine(err.Error()))
		return 1
	}
	// threads the setup started are named as well
	hprocess.SetName(coreProcessName, cliArgs)
	writeCoreLine(CoreReadyPrefix + base64.StdEncoding.EncodeToString(hcore.GetGrpcServerPublicKey()))
	// Nothing else goes to the app's pipes: once the app is gone a write to them would end the core
	// before it stops cleanly. Its log reaches the app over gRPC.
	hprocess.DetachStdio()

	reason := waitForCoreEnd(bufio.NewReader(os.Stdin), coreOptions.ownerPID)
	fmt.Fprintf(os.Stderr, "hiddify core: stopping (%s)\n", reason)
	stopped := make(chan struct{})
	go func() {
		hcore.Close(mode)
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(coreStopTimeout):
		fmt.Fprintln(os.Stderr, "hiddify core: the stop did not finish in time")
	}
	return 0
}

// waitForCoreEnd returns once the app is gone or asks the core to stop. The app holds the write end
// of stdin and never writes to it: it closes when the app exits, even when it is killed. Its
// process is watched as well, for a stdin that a wrapper keeps open.
func waitForCoreEnd(stdin io.Reader, ownerPID int32) string {
	end := make(chan string, 3)
	go func() {
		_, _ = io.Copy(io.Discard, stdin)
		end <- "the app closed its pipe"
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		end <- fmt.Sprint("signal ", <-signals)
	}()
	go func() {
		if hprocess.WaitOwnerGone(ownerPID, nil) {
			end <- "the app exited"
		}
	}()
	return <-end
}

func writeCoreLine(line string) {
	fmt.Fprintln(os.Stdout, line)
	_ = os.Stdout.Sync()
}

func oneLine(message string) string {
	out := []rune(message)
	for i, r := range out {
		if r == '\n' || r == '\r' {
			out[i] = ' '
		}
	}
	return string(out)
}
