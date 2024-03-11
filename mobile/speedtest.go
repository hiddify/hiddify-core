package mobile

import (
	"context"
	"fmt"
	"time"

	"go.jonnrb.io/speedtest/geo"
	"go.jonnrb.io/speedtest/speedtestdotnet"
	"go.jonnrb.io/speedtest/units"
	"golang.org/x/sync/errgroup"
)

var srvBlk []speedtestdotnet.ServerID = []speedtestdotnet.ServerID{}
var cfgTime time.Duration = time.Second * 10
var pngTime time.Duration = time.Second * 3
var ulTime time.Duration = time.Second * 5
var dlTime time.Duration = time.Second * 8

type SpeedTestDelegate interface {
	GotClientInfo(isp string, ip string)
	GotServerInfo(name string, location string, distance string, ping float64)
	GotUploadSpeed(value float64, final bool)
	GotDownloadSpeed(value float64, final bool)
	SpeedTestError(e error)
}

type SpeedTest struct {
	delegate *SpeedTestDelegate
	client   *speedtestdotnet.Client
	cancel   *context.CancelFunc
	stopped  bool
}

func NewSpeedTest(delegate SpeedTestDelegate) *SpeedTest {
	return &SpeedTest{
		delegate: &delegate,
		client:   &speedtestdotnet.Client{},
		cancel:   nil,
		stopped:  false,
	}
}

func (s *SpeedTest) Stop() {
	s.stopped = true
	if s.cancel != nil {
		(*s.cancel)()
	}
}

func (s *SpeedTest) Start() {
	s.stopped = false

	ctx, cancel := context.WithTimeout(context.Background(), cfgTime)
	s.cancel = &cancel
	defer cancel()
	servers, err := listServers(ctx, s.client)
	if err != nil {
		if !s.stopped {
			(*s.delegate).SpeedTestError(err)
		}
		return
	}

	if s.stopped {
		return
	}

	ctx, cancel = context.WithTimeout(context.Background(), cfgTime)
	s.cancel = &cancel
	defer cancel()
	cfg, err := s.client.Config(ctx)
	if err != nil {
		if !s.stopped {
			(*s.delegate).SpeedTestError(err)
		}
		return
	}

	(*s.delegate).GotClientInfo(cfg.ISP, cfg.IP)

	pingResult, err := selectServer(s.client, cfg, servers)
	if err != nil {
		if !s.stopped {
			(*s.delegate).SpeedTestError(err)
		}
		return
	}
	(*s.delegate).GotServerInfo(pingResult.server.Sponsor, pingResult.server.Name, pingResult.distance, pingResult.latency)

	if s.stopped {
		return
	}

	err = download(s.client, pingResult.server, s)
	if err != nil {
		if !s.stopped {
			(*s.delegate).SpeedTestError(err)
		}
		return
	}

	if s.stopped {
		return
	}

	err = upload(s.client, pingResult.server, s)
	if err != nil {
		if !s.stopped {
			(*s.delegate).SpeedTestError(err)
		}
		return
	}

}

// ------------------- Probe ------------------------
func download(client *speedtestdotnet.Client, server speedtestdotnet.Server, s *SpeedTest) error {
	ctx, cancel := context.WithTimeout(context.Background(), dlTime)
	s.cancel = &cancel
	defer cancel()

	stream, finalize := proberPrinter(s, false)
	speed, err := server.ProbeDownloadSpeed(ctx, client, stream)
	if err != nil {
		return err
	}
	finalize(speed)
	return nil
}

func upload(client *speedtestdotnet.Client, server speedtestdotnet.Server, s *SpeedTest) error {
	ctx, cancel := context.WithTimeout(context.Background(), ulTime)
	s.cancel = &cancel
	defer cancel()

	stream, finalize := proberPrinter(s, true)
	speed, err := server.ProbeUploadSpeed(ctx, client, stream)
	if err != nil {
		return err
	}
	finalize(speed)

	return nil
}

func proberPrinter(st *SpeedTest, upload bool) (
	stream chan units.BytesPerSecond,
	finalize func(units.BytesPerSecond),
) {
	stream = make(chan units.BytesPerSecond)
	var g errgroup.Group
	g.Go(func() error {
		for speed := range stream {
			if upload {
				(*st.delegate).GotUploadSpeed(formatSpeed(speed), false)
			} else {
				(*st.delegate).GotDownloadSpeed(formatSpeed(speed), false)
			}
		}
		return nil
	})

	finalize = func(s units.BytesPerSecond) {
		g.Wait()
		if upload {
			(*st.delegate).GotUploadSpeed(formatSpeed(s), true)
		} else {
			(*st.delegate).GotDownloadSpeed(formatSpeed(s), true)
		}
	}
	return
}

func formatSpeed(s units.BytesPerSecond) float64 {
	return float64(s.BitsPerSecond())
}

// ------------------- Ping Result ------------------
type pingResult struct {
	server   speedtestdotnet.Server
	distance string
	latency  float64
}

// ------------------- servers ----------------------
func selectServer(client *speedtestdotnet.Client, cfg speedtestdotnet.Config, servers []speedtestdotnet.Server) (*pingResult, error) {
	var (
		distance geo.Kilometers
		latency  time.Duration
		server   speedtestdotnet.Server
	)

	ctx, cancel := context.WithTimeout(context.Background(), pngTime)
	defer cancel()

	distanceMap := speedtestdotnet.SortServersByDistance(servers, cfg.Coordinates)

	// Truncate to just a few of the closest servers for the latency test.
	const maxCloseServers = 5
	closestServers := func() []speedtestdotnet.Server {
		if len(servers) > maxCloseServers {
			return servers[:maxCloseServers]
		} else {
			return servers
		}
	}()

	latencyMap, err := speedtestdotnet.StableSortServersByAverageLatency(
		closestServers, ctx, client, speedtestdotnet.DefaultLatencySamples)
	if err != nil {
		return nil, err
	}

	server = closestServers[0]
	latency = latencyMap[server.ID]
	distance = distanceMap[server.ID]

	return &pingResult{
		server:   server,
		latency:  float64(latency) / float64(time.Millisecond),
		distance: fmt.Sprintf("%s", distance),
	}, nil
}

func listServers(
	ctx context.Context,
	client *speedtestdotnet.Client,
) ([]speedtestdotnet.Server, error) {
	servers, err := client.LoadAllServers(ctx)
	if err != nil {
		return nil, err
	}
	if len(servers) == 0 {
		return nil, fmt.Errorf("no servers")
	}
	if len(srvBlk) != 0 {
		servers = pruneBlockedServers(servers)
	}
	return servers, nil
}

func pruneBlockedServers(servers []speedtestdotnet.Server) []speedtestdotnet.Server {
	n := make([]speedtestdotnet.Server, len(servers)-len(srvBlk))[:0]
	for _, s := range servers {
		var i bool
		for _, b := range srvBlk {
			if s.ID == b {
				i = true
			}
		}
		if !i {
			n = append(n, s)
		}
	}
	return n
}
