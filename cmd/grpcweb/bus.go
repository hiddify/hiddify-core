package grpcweb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/hiddify/hiddify-core/v2/hcore"
	"google.golang.org/protobuf/proto"
)

// Bidirectional multiplex bus:
//
//	client → {"op":"sub"|"unsub","method":"LogListener","params":{"level":"INFO"}}
//	server → {"method":"CoreInfoListener","data":{...}}
//
// One WebSocket per page. Always-on feeds: CoreInfoListener, OutboundsInfo, GrpcStats.
// Optional: MainOutboundsInfo, GetSystemInfoStream, LogListener (via sub).

type busEnvelope struct {
	Method string          `json:"method,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
	Error  string          `json:"error,omitempty"`
}

type busCommand struct {
	Op     string          `json:"op"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type busSession struct {
	s      *server
	conn   *websocket.Conn
	ctx    context.Context
	cancel context.CancelFunc

	mu   sync.Mutex
	subs map[string]context.CancelFunc // method → cancel feeder
	out  chan []byte
}

func (s *server) handleBus(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // local harness; same-origin UI
	})
	if err != nil {
		log.Printf("bus accept: %v", err)
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	sess := &busSession{
		s:      s,
		conn:   c,
		ctx:    ctx,
		cancel: cancel,
		subs:   make(map[string]context.CancelFunc),
		out:    make(chan []byte, 256),
	}
	defer func() {
		cancel()
		sess.stopAll()
		_ = c.Close(websocket.StatusNormalClosure, "")
	}()

	go sess.writer()

	// Always-on channels for the harness UI.
	_ = sess.startSub("CoreInfoListener", nil)
	_ = sess.startSub("OutboundsInfo", nil)
	_ = sess.startSub("GrpcStats", nil)

	_ = sess.sendJSON(busEnvelope{Method: "__hello", Data: json.RawMessage(`{"ok":true}`)})

	for {
		var cmd busCommand
		if err := wsjson.Read(ctx, c, &cmd); err != nil {
			return
		}
		cmd.Method = strings.TrimSpace(cmd.Method)
		switch strings.ToLower(strings.TrimSpace(cmd.Op)) {
		case "sub", "subscribe":
			if err := sess.startSub(cmd.Method, cmd.Params); err != nil {
				_ = sess.sendJSON(busEnvelope{Method: cmd.Method, Error: err.Error()})
			}
		case "unsub", "unsubscribe":
			sess.stopSub(cmd.Method)
		case "ping":
			_ = sess.sendJSON(busEnvelope{Method: "__pong", Data: json.RawMessage(`{}`)})
		default:
			_ = sess.sendJSON(busEnvelope{Error: "unknown op: " + cmd.Op})
		}
	}
}

func (sess *busSession) writer() {
	for {
		select {
		case <-sess.ctx.Done():
			return
		case b := <-sess.out:
			wctx, cancel := context.WithTimeout(sess.ctx, 5*time.Second)
			err := sess.conn.Write(wctx, websocket.MessageText, b)
			cancel()
			if err != nil {
				sess.cancel()
				return
			}
		}
	}
}

func (sess *busSession) sendJSON(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	select {
	case sess.out <- b:
		return nil
	case <-sess.ctx.Done():
		return sess.ctx.Err()
	default:
		// buffer full: drop oldest, keep the newest (never silently skip current)
		select {
		case <-sess.out:
		default:
		}
		select {
		case sess.out <- b:
			return nil
		case <-sess.ctx.Done():
			return sess.ctx.Err()
		default:
			return nil
		}
	}
}

func (sess *busSession) sendMethod(method string, m proto.Message) {
	b, err := marshal.Marshal(m)
	if err != nil {
		return
	}
	sess.s.noteRPC(method, len(b))
	_ = sess.sendJSON(busEnvelope{Method: method, Data: json.RawMessage(b)})
}

func (sess *busSession) sendMethodRaw(method string, raw []byte) {
	sess.s.noteRPC(method, len(raw))
	_ = sess.sendJSON(busEnvelope{Method: method, Data: json.RawMessage(raw)})
}

func (sess *busSession) stopAll() {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	for m, cancel := range sess.subs {
		cancel()
		delete(sess.subs, m)
	}
}

func (sess *busSession) stopSub(method string) {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if cancel, ok := sess.subs[method]; ok {
		cancel()
		delete(sess.subs, method)
	}
}

func (sess *busSession) startSub(method string, params json.RawMessage) error {
	if method == "" {
		return fmt.Errorf("method required")
	}
	sess.mu.Lock()
	if cancel, ok := sess.subs[method]; ok {
		cancel()
		delete(sess.subs, method)
	}
	ctx, cancel := context.WithCancel(sess.ctx)
	sess.subs[method] = cancel
	sess.mu.Unlock()

	switch method {
	case "CoreInfoListener":
		go sess.feedCore(ctx)
	case "OutboundsInfo":
		go sess.feedProxies(ctx, false)
	case "MainOutboundsInfo":
		go sess.feedProxies(ctx, true)
	case "GetSystemInfoStream":
		go sess.feedSystem(ctx)
	case "LogListener":
		go sess.feedLogs(ctx, params)
	case "GrpcStats":
		go sess.feedStats(ctx)
	default:
		cancel()
		sess.mu.Lock()
		delete(sess.subs, method)
		sess.mu.Unlock()
		return fmt.Errorf("unsupported bus method: %s", method)
	}
	return nil
}

func (sess *busSession) feedCore(ctx context.Context) {
	ch := sess.s.subscribeCore()
	defer sess.s.unsubscribeCore(ch)

	sess.s.mu.Lock()
	if sess.s.lastCore != nil {
		sess.sendMethod("CoreInfoListener", sess.s.lastCore)
	}
	sess.s.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return
		case info := <-ch:
			if info == nil {
				continue
			}
			sess.sendMethod("CoreInfoListener", info)
		}
	}
}

func (sess *busSession) feedProxies(ctx context.Context, mainOnly bool) {
	method := "OutboundsInfo"
	if mainOnly {
		method = "MainOutboundsInfo"
	}
	var last string
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sess.s.mu.Lock()
			var cur *hcore.OutboundGroupList
			if mainOnly {
				cur = sess.s.lastMain
			} else {
				cur = sess.s.lastGroups
			}
			sess.s.mu.Unlock()
			if cur == nil {
				continue
			}
			b, err := marshal.Marshal(cur)
			if err != nil {
				continue
			}
			if string(b) == last {
				continue
			}
			last = string(b)
			sess.sendMethodRaw(method, b)
		}
	}
}

func (sess *busSession) feedSystem(ctx context.Context) {
	var last string
	ticker := time.NewTicker(400 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sess.s.mu.Lock()
			cur := sess.s.lastSystem
			sess.s.mu.Unlock()
			if cur == nil {
				continue
			}
			b, err := marshal.Marshal(cur)
			if err != nil {
				continue
			}
			if string(b) == last {
				continue
			}
			last = string(b)
			sess.sendMethodRaw("GetSystemInfoStream", b)
		}
	}
}

func (sess *busSession) feedLogs(ctx context.Context, params json.RawMessage) {
	level := hcore.LogLevel_INFO
	if len(params) > 0 {
		var p struct {
			Level string `json:"level"`
		}
		if json.Unmarshal(params, &p) == nil && p.Level != "" {
			if n, ok := hcore.LogLevel_value[strings.ToUpper(p.Level)]; ok {
				level = hcore.LogLevel(n)
			}
		}
	}
	stream, err := sess.s.client.LogListener(ctx, &hcore.LogRequest{Level: level})
	if err != nil {
		_ = sess.sendJSON(busEnvelope{Method: "LogListener", Error: err.Error()})
		return
	}
	for {
		msg, err := stream.Recv()
		if err != nil {
			return
		}
		sess.sendMethod("LogListener", msg)
	}
}

func (sess *busSession) feedStats(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	var last string
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b, err := json.Marshal(sess.s.grpcStatsSnapshot())
			if err != nil {
				continue
			}
			if string(b) == last {
				continue
			}
			last = string(b)
			_ = sess.sendJSON(busEnvelope{Method: "GrpcStats", Data: json.RawMessage(b)})
		}
	}
}
