package config

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/hiddify/hiddify-core/v2/db"
	sbtls "github.com/sagernet/sing-box/common/tls"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
)

// The system's TLS is used only after it worked on this device: a TLS connection through it to a TLS server in
// this process, with a little data each way. The check runs in the background when there is no result or the
// last one is more than a month old (a system update may change the engine), and its result is kept in the
// database. Until a check passed, the TLS clients get Chrome's uTLS instead.

const tlsEngineCheckInterval = 30 * 24 * time.Hour

// TLSEngineCheck is the last check of a system TLS engine on this device.
type TLSEngineCheck struct {
	Id        string // the engine: apple or windows
	Supported bool
	CheckedAt time.Time
	Error     string
}

var tlsEngineChecking atomic.Bool

// usableTLSEngine is usableSystemTLSEngine; tests replace it.
var usableTLSEngine = usableSystemTLSEngine

// usableSystemTLSEngine is the system's TLS engine when its last check passed, or "". It starts a check when
// there is none or it is old; a passed result is used meanwhile.
func usableSystemTLSEngine() string {
	engine := systemTLSEngine()
	if engine == "" {
		return ""
	}
	check, err := db.GetTable[TLSEngineCheck]().Get(engine)
	if err != nil || check == nil || time.Since(check.CheckedAt) > tlsEngineCheckInterval {
		startTLSEngineCheck(engine)
	}
	if err == nil && check != nil && check.Supported {
		return engine
	}
	return ""
}

func startTLSEngineCheck(engine string) {
	if !tlsEngineChecking.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer tlsEngineChecking.Store(false)
		result := &TLSEngineCheck{Id: engine, CheckedAt: time.Now()}
		if err := checkTLSEngine(engine); err != nil {
			result.Error = err.Error()
		} else {
			result.Supported = true
		}
		db.GetTable[TLSEngineCheck]().UpdateInsert(result)
	}()
}

// checkTLSEngine connects with the engine to a TLS server in this process and exchanges a few bytes.
func checkTLSEngine(engine string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	keyPEM, certPEM, err := sbtls.GenerateCertificate(nil, nil, time.Now, "example.com", time.Now().Add(time.Hour))
	if err != nil {
		return err
	}
	certificate, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return err
	}
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{certificate}})
	if err != nil {
		return err
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(10 * time.Second))
		request := make([]byte, 4)
		if _, err := io.ReadFull(conn, request); err == nil && string(request) == "ping" {
			conn.Write([]byte("pong"))
		}
	}()
	config, err := sbtls.NewClient(ctx, log.NewNOPFactory().Logger(), "example.com", option.OutboundTLSOptions{
		Enabled:     true,
		Engine:      engine,
		ServerName:  "example.com",
		Certificate: []string{string(certPEM)},
	})
	if err != nil {
		return err
	}
	tcpConn, err := (&net.Dialer{}).DialContext(ctx, "tcp", listener.Addr().String())
	if err != nil {
		return err
	}
	defer tcpConn.Close()
	tcpConn.SetDeadline(time.Now().Add(10 * time.Second))
	conn, err := sbtls.ClientHandshake(ctx, tcpConn, config)
	if err != nil {
		return fmt.Errorf("handshake: %w", err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("ping")); err != nil {
		return err
	}
	reply := make([]byte, 4)
	if _, err := io.ReadFull(conn, reply); err != nil || string(reply) != "pong" {
		return fmt.Errorf("no reply through the %s TLS engine: %v", engine, err)
	}
	return nil
}
