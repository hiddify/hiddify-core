package hcore

import (
	"compress/gzip"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func fetchURLTestServer(t *testing.T, http2 bool) *httptest.Server {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "/sub", http.StatusFound)
		case "/sub":
			w.Header().Set("Subscription-Userinfo", "upload=1; download=2")
			w.Header().Add("X-Proto", r.Proto)
			w.Header().Add("X-Ua", r.UserAgent())
			if r.Header.Get("Accept-Encoding") == "gzip" {
				w.Header().Set("Content-Encoding", "gzip")
				zw := gzip.NewWriter(w)
				zw.Write([]byte("vless://a"))
				zw.Close()
				return
			}
			w.Write([]byte("vless://a"))
		default:
			http.NotFound(w, r)
		}
	}))
	server.EnableHTTP2 = http2
	server.StartTLS()
	t.Cleanup(server.Close)
	old := fetchURLTransport
	fetchURLTransport = chromeTransport{rootCAs: server.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs}
	t.Cleanup(func() { fetchURLTransport = old })
	return server
}

func TestFetchUrl(t *testing.T) {
	for _, tc := range []struct {
		name  string
		http2 bool
		proto string
	}{{"http2", true, "HTTP/2.0"}, {"http1", false, "HTTP/1.1"}} {
		t.Run(tc.name, func(t *testing.T) {
			server := fetchURLTestServer(t, tc.http2)
			response, err := (&CoreService{}).FetchUrl(context.Background(), &FetchUrlRequest{Url: server.URL + "/redirect", UserAgent: "HiddifyNext/1"})
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != 200 || string(response.Body) != "vless://a" {
				t.Fatalf("got %d %q", response.StatusCode, response.Body)
			}
			headers := map[string]string{}
			for _, h := range response.Headers {
				headers[h.Name] = h.Value
			}
			if headers["x-proto"] != tc.proto || headers["x-ua"] != "HiddifyNext/1" || headers["subscription-userinfo"] == "" {
				t.Fatalf("headers %v", headers)
			}
			if _, ok := headers["content-encoding"]; ok {
				t.Fatalf("content-encoding left in %v", headers)
			}
		})
	}
}

func TestFetchUrlNotFound(t *testing.T) {
	server := fetchURLTestServer(t, true)
	response, err := (&CoreService{}).FetchUrl(context.Background(), &FetchUrlRequest{Url: server.URL + "/missing"})
	if err != nil || response.StatusCode != 404 {
		t.Fatalf("got %v %v", response, err)
	}
}

func TestFetchUrlBadScheme(t *testing.T) {
	if _, err := (&CoreService{}).FetchUrl(context.Background(), &FetchUrlRequest{Url: "ftp://x/y"}); err == nil {
		t.Fatal("no error")
	}
}

func TestFetchUrlErrors(t *testing.T) {
	// accepts, reads the ClientHello, then resets the connection, as a filter does
	resetter, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer resetter.Close()
	go func() {
		for {
			conn, err := resetter.Accept()
			if err != nil {
				return
			}
			conn.Read(make([]byte, 512))
			conn.(*net.TCPConn).SetLinger(0)
			conn.Close()
		}
	}()
	// accepts and never answers
	silent, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer silent.Close()
	var heldLock sync.Mutex
	var held []net.Conn
	defer func() {
		heldLock.Lock()
		defer heldLock.Unlock()
		for _, c := range held {
			c.Close()
		}
	}()
	go func() {
		for {
			conn, err := silent.Accept()
			if err != nil {
				return
			}
			heldLock.Lock()
			held = append(held, conn)
			heldLock.Unlock()
		}
	}()
	closed, _ := net.Listen("tcp", "127.0.0.1:0")
	closedAddr := closed.Addr().String()
	closed.Close()
	untrusted := httptest.NewTLSServer(http.NotFoundHandler())
	defer untrusted.Close()

	for _, tc := range []struct {
		name string
		url  string
		want FetchUrlError
	}{
		{"refused", "https://" + closedAddr + "/sub", FetchUrlError_FETCH_URL_ERROR_REFUSED},
		{"dns", "https://no-such-host.invalid/sub", FetchUrlError_FETCH_URL_ERROR_DNS},
		{"reset", "https://" + resetter.Addr().String() + "/sub", FetchUrlError_FETCH_URL_ERROR_RESET},
		{"timeout", "https://" + silent.Addr().String() + "/sub", FetchUrlError_FETCH_URL_ERROR_TIMEOUT},
		{"certificate", untrusted.URL + "/sub", FetchUrlError_FETCH_URL_ERROR_CERTIFICATE},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response, err := (&CoreService{}).FetchUrl(context.Background(), &FetchUrlRequest{Url: tc.url + "?token=secret", TimeoutMs: 500})
			if err != nil {
				t.Fatal(err)
			}
			if response.Error != tc.want {
				t.Fatalf("got %v (%s), want %v", response.Error, response.ErrorDetail, tc.want)
			}
			if strings.Contains(response.ErrorDetail, "secret") {
				t.Fatalf("detail carries the URL: %s", response.ErrorDetail)
			}
		})
	}
}

func TestFetchUrlCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (&CoreService{}).FetchUrl(ctx, &FetchUrlRequest{Url: "https://example.com/sub"}); status.Code(err) != codes.Canceled {
		t.Fatalf("got %v", err)
	}
}
