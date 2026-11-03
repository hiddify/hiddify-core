package hcore

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sagernet/sing-box/adapter"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"

	"github.com/hiddify/hiddify-core/v2/config"
	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
)

// Speed test: how fast a proxy uploads and downloads, through Cloudflare's speed test. An upload
// sends a few megabytes, which the server answers once it has them all; a download receives them.
// The results are kept by tag and go out with the outbounds info (upload_speed, download_speed).
// They outlive the run, in speedTestsFile: the next run may be on another network, so they go out
// marked as from an earlier run (upload_speed_cached, download_speed_cached) until tested again,
// or until ClearSpeedTests forgets them.

type direction int

const (
	upload direction = iota
	download
)

func (d direction) String() string {
	if d == download {
		return "download"
	}
	return "upload"
}

const (
	speedTestHost    = "speed.cloudflare.com"
	speedTestDefault = 1 << 20
	speedTestMax     = 50 << 20
	// speedTestParallel proxies are tested at once
	speedTestParallel = 3
	// a proxy may take speedTestTimeout, and a second more for each speedTestSlowest bytes
	speedTestTimeout = 30 * time.Second
	speedTestSlowest = 256 << 10
)

// speedTestSlots are the tests that run at once, of every batch together: a download and an
// upload asked for together share them, as they share this device's link.
var speedTestSlots = make(chan struct{}, speedTestParallel)

// proxyTests keeps the tests of one proxy to one at a time: its download and upload at once would
// share its link, and each measure less than it can.
var proxyTests sync.Map // tag → *sync.Mutex

// lockProxyTest waits for tag's turn, and returns its release.
func lockProxyTest(tag string) func() {
	lock, _ := proxyTests.LoadOrStore(tag, &sync.Mutex{})
	mutex := lock.(*sync.Mutex)
	mutex.Lock()
	return mutex.Unlock
}

// uploadBlock is what an upload test sends, again and again: random, so nothing on the way can
// make it smaller, and one block for every test, so 50 MB tests do not take 50 MB of memory each.
var uploadBlock = sync.OnceValue(func() []byte {
	block := make([]byte, 64<<10)
	_, _ = rand.Read(block)
	return block
})

// uploadBody reads size bytes of uploadBlock, over and over.
type uploadBody struct {
	left int64
	at   int
}

func (b *uploadBody) Read(p []byte) (int, error) {
	if b.left <= 0 {
		return 0, io.EOF
	}
	block := uploadBlock()
	n := copy(p[:min(int64(len(p)), b.left)], block[b.at:])
	b.at = (b.at + n) % len(block)
	b.left -= int64(n)
	return n, nil
}

type speedResults struct {
	speed   map[string]int64 // tag → bytes a second; -1 failed
	testing map[string]bool
	cached  map[string]bool // the speed is from an earlier run
}

var speedTests struct {
	sync.Mutex
	results   [2]speedResults // by direction
	listeners map[chan struct{}]struct{}
	loaded    bool // the results of earlier runs were read
}

// speedInfo is what the outbounds info tells of tag's test in direction d.
func speedInfo(d direction, tag string) (speed int64, testing bool, cached bool) {
	speedTests.Lock()
	defer speedTests.Unlock()
	loadSpeedTestsLocked()
	results := speedTests.results[d]
	return results.speed[tag], results.testing[tag], results.cached[tag]
}

// savedSpeedTests is speedTestsFile's content: the last speed of each tag, by direction.
type savedSpeedTests struct {
	Upload   map[string]int64 `json:"upload,omitempty"`
	Download map[string]int64 `json:"download,omitempty"`
}

// speedTestsFile keeps the speed test results across runs and restarts; "" until set up.
func speedTestsFile() string {
	if sWorkingPath == "" {
		return ""
	}
	return filepath.Join(sWorkingPath, "data", "speed_tests.json")
}

// loadSpeedTestsLocked reads the results of earlier runs, once, as cached; speedTests must be
// locked.
func loadSpeedTestsLocked() {
	path := speedTestsFile()
	if speedTests.loaded || path == "" {
		return
	}
	speedTests.loaded = true
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var saved savedSpeedTests
	if json.Unmarshal(content, &saved) != nil {
		return
	}
	for d, speeds := range map[direction]map[string]int64{upload: saved.Upload, download: saved.Download} {
		results := resultsLocked(d)
		for tag, speed := range speeds {
			if _, known := results.speed[tag]; !known {
				results.speed[tag], results.cached[tag] = speed, true
			}
		}
	}
}

// saveSpeedTestsLocked writes every speed to speedTestsFile; speedTests must be locked.
func saveSpeedTestsLocked() {
	path := speedTestsFile()
	if path == "" {
		return
	}
	content, err := json.Marshal(savedSpeedTests{
		Upload:   speedTests.results[upload].speed,
		Download: speedTests.results[download].speed,
	})
	if err == nil {
		err = os.MkdirAll(filepath.Dir(path), 0o755)
	}
	if err == nil {
		err = os.WriteFile(path, content, 0o644)
	}
	if err != nil {
		Log(LogLevel_DEBUG, LogType_CORE, "saving the speed tests: ", err.Error())
	}
}

// resultsLocked is direction d's results, made ready; speedTests must be locked.
func resultsLocked(d direction) *speedResults {
	results := &speedTests.results[d]
	if results.speed == nil {
		results.speed, results.testing, results.cached = make(map[string]int64), make(map[string]bool), make(map[string]bool)
	}
	return results
}

// notifyLocked tells the listeners a speed test changed; speedTests must be locked.
func notifyLocked() {
	for ch := range speedTests.listeners {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// subscribeSpeedTests is told of each change of a speed test, until unsubscribed.
func subscribeSpeedTests() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	speedTests.Lock()
	if speedTests.listeners == nil {
		speedTests.listeners = make(map[chan struct{}]struct{})
	}
	speedTests.listeners[ch] = struct{}{}
	speedTests.Unlock()
	return ch, func() {
		speedTests.Lock()
		delete(speedTests.listeners, ch)
		speedTests.Unlock()
	}
}

// setSpeedLocked records tag's test in direction d, testing or done with speed, keeps it, and tells
// the listeners; speedTests must be locked.
func setSpeedLocked(d direction, tag string, testing bool, speed int64) {
	loadSpeedTestsLocked()
	results := resultsLocked(d)
	if testing {
		results.testing[tag] = true
	} else {
		delete(results.testing, tag)
		delete(results.cached, tag)
		results.speed[tag] = speed
		saveSpeedTestsLocked()
	}
	notifyLocked()
}

// dropTestLocked ends tag's test in direction d with no result, as one the core stopped; the speed
// it had stays. speedTests must be locked.
func dropTestLocked(d direction, tag string) {
	delete(resultsLocked(d).testing, tag)
	notifyLocked()
}

// cacheSpeedTests marks every speed as from an earlier run, as the core stops: the next run may be
// on another network.
func cacheSpeedTests() {
	speedTests.Lock()
	defer speedTests.Unlock()
	for d := range speedTests.results {
		results := resultsLocked(direction(d))
		for tag := range results.speed {
			results.cached[tag] = true
		}
		clear(results.testing)
	}
	notifyLocked()
}

// clearSpeedTests forgets every speed, of this run and the earlier ones.
func clearSpeedTests() {
	speedTests.Lock()
	defer speedTests.Unlock()
	speedTests.results = [2]speedResults{}
	// nothing left to read: the file goes with them
	speedTests.loaded = true
	if path := speedTestsFile(); path != "" {
		_ = os.Remove(path)
	}
	notifyLocked()
}

func (s *CoreService) ClearSpeedTests(ctx context.Context, empty *hcommon.Empty) (*hcommon.Response, error) {
	clearSpeedTests()
	return &hcommon.Response{Code: hcommon.ResponseCode_OK}, nil
}

func (s *CoreService) UploadTest(ctx context.Context, in *SpeedTestRequest) (*hcommon.Response, error) {
	return static.SpeedTest(upload, in)
}

func (s *CoreService) DownloadTest(ctx context.Context, in *SpeedTestRequest) (*hcommon.Response, error) {
	return static.SpeedTest(download, in)
}

// SpeedTest starts the test in direction d of a proxy, or of each proxy of a group, and returns at
// once; one already testing is not tested twice.
func (h *HiddifyInstance) SpeedTest(d direction, in *SpeedTestRequest) (*hcommon.Response, error) {
	box, ctx := h.Box(), h.Context()
	if box == nil || ctx == nil {
		err := E.New("the core is not running")
		return &hcommon.Response{Code: hcommon.ResponseCode_FAILED, Message: err.Error()}, err
	}
	tag := in.Tag
	if tag == "" {
		tag = h.activeSpeedTestTag()
	}
	outbound, loaded := box.Outbound().Outbound(tag)
	if !loaded {
		err := E.New("outbound not found: ", tag)
		return &hcommon.Response{Code: hcommon.ResponseCode_FAILED, Message: err.Error()}, err
	}
	size := in.Size
	if size <= 0 {
		size = speedTestDefault
	}
	size = min(size, speedTestMax)

	var targets []adapter.Outbound
	if group, isGroup := outbound.(adapter.OutboundGroup); isGroup {
		for _, tag := range group.All() {
			if out, ok := box.Outbound().Outbound(tag); ok && isProxyTag(tag) && !isGroupTag(box, tag) {
				targets = append(targets, out)
			}
		}
	} else {
		targets = []adapter.Outbound{outbound}
	}
	speedTests.Lock()
	queued := targets[:0]
	for _, out := range targets {
		if !speedTests.results[d].testing[out.Tag()] {
			setSpeedLocked(d, out.Tag(), true, 0)
			queued = append(queued, out)
		}
	}
	speedTests.Unlock()

	go func() {
		var wg sync.WaitGroup
		for _, out := range queued {
			wg.Add(1)
			speedTestSlots <- struct{}{}
			go func(out adapter.Outbound) {
				defer func() { <-speedTestSlots; wg.Done() }()
				release := lockProxyTest(out.Tag())
				speed, err := measureSpeed(ctx, d, out, size)
				release()
				// the core stopped under it: no result, rather than a failure of the proxy
				if err != nil && ctx.Err() != nil {
					speedTests.Lock()
					dropTestLocked(d, out.Tag())
					speedTests.Unlock()
					return
				}
				if err != nil {
					Log(LogLevel_DEBUG, LogType_CORE, d.String(), " test of ", out.Tag(), ": ", err.Error())
					speed = -1
				}
				speedTests.Lock()
				setSpeedLocked(d, out.Tag(), false, speed)
				speedTests.Unlock()
			}(out)
		}
		wg.Wait()
	}()
	return &hcommon.Response{Code: hcommon.ResponseCode_OK}, nil
}

// activeSpeedTestTag is the proxy the user's selection uses now, down through its groups.
func (h *HiddifyInstance) activeSpeedTestTag() string {
	return resolveActiveOutbound(h.Box(), config.OutboundSelectTag, h.UrlTestHistory())
}

// speedTestTagOf is whose speeds the outbound shows: its own, or a group's, the proxy the group
// uses now.
func (h *HiddifyInstance) speedTestTagOf(outbound adapter.Outbound) string {
	if _, isGroup := outbound.(adapter.OutboundGroup); isGroup {
		if box := h.Box(); box != nil {
			return resolveActiveOutbound(box, outbound.Tag(), h.UrlTestHistory())
		}
	}
	return outbound.Tag()
}

// measureSpeed moves size bytes through out in direction d and returns bytes a second. An upload
// is timed from the request's headers to the server's answer, which comes once it has them all; a
// download from the first byte of the answer to its last.
func measureSpeed(ctx context.Context, d direction, out adapter.Outbound, size int64) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, speedTestTimeout+time.Duration(size/speedTestSlowest)*time.Second)
	defer cancel()
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
				return out.DialContext(ctx, N.NetworkTCP, M.ParseSocksaddr(speedTestHost+":443"))
			},
			TLSClientConfig:    &tls.Config{ServerName: speedTestHost},
			DisableKeepAlives:  true,
			DisableCompression: true,
		},
	}
	defer client.CloseIdleConnections()
	var start, end time.Time
	trace := &httptrace.ClientTrace{
		WroteHeaders:         func() { start = time.Now() },
		GotFirstResponseByte: func() { end = time.Now() },
	}
	var request *http.Request
	var err error
	if d == upload {
		request, err = http.NewRequestWithContext(httptrace.WithClientTrace(ctx, trace), http.MethodPost, "https://"+speedTestHost+"/__up", &uploadBody{left: size})
		if err == nil {
			request.ContentLength = size
			request.Header.Set("Content-Type", "application/octet-stream")
		}
	} else {
		request, err = http.NewRequestWithContext(httptrace.WithClientTrace(ctx, trace), http.MethodGet, fmt.Sprintf("https://%s/__down?bytes=%d", speedTestHost, size), nil)
	}
	if err != nil {
		return 0, err
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("speed test answered %s", response.Status)
	}
	moved := size
	if d == upload {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
	} else {
		// the download runs from the answer's first byte to its last
		start = end
		if moved, err = io.Copy(io.Discard, response.Body); err != nil {
			return 0, err
		}
		end = time.Now()
	}
	elapsed := end.Sub(start)
	if start.IsZero() || end.IsZero() || elapsed <= 0 || moved <= 0 {
		return 0, E.New("no timing")
	}
	return int64(float64(moved) / elapsed.Seconds()), nil
}
