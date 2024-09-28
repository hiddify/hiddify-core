package cmd

// import (
// 	"context"
// 	"fmt"
// 	"io"
// 	"math/rand"
// 	"net/http"
// 	"net/netip"
// 	"time"

// 	"github.com/hiddify/hiddify-core/common"
// 	// "github.com/hiddify/hiddify-core/extension_repository/cleanip_scanner"
// 	"github.com/spf13/cobra"
// 	"golang.org/x/net/proxy"
// )

// var commandTemp = &cobra.Command{
// 	Use:   "temp",
// 	Short: "temp",
// 	Args:  cobra.MaximumNArgs(2),
// 	Run: func(cmd *cobra.Command, args []string) {
// 		// fmt.Printf("Ping time: %d ms\n", Ping())
// 		scanner := cleanip_scanner.NewScannerEngine(&cleanip_scanner.ScannerOptions{
// 			UseIPv4:         true,
// 			UseIPv6:         common.CanConnectIPv6(),
// 			MaxDesirableRTT: 500 * time.Millisecond,
// 			IPQueueSize:     4,
// 			IPQueueTTL:      10 * time.Second,
// 			ConcurrentPings: 10,
// 			// MaxDesirableIPs: e.count,
// 			CidrList: cleanip_scanner.DefaultCFRanges(),
// 			PingFunc: func(ip netip.Addr) (cleanip_scanner.IPInfo, error) {
// 				fmt.Printf("Ping: %s\n", ip.String())
// 				return cleanip_scanner.IPInfo{
// 					AddrPort:  netip.AddrPortFrom(ip, 80),
// 					RTT:       time.Duration(rand.Intn(1000)),
// 					CreatedAt: time.Now(),
// 				}, nil
// 			},
// 		},
// 		)

// 		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
// 		defer cancel()

// 		scanner.Run(ctx)

// 		t := time.NewTicker(1 * time.Second)
// 		defer t.Stop()

// 		for {
// 			ipList := scanner.GetAvailableIPs(false)
// 			if len(ipList) > 1 {
// 				// e.result = ""
// 				for i := 0; i < 2; i++ {
// 					// result = append(result, ipList[i])
// 					// e.result = e.result + ipList[i].AddrPort.String() + "\n"
// 					fmt.Printf("%d %s\n", ipList[i].RTT, ipList[i].AddrPort.String())
// 				}
// 				return
// 			}

// 			select {
// 			case <-ctx.Done():
// 				// Context is done
// 				return
// 			case <-t.C:
// 				// Prevent the loop from spinning too fast
// 				continue
// 			}
// 		}
// 	},
// }

// func init() {
// 	mainCommand.AddCommand(commandTemp)
// }

// func GetContent(url string) (string, error) {
// 	return ContentFromURL("GET", url, 10*time.Second)
// }

// func ContentFromURL(method string, url string, timeout time.Duration) (string, error) {
// 	if method == "" {
// 		return "", fmt.Errorf("empty method")
// 	}
// 	if url == "" {
// 		return "", fmt.Errorf("empty url")
// 	}

// 	req, err := http.NewRequest(method, url, nil)
// 	if err != nil {
// 		return "", err
// 	}

// 	dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:12334", nil, proxy.Direct)
// 	if err != nil {
// 		return "", err
// 	}

// 	transport := &http.Transport{
// 		Dial: dialer.Dial,
// 	}

// 	client := &http.Client{
// 		Transport: transport,
// 		Timeout:   timeout,
// 	}

// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
// 		return "", fmt.Errorf("request failed with status code: %d", resp.StatusCode)
// 	}

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return "", err
// 	}

// 	if body == nil {
// 		return "", fmt.Errorf("empty body")
// 	}

// 	return string(body), nil
// }

// func Ping() int {
// 	startTime := time.Now()
// 	_, err := ContentFromURL("HEAD", "https://cp.cloudflare.com", 4*time.Second)
// 	if err != nil {
// 		return -1
// 	}
// 	duration := time.Since(startTime)
// 	return int(duration.Milliseconds())
// }
