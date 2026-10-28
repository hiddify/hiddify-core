// Package grpcweb is a tiny HTML+gRPC harness for hiddify-core.
//
// It starts an insecure Core gRPC server and an HTML UI on the same TCP port
// (h2c multiplex: application/grpc → gRPC, everything else → HTTP UI).
//
//	go run ./cmd/main grpcweb --config /path/to/config.json
//
// Open http://127.0.0.1:18088 (works in Cursor Simple Browser).
// Dial the same host:port with a normal gRPC client.
package grpcweb

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/hiddify/hiddify-core/v2/hcore"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Same shape as the Flutter app (HiddifyNext/5.0.0 …).
func userAgent() string {
	return fmt.Sprintf("HiddifyNext/5.0.0 (%s) like ClashMeta v2ray sing-box", runtime.GOOS)
}

//go:embed index.html
var indexFS embed.FS

var marshal = protojson.MarshalOptions{EmitUnpopulated: true, UseEnumNumbers: false}

// Options configures the harness.
type Options struct {
	Workdir string
	// Listen is the single bind address for both Core gRPC and the HTTP UI
	// (default 0.0.0.0:18088). Ignored for the gRPC side when ConnectOnly.
	Listen string
	// GRPCAddr is only used with ConnectOnly: dial an already-running core.
	GRPCAddr    string
	ConnectOnly bool
	ConfigPath  string
	MixedPort   int
}

// uiSettings are the Hiddify options the harness exposes (proxy/VPN/chain).
type uiSettings struct {
	MixedPort      int    `json:"mixedPort"`
	ServiceMode    string `json:"serviceMode"` // proxy | system-proxy | vpn
	Chain          string `json:"chain"`       // off | extra_security_{warp_auto,warp,warp_masque,warp_awg,psiphon} | unblocker_{warp_auto,warp,warp_masque,warp_awg,psiphon}
	EnableTun      bool   `json:"enableTun"`
	SetSystemProxy bool   `json:"setSystemProxy"`
}

type server struct {
	listen       string
	grpcAddr     string
	mixedPort    int
	clashAPIPort int // stable across settings applies so hot reload does not see experimental change
	client       hcore.CoreClient
	conn         *grpc.ClientConn

	mu            sync.Mutex
	lastCore      *hcore.CoreInfoResponse
	lastGroups    *hcore.OutboundGroupList
	lastMain      *hcore.OutboundGroupList
	lastSystem    *hcore.SystemInfo
	lastConfig    string // last successfully started/parsed content (for restart/hotreload)
	settings      uiSettings
	coreListeners map[chan *hcore.CoreInfoResponse]struct{}

	// web ↔ grpcweb payload stats per Core RPC (not core VPN datapath)
	rpcStats *rpcStatsHub
}

// Run starts the local core + combined gRPC/HTTP listener (unless ConnectOnly),
// and blocks until SIGINT/SIGTERM.
func Run(opts Options) error {
	if opts.Listen == "" {
		// Bind all interfaces so Cursor Simple Browser / Electron can reach it.
		opts.Listen = "0.0.0.0:18088"
	}
	if opts.MixedPort == 0 {
		opts.MixedPort = 12334
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	dialAddr := dialTarget(opts.Listen)
	clashAPIPort := freePort()
	var grpcSrv *grpc.Server

	if !opts.ConnectOnly {
		dir := opts.Workdir
		if dir == "" {
			var err error
			dir, err = os.MkdirTemp("", "hiddify-grpcweb-*")
			if err != nil {
				return err
			}
			defer os.RemoveAll(dir)
		}
		for _, sub := range []string{"data", "configs"} {
			if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
				return err
			}
		}
		if err := setupLocalCore(dir, opts.MixedPort, clashAPIPort); err != nil {
			return fmt.Errorf("setup local core: %w", err)
		}
		grpcSrv = grpc.NewServer()
		hcore.RegisterCoreServer(grpcSrv, &hcore.CoreService{})
		log.Printf("local core ready · workdir %s · grpc+http %s", dir, opts.Listen)
	} else {
		if opts.GRPCAddr == "" {
			return fmt.Errorf("--connect requires --grpc host:port")
		}
		dialAddr = opts.GRPCAddr
		log.Printf("connect-only · dial grpc %s · http %s", dialAddr, opts.Listen)
	}

	mux := http.NewServeMux()
	s := &server{
		listen:       opts.Listen,
		grpcAddr:     dialAddr,
		mixedPort:    opts.MixedPort,
		clashAPIPort: clashAPIPort,
		settings: uiSettings{
			MixedPort:   opts.MixedPort,
			ServiceMode: "proxy",
			Chain:       "off",
		},
		coreListeners: make(map[chan *hcore.CoreInfoResponse]struct{}),
		rpcStats:      newRPCStatsHub(),
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		b, err := indexFS.ReadFile("index.html")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		s.handleStatus(w, r, opts.ConfigPath)
	})
	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/api/resolve", s.handleResolve)
	mux.HandleFunc("/api/start", s.handleStart)
	mux.HandleFunc("/api/stop", s.handleStop)
	mux.HandleFunc("/api/restart", s.handleRestart)
	mux.HandleFunc("/api/hotreload", s.handleHotReload)
	mux.HandleFunc("/api/select", s.handleSelect)
	mux.HandleFunc("/api/urltest-active", s.handleUrlTestActive)
	mux.HandleFunc("/api/urltest", s.handleUrlTest)
	mux.HandleFunc("/api/rpc", s.handleRPC)
	mux.HandleFunc("/api/bus", s.handleBus)
	// Legacy per-method SSE kept for compatibility; the UI uses /api/bus.
	mux.HandleFunc("/api/stream/core", s.streamCore)
	mux.HandleFunc("/api/stream/proxies", s.streamProxies(false))
	mux.HandleFunc("/api/stream/main", s.streamProxies(true))
	mux.HandleFunc("/api/stream/system", s.streamSystemInfo)
	mux.HandleFunc("/api/stream/logs", s.streamLogs)
	mux.HandleFunc("/api/stream/grpc-stats", s.streamGrpcStats)

	var handler http.Handler = withCORS(mux)
	if grpcSrv != nil {
		handler = withCORS(grpcHTTPMux(grpcSrv, mux))
	}

	httpServer := &http.Server{
		Addr:    opts.Listen,
		Handler: h2c.NewHandler(handler, &http2.Server{}),
	}
	ln, err := net.Listen("tcp", opts.Listen)
	if err != nil {
		return fmt.Errorf("listen %s: %w", opts.Listen, err)
	}
	openURL := displayURL(opts.Listen)
	go func() {
		log.Printf("open %s  · gRPC on same port (h2c)  (Cursor: Simple Browser → this URL)", openURL)
		if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Dial after the listener is up so the in-process CoreClient can attach.
	deadline := time.Now().Add(5 * time.Second)
	for {
		conn, err := grpc.NewClient(dialAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			s.conn = conn
			s.client = hcore.NewCoreClient(conn)
			break
		}
		if time.Now().After(deadline) {
			_ = httpServer.Close()
			return fmt.Errorf("dial grpc %s: %w", dialAddr, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
	defer s.conn.Close()

	go s.watchCore(ctx)
	go s.watchProxies(ctx, false)
	go s.watchProxies(ctx, true)
	go s.watchSystemInfo(ctx)

	<-ctx.Done()
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelShutdown()
	_ = httpServer.Shutdown(shutdown)
	if grpcSrv != nil {
		grpcSrv.GracefulStop()
	}
	if !opts.ConnectOnly {
		_, _ = hcore.Stop()
	}
	return nil
}

// grpcHTTPMux serves native gRPC (HTTP/2 + application/grpc) and the HTML UI
// on one listener.
func grpcHTTPMux(grpcSrv *grpc.Server, ui http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			grpcSrv.ServeHTTP(w, r)
			return
		}
		ui.ServeHTTP(w, r)
	})
}

func displayURL(httpAddr string) string {
	host, port, err := net.SplitHostPort(httpAddr)
	if err != nil {
		return "http://" + httpAddr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port)
}

// dialTarget turns a bind address into a loopback dial target.
func dialTarget(listen string) string {
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return listen
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}

// setupLocalCore initializes libbox/core without opening its own gRPC port;
// we register Core on the shared h2c listener instead.
func setupLocalCore(dir string, mixedPort, clashAPIPort int) error {
	err := hcore.Setup(&hcore.SetupRequest{
		BasePath:   dir,
		WorkingDir: dir,
		TempDir:    dir,
		Listen:     "",
		Mode:       hcore.SetupMode_OLD,
		Debug:      true,
	}, nil)
	if err != nil {
		return fmt.Errorf("Setup: %w", err)
	}
	settings := fmt.Sprintf(
		`{"mixed-port":%d,"tproxy-port":0,"redirect-port":0,"direct-port":0,"clash-api-port":%d,"enable-clash-api":true,"log-level":"warn"}`,
		mixedPort, clashAPIPort,
	)
	if _, err := hcore.ChangeHiddifySettings(&hcore.ChangeHiddifySettingsRequest{HiddifySettingsJson: settings}, false); err != nil {
		return fmt.Errorf("ChangeHiddifySettings: %w", err)
	}
	return nil
}

func freePort() int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 16756
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func (s *server) watchCore(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		stream, err := s.client.CoreInfoListener(ctx, &hcommon.Empty{})
		if err != nil {
			log.Printf("CoreInfoListener: %v", err)
			time.Sleep(time.Second)
			continue
		}
		for {
			info, err := stream.Recv()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("CoreInfo recv: %v", err)
				break
			}
			cloned := proto.Clone(info).(*hcore.CoreInfoResponse)
			s.mu.Lock()
			s.lastCore = cloned
			for ch := range s.coreListeners {
				select {
				case ch <- cloned:
				default:
					// slow client: drop oldest, keep newest
					select {
					case <-ch:
					default:
					}
					select {
					case ch <- cloned:
					default:
					}
				}
			}
			s.mu.Unlock()
			log.Printf("core state → %s (%s)", info.CoreState.String(), info.MessageType.String())
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func (s *server) subscribeCore() chan *hcore.CoreInfoResponse {
	ch := make(chan *hcore.CoreInfoResponse, 128)
	s.mu.Lock()
	s.coreListeners[ch] = struct{}{}
	s.mu.Unlock()
	return ch
}

func (s *server) unsubscribeCore(ch chan *hcore.CoreInfoResponse) {
	s.mu.Lock()
	delete(s.coreListeners, ch)
	s.mu.Unlock()
}

type outboundStream interface {
	Recv() (*hcore.OutboundGroupList, error)
}

func (s *server) coreRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastCore == nil {
		return false
	}
	switch s.lastCore.CoreState {
	case hcore.CoreStates_WAITING, hcore.CoreStates_STARTED,
		hcore.CoreStates_NO_AVAILABLE_PROXIES, hcore.CoreStates_HOT_RELOADING:
		return true
	default:
		return false
	}
}

func (s *server) clearProxies() {
	s.mu.Lock()
	s.lastGroups, s.lastMain = nil, nil
	s.mu.Unlock()
}

func (s *server) watchProxies(ctx context.Context, mainOnly bool) {
	for {
		if ctx.Err() != nil {
			return
		}
		if !s.coreRunning() {
			s.clearProxies()
			select {
			case <-ctx.Done():
				return
			case <-time.After(400 * time.Millisecond):
			}
			continue
		}
		var (
			stream outboundStream
			err    error
		)
		if mainOnly {
			stream, err = s.client.MainOutboundsInfo(ctx, &hcommon.Empty{})
		} else {
			stream, err = s.client.OutboundsInfo(ctx, &hcommon.Empty{})
		}
		if err != nil {
			if !s.coreRunning() {
				time.Sleep(400 * time.Millisecond)
				continue
			}
			log.Printf("OutboundsInfo(%v): %v", mainOnly, err)
			time.Sleep(time.Second)
			continue
		}
		for {
			list, err := stream.Recv()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				if !s.coreRunning() || strings.Contains(err.Error(), "hiddify service not found") {
					break
				}
				log.Printf("OutboundsInfo recv(%v): %v", mainOnly, err)
				break
			}
			s.mu.Lock()
			clone := proto.Clone(list).(*hcore.OutboundGroupList)
			if mainOnly {
				s.lastMain = clone
			} else {
				s.lastGroups = clone
			}
			s.mu.Unlock()
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func (s *server) grpcStatsSnapshot() map[string]any {
	if s.rpcStats == nil {
		return map[string]any{"total": map[string]any{}, "methods": []any{}, "catalog": coreRPCCatalog}
	}
	return s.rpcStats.snapshot()
}

func (s *server) noteRPC(method string, n int) {
	if s.rpcStats != nil {
		s.rpcStats.note(method, n)
	}
}

func (s *server) writeSSEMethod(w http.ResponseWriter, method string, m proto.Message) int {
	b, err := marshal.Marshal(m)
	if err != nil {
		return 0
	}
	s.noteRPC(method, len(b))
	fmt.Fprintf(w, "data: %s\n\n", b)
	return len(b)
}

func (s *server) writeJSONMethod(w http.ResponseWriter, method string, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		httpError(w, 500, err)
		return
	}
	s.noteRPC(method, len(b))
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func (s *server) streamGrpcStats(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", 500)
		return
	}
	setSSEHeaders(w)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	var last string
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			b, err := json.Marshal(s.grpcStatsSnapshot())
			if err != nil {
				continue
			}
			if string(b) == last {
				continue
			}
			last = string(b)
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		}
	}
}

func (s *server) handleStatus(w http.ResponseWriter, r *http.Request, defaultConfig string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	writeJSON(w, map[string]any{
		"grpc":          s.grpcAddr,
		"http":          s.listen,
		"listen":        s.listen,
		"samePort":      true,
		"userAgent":     userAgent(),
		"defaultConfig": defaultConfig,
		"core":          protoToAny(s.lastCore),
		"proxies":       protoToAny(s.lastGroups),
		"main":          protoToAny(s.lastMain),
		"openURL":       displayURL(s.listen),
		"settings":      s.settings,
		"grpcStats":     s.grpcStatsSnapshot(),
		"system":        protoToAny(s.lastSystem),
		"rpcCatalog":    coreRPCCatalog,
	})
}

type resolveResult struct {
	Kind       string `json:"kind"` // url | path | content
	Source     string `json:"source"`
	UserAgent  string `json:"userAgent,omitempty"`
	Raw        string `json:"raw"`
	Parsed     string `json:"parsed"`
	ParseError string `json:"parseError,omitempty"`
}

func (s *server) handleResolve(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Config string `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		httpError(w, 400, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	out, err := s.resolveAndParse(ctx, req.Config)
	if err != nil {
		httpError(w, 500, err)
		return
	}
	s.writeJSONMethod(w, "Parse", out)
}

func (s *server) handleStart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Config string `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		httpError(w, 400, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	if err := s.applyHiddifySettings(); err != nil {
		httpError(w, 500, err)
		return
	}
	need, blocked, err := s.needAdminBgCheck(ctx)
	if err != nil {
		httpError(w, 500, err)
		return
	}
	if blocked {
		s.writeBlockedByAdmin(w, need)
		return
	}
	resolved, err := s.resolveAndParse(ctx, req.Config)
	if err != nil {
		httpError(w, 500, err)
		return
	}
	content := resolved.Parsed
	if content == "" {
		content = resolved.Raw
	}
	out, err := s.client.Start(ctx, &hcore.StartRequest{
		ConfigContent: content,
		ConfigName:    "grpcweb",
	})
	if err != nil {
		httpError(w, 500, err)
		return
	}
	s.mu.Lock()
	s.lastConfig = content
	s.mu.Unlock()
	s.writeJSONMethod(w, "Start", map[string]any{
		"start":     protoToAny(out),
		"resolved":  resolved,
		"needAdmin": protoToAny(need),
	})
}

func (s *server) resolveAndParse(ctx context.Context, input string) (*resolveResult, error) {
	raw, kind, source, err := loadConfigInput(ctx, input)
	if err != nil {
		return nil, err
	}
	res := &resolveResult{
		Kind:      kind,
		Source:    source,
		Raw:       raw,
		UserAgent: userAgent(),
	}
	parsed, err := s.client.Parse(ctx, &hcore.ParseRequest{Content: raw, Debug: true})
	if err != nil {
		res.ParseError = err.Error()
		if parsed != nil && parsed.Message != "" {
			res.ParseError = parsed.Message
		}
		return res, nil
	}
	if parsed.ResponseCode != hcommon.ResponseCode_OK {
		res.ParseError = parsed.Message
		return res, nil
	}
	res.Parsed = parsed.Content
	return res, nil
}

func loadConfigInput(ctx context.Context, input string) (raw, kind, source string, err error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", "", fmt.Errorf("config is empty (path, URL, JSON, or share links)")
	}
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		urlLine := strings.Split(input, "\n")[0]
		urlLine = strings.TrimSpace(urlLine)
		body, err := fetchConfigURL(ctx, urlLine)
		if err != nil {
			return "", "url", urlLine, err
		}
		return body, "url", urlLine, nil
	}
	if looksLikePath(input) {
		path := input
		if strings.HasPrefix(path, "~/") {
			home, _ := os.UserHomeDir()
			path = filepath.Join(home, path[2:])
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", "path", path, err
		}
		return string(data), "path", path, nil
	}
	return input, "content", "inline", nil
}

func looksLikePath(s string) bool {
	if strings.ContainsAny(s, "\n\r") {
		return false
	}
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") || strings.HasPrefix(s, "~/") {
		return true
	}
	if _, err := os.Stat(s); err == nil {
		return true
	}
	return false
}

func fetchConfigURL(ctx context.Context, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent())
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("fetch %s: HTTP %d", rawURL, resp.StatusCode)
	}
	return string(body), nil
}

func (s *server) handleStop(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	out, err := s.client.Stop(ctx, &hcommon.Empty{})
	if err != nil {
		httpError(w, 500, err)
		return
	}
	s.writeProtoMethod(w, "Stop", out)
}

func (s *server) handleSelect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GroupTag    string `json:"groupTag"`
		OutboundTag string `json:"outboundTag"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, 400, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	out, err := s.client.SelectOutbound(ctx, &hcore.SelectOutboundRequest{
		GroupTag:    req.GroupTag,
		OutboundTag: req.OutboundTag,
	})
	if err != nil {
		httpError(w, 500, err)
		return
	}
	s.writeProtoMethod(w, "SelectOutbound", out)
}

func (s *server) handleUrlTestActive(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	out, err := s.client.UrlTestActive(ctx, &hcommon.Empty{})
	if err != nil {
		httpError(w, 500, err)
		return
	}
	s.writeProtoMethod(w, "UrlTestActive", out)
}

func (s *server) handleUrlTest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Tag string `json:"tag"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		httpError(w, 400, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	out, err := s.client.UrlTest(ctx, &hcore.UrlTestRequest{Tag: req.Tag})
	if err != nil {
		httpError(w, 500, err)
		return
	}
	s.writeProtoMethod(w, "UrlTest", out)
}

func (s *server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.mu.Lock()
		defer s.mu.Unlock()
		writeJSON(w, s.settings)
	case http.MethodPost:
		var req uiSettings
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, 400, err)
			return
		}
		s.mu.Lock()
		if req.MixedPort > 0 {
			s.settings.MixedPort = req.MixedPort
			s.mixedPort = req.MixedPort
		}
		if req.ServiceMode != "" {
			s.settings.ServiceMode = req.ServiceMode
		}
		if req.Chain != "" {
			s.settings.Chain = req.Chain
		}
		normalizeServiceMode(&s.settings)
		settingsCopy := s.settings
		s.mu.Unlock()
		if err := s.applyHiddifySettings(); err != nil {
			httpError(w, 500, err)
			return
		}
		s.writeJSONMethod(w, "ChangeHiddifySettings", settingsCopy)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func normalizeServiceMode(st *uiSettings) {
	switch st.ServiceMode {
	case "vpn":
		st.EnableTun = true
		st.SetSystemProxy = false
	case "system-proxy":
		st.EnableTun = false
		st.SetSystemProxy = true
	default:
		st.ServiceMode = "proxy"
		st.EnableTun = false
		st.SetSystemProxy = false
	}
}

func (s *server) settingsJSON() (string, error) {
	s.mu.Lock()
	st := s.settings
	clashPort := s.clashAPIPort
	s.mu.Unlock()
	normalizeServiceMode(&st)
	if clashPort == 0 {
		clashPort = 16756
	}

	m := map[string]any{
		"mixed-port":         st.MixedPort,
		"tproxy-port":        0,
		"redirect-port":      0,
		"direct-port":        0,
		"enable-clash-api":   true,
		"clash-api-port":     clashPort,
		"log-level":          "warn",
		"enable-tun":         st.EnableTun,
		"set-system-proxy":   st.SetSystemProxy,
		"enable-tun-service": false,
		"chain-status":       "off",
	}
	switch st.Chain {
	case "extra_security_warp_auto":
		m["chain-status"] = "extra_security"
		m["extra-security"] = map[string]any{"mode": "warp_auto"}
	case "extra_security_warp":
		m["chain-status"] = "extra_security"
		m["extra-security"] = map[string]any{"mode": "warp"}
	case "extra_security_warp_masque":
		m["chain-status"] = "extra_security"
		m["extra-security"] = map[string]any{"mode": "warp_masque"}
	case "extra_security_warp_awg":
		m["chain-status"] = "extra_security"
		m["extra-security"] = map[string]any{"mode": "warp_awg"}
	case "extra_security_psiphon":
		m["chain-status"] = "extra_security"
		m["extra-security"] = map[string]any{"mode": "psiphon", "psiphon": map[string]any{"region": "AUTO"}}
	case "unblocker_warp_auto":
		m["chain-status"] = "unblocker"
		m["unblocker"] = map[string]any{"mode": "warp_auto"}
	case "unblocker_warp":
		m["chain-status"] = "unblocker"
		m["unblocker"] = map[string]any{"mode": "warp"}
	case "unblocker_warp_masque":
		m["chain-status"] = "unblocker"
		m["unblocker"] = map[string]any{"mode": "warp_masque"}
	case "unblocker_warp_awg":
		m["chain-status"] = "unblocker"
		m["unblocker"] = map[string]any{"mode": "warp_awg"}
	case "unblocker_psiphon":
		m["chain-status"] = "unblocker"
		m["unblocker"] = map[string]any{"mode": "psiphon", "psiphon": map[string]any{"region": "AUTO"}}
	default:
		m["chain-status"] = "off"
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *server) applyHiddifySettings() error {
	js, err := s.settingsJSON()
	if err != nil {
		return err
	}
	_, err = hcore.ChangeHiddifySettings(&hcore.ChangeHiddifySettingsRequest{HiddifySettingsJson: js}, false)
	return err
}

// needAdminBgCheck asks NeedAdminBgService. blocked is true when this harness must not
// start/restart in-process (admin / already-connected / optional-proxy-need-admin).
func (s *server) needAdminBgCheck(ctx context.Context) (resp *hcore.NeedAdminBgServiceResponse, blocked bool, err error) {
	js, err := s.settingsJSON()
	if err != nil {
		return nil, false, err
	}
	s.mu.Lock()
	mode := s.settings.ServiceMode
	s.mu.Unlock()
	resp, err = s.client.NeedAdminBgService(ctx, &hcore.NeedAdminBgServiceRequest{
		ServiceMode:         mode,
		HiddifySettingsJson: js,
		AdminPreference:     hcore.AdminPreference_ADMIN_PREFERENCE_UNSET,
	})
	if err != nil {
		return nil, false, err
	}
	switch resp.GetDecision() {
	case hcore.AdminBgServiceDecision_ADMIN_BG_SERVICE_NO:
		return resp, false, nil
	default:
		// NEEDS_ADMIN / ALREADY_CONNECTED / OPTIONAL_PROXY_NEED_ADMIN: show and stop (no Ensure)
		return resp, true, nil
	}
}

func (s *server) writeBlockedByAdmin(w http.ResponseWriter, need *hcore.NeedAdminBgServiceResponse) {
	s.writeJSONMethod(w, "NeedAdminBgService", map[string]any{
		"blocked":   true,
		"reason":    "NeedAdminBgService",
		"needAdmin": protoToAny(need),
	})
}

func (s *server) resolveConfigBody(ctx context.Context, config string) (content string, resolved *resolveResult, err error) {
	if strings.TrimSpace(config) == "" {
		s.mu.Lock()
		content = s.lastConfig
		s.mu.Unlock()
		if content == "" {
			return "", nil, fmt.Errorf("no config (start once, or pass config)")
		}
		return content, nil, nil
	}
	resolved, err = s.resolveAndParse(ctx, config)
	if err != nil {
		return "", nil, err
	}
	content = resolved.Parsed
	if content == "" {
		content = resolved.Raw
	}
	return content, resolved, nil
}

func (s *server) handleRestart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Config string `json:"config"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()
	if err := s.applyHiddifySettings(); err != nil {
		httpError(w, 500, err)
		return
	}
	need, blocked, err := s.needAdminBgCheck(ctx)
	if err != nil {
		httpError(w, 500, err)
		return
	}
	if blocked {
		s.writeBlockedByAdmin(w, need)
		return
	}
	content, resolved, err := s.resolveConfigBody(ctx, req.Config)
	if err != nil {
		httpError(w, 500, err)
		return
	}
	out, err := s.client.Restart(ctx, &hcore.StartRequest{ConfigContent: content, ConfigName: "grpcweb"})
	if err != nil {
		httpError(w, 500, err)
		return
	}
	s.mu.Lock()
	s.lastConfig = content
	s.mu.Unlock()
	s.writeJSONMethod(w, "Restart", map[string]any{"restart": protoToAny(out), "resolved": resolved, "needAdmin": protoToAny(need)})
}

func (s *server) handleHotReload(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Config string `json:"config"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()
	if err := s.applyHiddifySettings(); err != nil {
		httpError(w, 500, err)
		return
	}
	need, blocked, err := s.needAdminBgCheck(ctx)
	if err != nil {
		httpError(w, 500, err)
		return
	}
	if blocked {
		s.writeBlockedByAdmin(w, need)
		return
	}
	content, resolved, err := s.resolveConfigBody(ctx, req.Config)
	if err != nil {
		httpError(w, 500, err)
		return
	}
	startReq := &hcore.StartRequest{ConfigContent: content, ConfigName: "grpcweb"}
	out, err := s.client.HotReload(ctx, startReq)
	if err == nil {
		s.mu.Lock()
		s.lastConfig = content
		s.mu.Unlock()
		s.writeJSONMethod(w, "HotReload", map[string]any{"hotReload": protoToAny(out), "resolved": resolved, "fallback": "none", "needAdmin": protoToAny(need)})
		return
	}
	// TUN / experimental / other immutable sections: fall back to a full restart.
	log.Printf("hot reload failed (%v); restarting", err)
	restartOut, restartErr := s.client.Restart(ctx, startReq)
	if restartErr != nil {
		httpError(w, 500, fmt.Errorf("hot reload failed (%w); restart also failed: %v", err, restartErr))
		return
	}
	s.mu.Lock()
	s.lastConfig = content
	s.mu.Unlock()
	s.writeJSONMethod(w, "Restart", map[string]any{
		"hotReloadError": err.Error(),
		"fallback":       "restart",
		"restart":        protoToAny(restartOut),
		"resolved":       resolved,
		"needAdmin":      protoToAny(need),
	})
}

func setSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	// flush headers immediately so the browser does not sit on a blank connection
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *server) streamCore(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", 500)
		return
	}
	setSSEHeaders(w)

	ch := s.subscribeCore()
	defer s.unsubscribeCore(ch)

	s.mu.Lock()
	if s.lastCore != nil {
		s.writeSSEMethod(w, "CoreInfoListener", s.lastCore)
		flusher.Flush()
	}
	s.mu.Unlock()

	for {
		select {
		case <-r.Context().Done():
			return
		case info := <-ch:
			if info == nil {
				continue
			}
			s.writeSSEMethod(w, "CoreInfoListener", info)
			flusher.Flush()
		}
	}
}

func (s *server) streamProxies(mainOnly bool) http.HandlerFunc {
	method := "OutboundsInfo"
	if mainOnly {
		method = "MainOutboundsInfo"
	}
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "stream unsupported", 500)
			return
		}
		setSSEHeaders(w)

		var last string
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				s.mu.Lock()
				var cur *hcore.OutboundGroupList
				if mainOnly {
					cur = s.lastMain
				} else {
					cur = s.lastGroups
				}
				s.mu.Unlock()
				if cur == nil {
					continue
				}
				b, _ := marshal.Marshal(cur)
				if string(b) == last {
					continue
				}
				last = string(b)
				s.noteRPC(method, len(b))
				fmt.Fprintf(w, "data: %s\n\n", b)
				flusher.Flush()
			}
		}
	}
}

func (s *server) watchSystemInfo(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		if !s.coreRunning() {
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}
		stream, err := s.client.GetSystemInfoStream(ctx, &hcommon.Empty{})
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		for {
			info, err := stream.Recv()
			if err != nil {
				break
			}
			s.mu.Lock()
			s.lastSystem = proto.Clone(info).(*hcore.SystemInfo)
			s.mu.Unlock()
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func (s *server) streamSystemInfo(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", 500)
		return
	}
	setSSEHeaders(w)
	var last string
	ticker := time.NewTicker(400 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			cur := s.lastSystem
			s.mu.Unlock()
			if cur == nil {
				continue
			}
			b, _ := marshal.Marshal(cur)
			if string(b) == last {
				continue
			}
			last = string(b)
			s.noteRPC("GetSystemInfoStream", len(b))
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		}
	}
}

func (s *server) streamLogs(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", 500)
		return
	}
	level := hcore.LogLevel_INFO
	if v := strings.TrimSpace(r.URL.Query().Get("level")); v != "" {
		if n, ok := hcore.LogLevel_value[strings.ToUpper(v)]; ok {
			level = hcore.LogLevel(n)
		}
	}
	setSSEHeaders(w)
	stream, err := s.client.LogListener(r.Context(), &hcore.LogRequest{Level: level})
	if err != nil {
		httpError(w, 500, err)
		return
	}
	for {
		msg, err := stream.Recv()
		if err != nil {
			return
		}
		s.writeSSEMethod(w, "LogListener", msg)
		flusher.Flush()
	}
}

func (s *server) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Method string          `json:"method"`
		Body   json.RawMessage `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, 400, err)
		return
	}
	s.noteRPC(req.Method, len(req.Body))
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	switch req.Method {
	case "GetSystemInfo":
		out, err := s.client.GetSystemInfo(ctx, &hcommon.Empty{})
		if err != nil {
			httpError(w, 500, err)
			return
		}
		s.writeJSONMethod(w, req.Method, protoToAny(out))
	case "GetSystemProxyStatus":
		out, err := s.client.GetSystemProxyStatus(ctx, &hcommon.Empty{})
		if err != nil {
			httpError(w, 500, err)
			return
		}
		s.writeJSONMethod(w, req.Method, protoToAny(out))
	case "GetLANIP":
		out, err := s.client.GetLANIP(ctx, &hcommon.Empty{})
		if err != nil {
			httpError(w, 500, err)
			return
		}
		s.writeJSONMethod(w, req.Method, protoToAny(out))
	case "UrlTestActive":
		out, err := s.client.UrlTestActive(ctx, &hcommon.Empty{})
		if err != nil {
			httpError(w, 500, err)
			return
		}
		s.writeJSONMethod(w, req.Method, protoToAny(out))
	case "Stop":
		out, err := s.client.Stop(ctx, &hcommon.Empty{})
		if err != nil {
			httpError(w, 500, err)
			return
		}
		s.writeJSONMethod(w, req.Method, protoToAny(out))
	case "Start", "StartService", "Restart", "HotReload":
		var body struct {
			Config string `json:"config"`
		}
		_ = json.Unmarshal(req.Body, &body)
		if err := s.applyHiddifySettings(); err != nil {
			httpError(w, 500, err)
			return
		}
		need, blocked, err := s.needAdminBgCheck(ctx)
		if err != nil {
			httpError(w, 500, err)
			return
		}
		if blocked {
			s.writeBlockedByAdmin(w, need)
			return
		}
		content, resolved, err := s.resolveConfigBody(ctx, body.Config)
		if err != nil {
			httpError(w, 500, err)
			return
		}
		startReq := &hcore.StartRequest{ConfigContent: content, ConfigName: "grpcweb"}
		var out proto.Message
		switch req.Method {
		case "Start":
			out, err = s.client.Start(ctx, startReq)
		case "StartService":
			out, err = s.client.StartService(ctx, startReq)
		case "Restart":
			out, err = s.client.Restart(ctx, startReq)
		case "HotReload":
			out, err = s.client.HotReload(ctx, startReq)
			if err != nil {
				restartOut, restartErr := s.client.Restart(ctx, startReq)
				if restartErr != nil {
					httpError(w, 500, fmt.Errorf("hot reload failed (%w); restart also failed: %v", err, restartErr))
					return
				}
				s.mu.Lock()
				s.lastConfig = content
				s.mu.Unlock()
				s.writeJSONMethod(w, "Restart", map[string]any{
					"hotReloadError": err.Error(),
					"fallback":       "restart",
					"restart":        protoToAny(restartOut),
					"resolved":       resolved,
					"needAdmin":      protoToAny(need),
				})
				return
			}
		}
		if err != nil {
			httpError(w, 500, err)
			return
		}
		s.mu.Lock()
		s.lastConfig = content
		s.mu.Unlock()
		s.writeJSONMethod(w, req.Method, map[string]any{
			"result":    protoToAny(out),
			"resolved":  resolved,
			"needAdmin": protoToAny(need),
		})
	case "SelectOutbound":
		var body struct {
			GroupTag    string `json:"groupTag"`
			OutboundTag string `json:"outboundTag"`
		}
		_ = json.Unmarshal(req.Body, &body)
		out, err := s.client.SelectOutbound(ctx, &hcore.SelectOutboundRequest{
			GroupTag: body.GroupTag, OutboundTag: body.OutboundTag,
		})
		if err != nil {
			httpError(w, 500, err)
			return
		}
		s.writeJSONMethod(w, req.Method, protoToAny(out))
	case "UrlTest":
		var body struct {
			Tag string `json:"tag"`
		}
		_ = json.Unmarshal(req.Body, &body)
		out, err := s.client.UrlTest(ctx, &hcore.UrlTestRequest{Tag: body.Tag})
		if err != nil {
			httpError(w, 500, err)
			return
		}
		s.writeJSONMethod(w, req.Method, protoToAny(out))
	case "SetSystemProxyEnabled":
		var body hcore.SetSystemProxyEnabledRequest
		if len(req.Body) > 0 {
			_ = json.Unmarshal(req.Body, &body)
		}
		out, err := s.client.SetSystemProxyEnabled(ctx, &body)
		if err != nil {
			httpError(w, 500, err)
			return
		}
		s.writeJSONMethod(w, req.Method, protoToAny(out))
	case "GenerateWarpConfig":
		var body hcore.GenerateWarpConfigRequest
		if len(req.Body) > 0 {
			_ = json.Unmarshal(req.Body, &body)
		}
		out, err := s.client.GenerateWarpConfig(ctx, &body)
		if err != nil {
			httpError(w, 500, err)
			return
		}
		s.writeJSONMethod(w, req.Method, protoToAny(out))
	case "NeedAdminBgService":
		need, _, err := s.needAdminBgCheck(ctx)
		if err != nil {
			httpError(w, 500, err)
			return
		}
		s.writeJSONMethod(w, req.Method, protoToAny(need))
	case "Parse":
		var body struct {
			Content string `json:"content"`
			Debug   bool   `json:"debug"`
		}
		_ = json.Unmarshal(req.Body, &body)
		if body.Content == "" {
			s.mu.Lock()
			body.Content = s.lastConfig
			s.mu.Unlock()
		}
		out, err := s.client.Parse(ctx, &hcore.ParseRequest{Content: body.Content, Debug: body.Debug})
		if err != nil {
			httpError(w, 500, err)
			return
		}
		s.writeJSONMethod(w, req.Method, protoToAny(out))
	case "Setup":
		s.writeJSONMethod(w, req.Method, map[string]any{
			"skipped": true,
			"message": "Setup already ran at harness boot; pass body only for inspection",
			"body":    json.RawMessage(req.Body),
		})
	case "Close":
		var body hcore.CloseRequest
		if len(req.Body) > 0 {
			_ = json.Unmarshal(req.Body, &body)
		}
		out, err := s.client.Close(ctx, &body)
		if err != nil {
			httpError(w, 500, err)
			return
		}
		s.writeJSONMethod(w, req.Method, protoToAny(out))
	case "ChangeHiddifySettings":
		if err := s.applyHiddifySettings(); err != nil {
			httpError(w, 500, err)
			return
		}
		s.writeJSONMethod(w, req.Method, map[string]any{"ok": true, "settings": s.settings})
	case "EnsureAdminBgService":
		s.writeJSONMethod(w, req.Method, map[string]any{
			"blocked": true,
			"message": "EnsureAdminBgService is not run from this harness (see NeedAdminBgService)",
		})
	default:
		httpError(w, 400, fmt.Errorf("unknown method %q", req.Method))
	}
}

func writeProto(w http.ResponseWriter, m proto.Message) {
	b, err := marshal.Marshal(m)
	if err != nil {
		httpError(w, 500, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func (s *server) writeProtoMethod(w http.ResponseWriter, method string, m proto.Message) {
	b, err := marshal.Marshal(m)
	if err != nil {
		httpError(w, 500, err)
		return
	}
	s.noteRPC(method, len(b))
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func protoToAny(m proto.Message) any {
	if m == nil {
		return nil
	}
	b, err := marshal.Marshal(m)
	if err != nil {
		return nil
	}
	var v any
	_ = json.Unmarshal(b, &v)
	return v
}

func httpError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		// Chrome / Cursor Simple Browser private-network access preflight
		w.Header().Set("Access-Control-Allow-Private-Network", "true")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
