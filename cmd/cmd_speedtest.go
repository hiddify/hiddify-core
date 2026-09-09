package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/showwin/speedtest-go/speedtest"
	"github.com/spf13/cobra"
)

// speedtest.net often resets non-browser clients, especially through proxies.
const speedtestUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

var (
	speedtestProxy      string
	speedtestDistance   float64
	speedtestNoDownload bool
	speedtestNoUpload   bool
	speedtestNoPing     bool
	speedtestNoJitter   bool
	speedtestVerbose    int
)

var commandSpeedtest = &cobra.Command{
	Use:   "speedtest",
	Short: "Run speedtest.net download/upload/ping/jitter through a proxy",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runSpeedtest(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	commandSpeedtest.Flags().StringVar(&speedtestProxy, "proxy", "", "proxy URL (socks://[user:pass@]host:port)")
	commandSpeedtest.Flags().Float64Var(&speedtestDistance, "distance", 2000, "target server distance in km")
	commandSpeedtest.Flags().BoolVar(&speedtestNoDownload, "no-download", false, "disable download test")
	commandSpeedtest.Flags().BoolVar(&speedtestNoUpload, "no-upload", false, "disable upload test")
	commandSpeedtest.Flags().BoolVar(&speedtestNoPing, "no-ping", false, "disable ping test")
	commandSpeedtest.Flags().BoolVar(&speedtestNoJitter, "no-jitter", false, "disable jitter output")
	commandSpeedtest.Flags().CountVarP(&speedtestVerbose, "verbose", "v", "debug logs (-vv for more detail)")
	mainCommand.AddCommand(commandSpeedtest)
}

func runSpeedtest() error {
	if speedtestNoPing {
		speedtestNoJitter = true
	}

	proxy := speedtestProxy
	if proxy != "" && !strings.Contains(proxy, "://") {
		proxy = "socks://" + proxy
	}
	if speedtestVerbose >= 2 && proxy != "" {
		fmt.Fprintf(os.Stderr, "debug: speedtest proxy %s\n", proxy)
	}

	userConfig := &speedtest.UserConfig{
		UserAgent: speedtestUserAgent,
	}
	opts := []speedtest.Option{speedtest.WithUserConfig(userConfig)}
	if proxy != "" {
		doer, err := newSpeedtestHTTPClient(proxy, speedtestUserAgent)
		if err != nil {
			return err
		}
		opts = append(opts, speedtest.WithDoer(doer))
	}
	client := speedtest.New(opts...)

	if _, err := client.FetchUserInfo(); err != nil {
		return fmt.Errorf("fetch user info: %w", err)
	}

	servers, err := client.FetchServers()
	if err != nil {
		return fmt.Errorf("fetch servers: %w", err)
	}

	server, err := selectSpeedtestServer(servers, speedtestDistance)
	if err != nil {
		return err
	}
	if speedtestVerbose >= 2 {
		fmt.Fprintf(os.Stderr, "debug: using server %s (target %.0fkm)\n", server.String(), speedtestDistance)
	}

	var runErr error

	if !speedtestNoPing {
		err := server.PingTest(nil)
		pingValue := formatDuration(server.Latency)
		if !speedtestNoJitter {
			pingValue += " ±" + formatDuration(server.Jitter)
		}
		printCheckResult("ping", err == nil, pingValue)
		if err != nil {
			runErr = fmt.Errorf("ping: %w", err)
		}
	}

	if !speedtestNoDownload {
		err := server.DownloadTest()
		printCheckResult("download", err == nil, formatSpeedtestRate(server.DLSpeed))
		if err != nil && runErr == nil {
			runErr = fmt.Errorf("download: %w", err)
		}
	}

	if !speedtestNoUpload {
		err := server.UploadTest()
		printCheckResult("upload", err == nil, formatSpeedtestRate(server.ULSpeed))
		if err != nil && runErr == nil {
			runErr = fmt.Errorf("upload: %w", err)
		}
	}

	return runErr
}

func selectSpeedtestServer(servers speedtest.Servers, targetKm float64) (*speedtest.Server, error) {
	if len(servers) == 0 {
		return nil, fmt.Errorf("no speedtest server found")
	}
	bestIndex := 0
	bestDelta := math.MaxFloat64
	for i, server := range servers {
		delta := math.Abs(server.Distance - targetKm)
		if delta < bestDelta {
			bestDelta = delta
			bestIndex = i
		}
	}
	return servers[bestIndex], nil
}

func formatSpeedtestRate(rate speedtest.ByteRate) string {
	s := strings.TrimSpace(rate.String())
	return strings.ReplaceAll(s, " ", "")
}

func newSpeedtestHTTPClient(proxy, userAgent string) (*http.Client, error) {
	dialer, err := newHealthDialer(proxy)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialer.Dial(network, address)
		},
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	return &http.Client{
		Transport: &speedtestUserAgentTransport{
			userAgent: userAgent,
			base:      transport,
		},
	}, nil
}

type speedtestUserAgentTransport struct {
	userAgent string
	base      http.RoundTripper
}

func (t *speedtestUserAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.Header.Set("User-Agent", t.userAgent)
	return t.base.RoundTrip(cloned)
}
