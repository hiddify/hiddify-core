package cmd

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/protocol/socks"
	"github.com/spf13/cobra"
)

var (
	healthProxyAddr string
	healthDNS       bool
	healthHTTP      bool
	healthQUIC      bool
	healthVerbose   int
	healthTimeout   time.Duration
	healthDNSAddr   string
	healthTarget    string
)

var commandHealth = &cobra.Command{
	Use:   "health [target]",
	Short: "Check connectivity through a proxy",
	Long:  "Test DNS, HTTP, or QUIC through socks:// or direct dial.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			healthTarget = args[0]
		}
		if healthTarget == "" {
			healthTarget = "google.com"
		}
		if !healthDNS && !healthHTTP && !healthQUIC {
			healthHTTP = true
		}
		if err := runHealthChecks(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	commandHealth.Flags().StringVar(&healthProxyAddr, "proxy", "", "proxy URL (socks://[user:pass@]host:port)")
	commandHealth.Flags().BoolVar(&healthDNS, "dns", false, "test DNS")
	commandHealth.Flags().BoolVar(&healthHTTP, "http", false, "test HTTP over TCP")
	commandHealth.Flags().BoolVar(&healthQUIC, "quic", false, "test QUIC over UDP")
	commandHealth.Flags().CountVarP(&healthVerbose, "verbose", "v", "debug logs (-vv for more detail)")
	commandHealth.Flags().DurationVar(&healthTimeout, "timeout", 15*time.Second, "timeout per check")
	commandHealth.Flags().StringVar(&healthDNSAddr, "dns-server", "1.1.1.1:53", "DNS server (host:port, udp://host:port, or tcp://host:port)")
	mainCommand.AddCommand(commandHealth)
}

type healthCheckResult struct {
	name     string
	ok       bool
	duration time.Duration
	err      error
}

func runHealthChecks() error {
	start := time.Now()
	dialer, err := newHealthDialer(healthProxyAddr)
	if err != nil {
		return err
	}

	run := func(name string, fn func() (healthCheckResult, error)) error {
		checkStart := time.Now()
		result, err := fn()
		if result.name == "" {
			result.name = name
		}
		if result.duration == 0 {
			result.duration = time.Since(checkStart)
		}
		result.ok = err == nil
		result.err = err
		printHealthResult(result)
		if err != nil {
			return fmt.Errorf("%s: %w", result.name, err)
		}
		return nil
	}

	if healthDNS {
		if err := run("dns", func() (healthCheckResult, error) {
			duration, err := testHealthDNS(dialer)
			if err != nil {
				fmt.Fprintf(os.Stderr, "debug: dns failed, retrying\n")
				duration, err = testHealthDNS(dialer)
			}
			return healthCheckResult{duration: duration}, err
		}); err != nil {
			return err
		}
	}
	if healthHTTP {
		if err := run("http", func() (healthCheckResult, error) {
			duration, err := testHealthHTTP(dialer)
			return healthCheckResult{duration: duration}, err
		}); err != nil {
			return err
		}
	}
	if healthQUIC {
		if err := run("quic", func() (healthCheckResult, error) {
			duration, err := testHealthQUIC(dialer)
			return healthCheckResult{duration: duration}, err
		}); err != nil {
			return err
		}
	}
	if healthVerbose >= 2 {
		fmt.Fprintf(os.Stderr, "debug: total duration=%s\n", formatDuration(time.Since(start)))
	}
	return nil
}

func printHealthResult(r healthCheckResult) {
	value := ""
	if r.ok && r.duration > 0 {
		value = formatDuration(r.duration)
	}
	printCheckResult(r.name, r.ok, value)
	if healthVerbose >= 2 && r.err != nil {
		fmt.Fprintf(os.Stderr, "debug: %s: %v\n", r.name, r.err)
	}
}

func printCheckResult(name string, ok bool, value string) {
	mark := "✓"
	if !ok {
		mark = "✗"
	}
	if value != "" {
		fmt.Printf("%s: %s %s\n", name, mark, value)
		return
	}
	fmt.Printf("%s: %s\n", name, mark)
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return d.Round(time.Microsecond).String()
	}
	return d.Round(time.Millisecond).String()
}

func healthLogf(format string, args ...any) {
	if healthVerbose >= 2 {
		fmt.Fprintf(os.Stderr, "debug: "+format+"\n", args...)
	}
}

type healthDialer interface {
	Dial(network, address string) (net.Conn, error)
}

func newHealthDialer(proxyAddr string) (healthDialer, error) {
	if proxyAddr == "" {
		return healthDirectDialer{}, nil
	}
	if !strings.Contains(proxyAddr, "://") {
		proxyAddr = "socks://" + proxyAddr
	}
	healthLogf("using proxy %s", proxyAddr)
	client, err := socks.NewClientFromURL(N.SystemDialer, proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("parse proxy URL: %w", err)
	}
	return healthSocksDialer{client: client}, nil
}

type healthDirectDialer struct{}

func (healthDirectDialer) Dial(network, address string) (net.Conn, error) {
	healthLogf("direct dial %s %s", network, address)
	return N.SystemDialer.DialContext(context.Background(), network, metadata.ParseSocksaddr(address))
}

type healthSocksDialer struct {
	client *socks.Client
}

func (d healthSocksDialer) Dial(network, address string) (net.Conn, error) {
	healthLogf("proxy dial %s %s", network, address)
	return d.client.DialContext(context.Background(), network, metadata.ParseSocksaddr(address))
}

func testHealthDNS(d healthDialer) (time.Duration, error) {
	start := time.Now()
	server := healthDNSAddr
	network := "udp"
	if strings.HasPrefix(server, "udp://") {
		server = strings.TrimPrefix(server, "udp://")
	} else if strings.HasPrefix(server, "tcp://") {
		network = "tcp"
		server = strings.TrimPrefix(server, "tcp://")
	}
	if !strings.Contains(server, ":") {
		server = net.JoinHostPort(server, "53")
	}

	name := healthTarget
	if strings.Contains(name, "://") {
		u, err := url.Parse(name)
		if err == nil && u.Hostname() != "" {
			name = u.Hostname()
		}
	}
	query := buildDNSQuery(name)

	conn, err := d.Dial(network, server)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	deadline := time.Now().Add(healthTimeout)
	if err := conn.SetDeadline(deadline); err != nil {
		return 0, err
	}

	payload := query
	if network == "tcp" {
		payload = make([]byte, 2+len(query))
		binary.BigEndian.PutUint16(payload[0:2], uint16(len(query)))
		copy(payload[2:], query)
	}

	if _, err = conn.Write(payload); err != nil {
		return 0, err
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return 0, err
	}
	if network == "tcp" {
		if n < 2 {
			return 0, fmt.Errorf("invalid DNS response")
		}
		length := int(binary.BigEndian.Uint16(buf[0:2]))
		if length+2 > n {
			return 0, fmt.Errorf("invalid DNS response")
		}
		buf = buf[2 : 2+length]
		n = length
	}
	if n < 12 || buf[3]&0x0F != 0 {
		return 0, fmt.Errorf("invalid DNS response")
	}
	return time.Since(start), nil
}

func buildDNSQuery(name string) []byte {
	query := make([]byte, 12)
	binary.BigEndian.PutUint16(query[0:2], 0xabcd)
	binary.BigEndian.PutUint16(query[2:4], 0x0100)
	binary.BigEndian.PutUint16(query[4:6], 1)

	labels := strings.Split(name, ".")
	for _, label := range labels {
		if label == "" {
			continue
		}
		query = append(query, byte(len(label)))
		query = append(query, label...)
	}
	query = append(query, 0x00, 0x00, 0x01, 0x00, 0x01)
	return query
}

func testHealthHTTP(d healthDialer) (time.Duration, error) {
	start := time.Now()
	testURL := "http://connectivitycheck.gstatic.com/generate_204"
	if strings.Contains(healthTarget, "://") {
		testURL = healthTarget
	}
	u, err := url.Parse(testURL)
	if err != nil {
		return 0, err
	}
	if u.Hostname() == "" {
		return 0, fmt.Errorf("invalid HTTP target")
	}
	client := newHealthHTTPClient(d, u)
	req, err := http.NewRequest(http.MethodHead, testURL, nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("HTTP status %s", resp.Status)
	}
	return time.Since(start), nil
}

func newHealthHTTPClient(d healthDialer, u *url.URL) *http.Client {
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	addr := net.JoinHostPort(u.Hostname(), port)
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return d.Dial("tcp", addr)
		},
		TLSClientConfig: &tls.Config{
			ServerName: u.Hostname(),
			MinVersion: tls.VersionTLS12,
		},
		DisableKeepAlives: true,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   healthTimeout,
	}
}

func testHealthQUIC(d healthDialer) (time.Duration, error) {
	start := time.Now()
	host := "cloudflare-dns.com"
	addr := net.JoinHostPort("1.1.1.1", "443")

	conn, err := d.Dial("udp", addr)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	packetConn := &healthUDPConn{Conn: conn, raddr: &net.UDPAddr{IP: net.ParseIP("1.1.1.1"), Port: 443}}

	ctx, cancel := context.WithTimeout(context.Background(), healthTimeout)
	defer cancel()

	tlsConf := &tls.Config{
		ServerName: host,
		NextProtos: []string{http3NextProto},
		MinVersion: tls.VersionTLS13,
	}

	tr := &quic.Transport{Conn: packetConn}
	qconn, err := tr.Dial(ctx, packetConn.raddr, tlsConf, &quic.Config{
		HandshakeIdleTimeout: healthTimeout,
		MaxIdleTimeout:       healthTimeout,
	})
	if err != nil {
		return 0, err
	}
	_ = qconn.CloseWithError(0, "health check done")
	return time.Since(start), nil
}

const http3NextProto = "h3"

type healthUDPConn struct {
	net.Conn
	raddr *net.UDPAddr
}

func (c *healthUDPConn) ReadFrom(p []byte) (int, net.Addr, error) {
	n, err := c.Conn.Read(p)
	return n, c.raddr, err
}

func (c *healthUDPConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	return c.Conn.Write(p)
}

func (c *healthUDPConn) LocalAddr() net.Addr {
	if c.raddr != nil {
		return &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	}
	return c.Conn.LocalAddr()
}

func (c *healthUDPConn) SetReadBuffer(bytes int) error  { return nil }
func (c *healthUDPConn) SetWriteBuffer(bytes int) error { return nil }
