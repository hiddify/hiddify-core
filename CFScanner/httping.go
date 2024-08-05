package CFScanner

import (
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// pingReceived pingTotalTime
func (p *Ping) httping(ip *net.IPAddr) (int, time.Duration) {
	hc := http.Client{
		Timeout: time.Second * 2,
		Transport: &http.Transport{
			DialContext: getDialContext(ip),
			//TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Skip certificate verification
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Prevent redirection
		},
	}

	// Access once to get the HTTP status code and Cloudflare Colo
	{
		requ, err := http.NewRequest(http.MethodHead, URL, nil)
		if err != nil {
			return 0, 0
		}
		requ.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36")
		resp, err := hc.Do(requ)
		if err != nil {
			return 0, 0
		}
		defer resp.Body.Close()

		//fmt.Println("IP:", ip, "StatusCode:", resp.StatusCode, resp.Request.URL)
		// If no HTTP status code is specified or the specified status code is invalid, only consider 200, 301, and 302 as successful HTTPing
		if HttpingStatusCode == 0 || (HttpingStatusCode < 100 && HttpingStatusCode > 599) {
			if resp.StatusCode != 200 && resp.StatusCode != 301 && resp.StatusCode != 302 {
				return 0, 0
			}
		} else {
			if resp.StatusCode != HttpingStatusCode {
				return 0, 0
			}
		}

		io.Copy(io.Discard, resp.Body)

		// Match airport code only if the region is specified
		if HttpingCFColo != "" {
			// Determine whether it is Cloudflare or AWS CloudFront based on the Server header value and set cfRay to their respective airport code
			cfRay := func() string {
				if resp.Header.Get("Server") == "cloudflare" {
					return resp.Header.Get("CF-RAY") // Example cf-ray: 7bd32409eda7b020-SJC
				}
				return resp.Header.Get("x-amz-cf-pop") // Example X-Amz-Cf-Pop: SIN52-P1
			}()
			colo := p.getColo(cfRay)
			if colo == "" { // If no matching airport code or does not match the specified region, end the IP test directly
				return 0, 0
			}
		}

	}

	// Loop to calculate latency
	success := 0
	var delay time.Duration
	for i := 0; i < PingTimes; i++ {
		requ, err := http.NewRequest(http.MethodHead, URL, nil)
		if err != nil {
			log.Fatal("Unexpected error, please report:", err)
			return 0, 0
		}
		requ.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36")
		if i == PingTimes-1 {
			requ.Header.Set("Connection", "close")
		}
		startTime := time.Now()
		resp, err := hc.Do(requ)
		if err != nil {
			continue
		}
		success++
		io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		duration := time.Since(startTime)
		delay += duration

	}

	return success, delay

}

func MapColoMap() *sync.Map {
	if HttpingCFColo == "" {
		return nil
	}
	// Convert the specified region codes to uppercase and format them
	colos := strings.Split(strings.ToUpper(HttpingCFColo), ",")
	colomap := &sync.Map{}
	for _, colo := range colos {
		colomap.Store(colo, colo)
	}
	return colomap
}

func (p *Ping) getColo(b string) string {
	if b == "" {
		return ""
	}
	// Match and return the airport code using regular expressions
	out := OutRegexp.FindString(b)

	if HttpingCFColomap == nil {
		return out
	}
	// Match whether the airport code is in the specified region
	_, ok := HttpingCFColomap.Load(out)
	if ok {
		return out
	}

	return ""
}
