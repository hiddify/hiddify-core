package hcore

import (
	"bufio"
	"compress/gzip"
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	fetchURLTimeout = 30 * time.Second
	fetchURLBodyMax = 16 << 20
)

// FetchUrl downloads a URL with the TLS handshake of Chrome, which the app tries before its own
// download: some networks reset TLS clients they do not take for a browser, Dart's among them. A
// failed download is a response that says why, for the app to tell the user what to try.
func (s *CoreService) FetchUrl(ctx context.Context, req *FetchUrlRequest) (*FetchUrlResponse, error) {
	timeout := fetchURLTimeout
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, req.Url, nil)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if request.URL.Scheme != "http" && request.URL.Scheme != "https" {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported scheme %q", request.URL.Scheme)
	}
	if req.UserAgent != "" {
		request.Header.Set("User-Agent", req.UserAgent)
	}
	request.Header.Set("Accept", "*/*")
	request.Header.Set("Accept-Language", "en-US,en;q=0.9")

	client := &http.Client{Transport: fetchURLTransport}
	response, err := client.Do(request)
	if err != nil {
		return fetchURLFailure(ctx, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, fetchURLBodyMax+1))
	if err != nil {
		return fetchURLFailure(ctx, err)
	}
	if len(body) > fetchURLBodyMax {
		return &FetchUrlResponse{
			Error:       FetchUrlError_FETCH_URL_ERROR_TOO_LARGE,
			ErrorDetail: fmt.Sprintf("body over %d bytes", fetchURLBodyMax),
		}, nil
	}

	result := &FetchUrlResponse{StatusCode: int32(response.StatusCode), Body: body}
	for name, values := range response.Header {
		for _, value := range values {
			result.Headers = append(result.Headers, &HttpHeader{Name: strings.ToLower(name), Value: value})
		}
	}
	return result, nil
}

var fetchURLTransport http.RoundTripper = chromeTransport{}

// chromeTransport sends each request on a connection of its own: https with the ClientHello of
// Chrome, then HTTP/2 or HTTP/1.1 as the server picks by ALPN, the way Chrome would.
type chromeTransport struct {
	// nil for the system's
	rootCAs *x509.CertPool
}

func (t chromeTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	ctx := request.Context()
	host := request.URL.Hostname()
	port := request.URL.Port()
	if port == "" {
		port = map[string]string{"http": "80", "https": "443"}[request.URL.Scheme]
	}

	dialer := net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		return nil, err
	}
	// the handshake and the request end with the request's context
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	fail := func(err error) (*http.Response, error) {
		stop()
		conn.Close()
		return nil, err
	}

	if request.URL.Scheme == "https" {
		tlsConn := utls.UClient(conn, &utls.Config{ServerName: host, RootCAs: t.rootCAs}, utls.HelloChrome_Auto)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			return fail(&tlsHandshakeError{err})
		}
		conn = tlsConn
		if tlsConn.ConnectionState().NegotiatedProtocol == http2.NextProtoTLS {
			clientConn, err := (&http2.Transport{}).NewClientConn(conn)
			if err != nil {
				return fail(err)
			}
			response, err := clientConn.RoundTrip(request)
			if err != nil {
				clientConn.Close()
				return fail(err)
			}
			response.Body = &closeWithBody{ReadCloser: response.Body, close: func() { stop(); clientConn.Close(); conn.Close() }}
			return response, nil
		}
	}

	// HTTP/1.1: one request, then the connection closes; gzip is asked for and undone here, as
	// http.Transport would
	request = request.Clone(ctx)
	request.Close = true
	request.Header.Set("Accept-Encoding", "gzip")
	if err := request.Write(conn); err != nil {
		return fail(err)
	}
	response, err := http.ReadResponse(bufio.NewReader(conn), request)
	if err != nil {
		return fail(err)
	}
	closeConn := func() { stop(); conn.Close() }
	if strings.EqualFold(response.Header.Get("Content-Encoding"), "gzip") && request.Method != http.MethodHead {
		compressed := response.Body
		reader, err := gzip.NewReader(compressed)
		if err != nil && !errors.Is(err, io.EOF) {
			compressed.Close()
			return fail(err)
		}
		// an empty body has no gzip header and is left as it is
		if reader != nil {
			response.Body = &closeWithBody{ReadCloser: reader, close: func() { compressed.Close(); closeConn() }}
			response.Header.Del("Content-Encoding")
			response.Header.Del("Content-Length")
			response.ContentLength = -1
			response.Uncompressed = true
			return response, nil
		}
	}
	response.Body = &closeWithBody{ReadCloser: response.Body, close: closeConn}
	return response, nil
}

type tlsHandshakeError struct{ err error }

func (e *tlsHandshakeError) Error() string { return "tls handshake: " + e.err.Error() }
func (e *tlsHandshakeError) Unwrap() error { return e.err }

// fetchURLFailure is the response for a download that failed, or the gRPC error of one the app
// cancelled.
func fetchURLFailure(ctx context.Context, err error) (*FetchUrlResponse, error) {
	if errors.Is(ctx.Err(), context.Canceled) {
		return nil, status.Error(codes.Canceled, "cancelled")
	}
	// the URL is left out: it carries the user's token
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	return &FetchUrlResponse{Error: classifyFetchURLError(ctx, err), ErrorDetail: err.Error()}, nil
}

func classifyFetchURLError(ctx context.Context, err error) FetchUrlError {
	var dnsErr *net.DNSError
	var handshakeErr *tlsHandshakeError
	var netErr net.Error
	text := strings.ToLower(err.Error())
	switch {
	case errors.As(err, &dnsErr):
		return FetchUrlError_FETCH_URL_ERROR_DNS
	case isCertificateError(err):
		return FetchUrlError_FETCH_URL_ERROR_CERTIFICATE
	case errors.Is(err, syscall.ENETUNREACH) || errors.Is(err, syscall.EHOSTUNREACH) ||
		strings.Contains(text, "network is unreachable") || strings.Contains(text, "no route to host"):
		return FetchUrlError_FETCH_URL_ERROR_UNREACHABLE
	case errors.Is(err, syscall.ECONNREFUSED) || strings.Contains(text, "refused"):
		return FetchUrlError_FETCH_URL_ERROR_REFUSED
	case errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNABORTED) ||
		strings.Contains(text, "reset by peer") || strings.Contains(text, "forcibly closed") ||
		errors.As(err, &handshakeErr) && (errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)):
		return FetchUrlError_FETCH_URL_ERROR_RESET
	case errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.As(err, &netErr) && netErr.Timeout():
		return FetchUrlError_FETCH_URL_ERROR_TIMEOUT
	case errors.As(err, &handshakeErr):
		return FetchUrlError_FETCH_URL_ERROR_TLS
	}
	return FetchUrlError_FETCH_URL_ERROR_OTHER
}

func isCertificateError(err error) bool {
	var unknownAuthority x509.UnknownAuthorityError
	var hostname x509.HostnameError
	var invalid x509.CertificateInvalidError
	var verification *utls.CertificateVerificationError
	return errors.As(err, &unknownAuthority) || errors.As(err, &hostname) || errors.As(err, &invalid) ||
		errors.As(err, &verification)
}

type closeWithBody struct {
	io.ReadCloser
	close func()
}

func (c *closeWithBody) Close() error {
	err := c.ReadCloser.Close()
	c.close()
	return err
}
