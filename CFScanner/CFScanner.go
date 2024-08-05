package CFScanner

import (
	"regexp"
	"time"
)

// this is inspired by github.com/XIU2/CloudflareSpeedTest
// and modified to be used as a library used for scan the best cloudflare ip
// Thanks to the original author

// Constants related to default configurations
const (
	defaultOutput = "result.csv" // Default output file name
	// defaultInputFile       = "ip.txt"                  // Default input file name
	defaultURL             = "https://cf.xiu2.xyz/url" // Default URL for download testing
	defaultTimeout         = 10 * time.Second          // Default timeout for requests
	defaultDisableDownload = false                     // Default setting to not disable downloads
	defaultTestNum         = 10                        // Default number of tests to perform
	defaultMinSpeed        = 0.0                       // Default minimum speed in float64
)

// Constants related to delay and timeout configurations
const (
	maxDelay          = 9999 * time.Millisecond // Maximum delay allowed
	minDelay          = 0 * time.Millisecond    // Minimum delay allowed
	tcpConnectTimeout = time.Second * 1         // Timeout for TCP connections
)

// Constants related to loss rates and buffer sizes
const (
	maxLossRate      float32 = 1.0   // Maximum loss rate allowed
	bufferSize               = 1024  // Size of the buffer used
	maxRoutine               = 1000  // Maximum number of routines allowed
	defaultRoutines          = 200   // Default number of routines
	defaultPort              = 443   // Default TCP port
	defaultPingTimes         = 4     // Default number of pings
	defaultTestIP            = false // Default setting to not test IP
	defaultTestIPNum         = 200   // Default number of IPs to test
)

// Variables linked to default constants
var (
	InputMaxDelay     = time.Duration(9999) * time.Millisecond // Maximum delay allowed
	InputMinDelay     = time.Duration(0) * time.Millisecond    // Minimum delay allowed
	InputMaxLossRate  = float32(1.0)                           // Maximum loss rate allowed
	Timeout           = time.Duration(10) * time.Second        // Default timeout for requests
	HttpingCFColomap  = MapColoMap()                           // Map for HTTPing CF Colo data
	Routines          = 200                                    // Default number of routines
	PingTimes         = 4                                      // Default number of pings
	TestCount         = 10                                     // Default number of tests to perform
	TCPPort           = 443                                    // Default TCP port
	URL               = "https://cf.xiu2.xyz/url"              // Default URL for testing
	Httping           = false                                  // HTTPing status
	HttpingStatusCode = 0                                      // HTTPing status code
	HttpingCFColo     = ""
	MinSpeed          = defaultMinSpeed
	OutRegexp         = regexp.MustCompile(`[A-Z]{3}`)                                                                                                                                                                                                                         // HTTPing CF Colo data
	PrintNum          = 10                                                                                                                                                                                                                                                     // Number of prints to output
	IPFile            = "ip.txt"                                                                                                                                                                                                                                               // Default IP file name
	Output            = "result.csv"                                                                                                                                                                                                                                           // Default output file name
	Disable           = false                                                                                                                                                                                                                                                  // Setting to disable downloads
	TestAll           = false                                                                                                                                                                                                                                                  // Variable to control testing all IPs
	TestIP            = false                                                                                                                                                                                                                                                  // Variable to test IP
	TestIPNum         = 200                                                                                                                                                                                                                                                    // Variable for the number of IPs to test
	IPText            = "173.245.48.0/20, 103.21.244.0/22, 103.22.200.0/22, 103.31.4.0/22, 141.101.64.0/18, 108.162.192.0/18, 190.93.240.0/20, 188.114.96.0/20, 197.234.240.0/22, 198.41.128.0/17, 162.158.0.0/15, 104.16.0.0/13, 104.24.0.0/14, 172.64.0.0/13, 131.0.72.0/22" // Specific IP ranges
)

type CloudFlareOptions struct {
	EnableCloudFlare bool `json:"enable-cloudflare"`
	CloudFlareIPNum  int  `json:"cloudflare-ip-num"`
	CloudFlareIPs    []string
}

func Run(base CloudFlareOptions) CloudFlareOptions {
	InitRandSeed() // set random seed
	TestIP = base.EnableCloudFlare
	TestIPNum = base.CloudFlareIPNum
	pingData := NewPing().Run().FilterDelay().FilterLossRate()
	// jsonData, _ := json.Marshal(pingData)
	var CloudflareIPList []string
	if len(pingData) == 0 {
		return base
	}
	for _, v := range pingData[:10] {
		CloudflareIPList = append(CloudflareIPList, v.IP.String())
	}
	base.CloudFlareIPs = CloudflareIPList
	return base
}
