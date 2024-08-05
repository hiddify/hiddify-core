package CFScanner

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/VividCortex/ewma"
)

func checkDownloadDefault() {
	if URL == "" {
		URL = defaultURL
	}
	if Timeout <= 0 {
		Timeout = defaultTimeout
	}
	if TestCount <= 0 {
		TestCount = defaultTestNum
	}
	if MinSpeed <= 0.0 {
		MinSpeed = defaultMinSpeed
	}
}

func TestDownloadSpeed(ipSet PingDelaySet) (speedSet DownloadSpeedSet) {
	checkDownloadDefault()
	if Disable {
		return DownloadSpeedSet(ipSet)
	}
	if len(ipSet) <= 0 { // Continue downloading speed test only when the length of the IP array is greater than 0
		fmt.Println("\n[Info] The number of IP addresses for delay speed test is 0, skipping download speed test.")
		return
	}
	testNum := TestCount
	if len(ipSet) < TestCount || MinSpeed > 0 { // If the length of the IP array is less than the download speed test count (-dn), modify the count to the length of the IP array
		testNum = len(ipSet)
	}
	if testNum < TestCount {
		TestCount = testNum
	}

	fmt.Printf("Start download speed test (minimum: %.2f MB/s, count: %d, queue: %d)\n", MinSpeed, TestCount, testNum)
	// Make the length of the download speed progress bar consistent with the delay speed progress bar (for OCD)
	bar_a := len(strconv.Itoa(len(ipSet)))
	bar_b := "     "
	for i := 0; i < bar_a; i++ {
		bar_b += " "
	}
	bar := NewBar(TestCount, bar_b, "")
	for i := 0; i < testNum; i++ {
		speed := downloadHandler(ipSet[i].IP)
		ipSet[i].DownloadSpeed = speed
		// Filter the results based on the [minimum download speed] condition after each IP download speed test
		if speed >= MinSpeed*1024*1024 {
			bar.Grow(1, "")
			speedSet = append(speedSet, ipSet[i]) // Add to the new array when the speed is higher than the minimum download speed
			if len(speedSet) == TestCount {       // Break the loop when enough IPs that meet the condition (download speed test count -dn) are obtained
				break
			}
		}
	}
	bar.Done()
	if len(speedSet) == 0 { // If there is no data that meets the speed limit, return all test data
		speedSet = DownloadSpeedSet(ipSet)
	}
	// Sort by speed
	sort.Sort(speedSet)
	return
}

func getDialContext(ip *net.IPAddr) func(ctx context.Context, network, address string) (net.Conn, error) {
	var fakeSourceAddr string
	if isIPv4(ip.String()) {
		fakeSourceAddr = fmt.Sprintf("%s:%d", ip.String(), TCPPort)
	} else {
		fakeSourceAddr = fmt.Sprintf("[%s]:%d", ip.String(), TCPPort)
	}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, fakeSourceAddr)
	}
}

// return download Speed
func downloadHandler(ip *net.IPAddr) float64 {
	client := &http.Client{
		Transport: &http.Transport{DialContext: getDialContext(ip)},
		Timeout:   Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 10 { // Limit the maximum number of redirects to 10
				return http.ErrUseLastResponse
			}
			if req.Header.Get("Referer") == defaultURL { // When using the default download speed test URL, do not include Referer in the redirect
				req.Header.Del("Referer")
			}
			return nil
		},
	}
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return 0.0
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36")

	response, err := client.Do(req)
	if err != nil {
		return 0.0
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return 0.0
	}
	timeStart := time.Now()           // Start time (current time)
	timeEnd := timeStart.Add(Timeout) // End time calculated by adding the download speed test time

	contentLength := response.ContentLength // File size
	buffer := make([]byte, bufferSize)

	var (
		contentRead     int64 = 0
		timeSlice             = Timeout / 100
		timeCounter           = 1
		lastContentRead int64 = 0
	)

	var nextTime = timeStart.Add(timeSlice * time.Duration(timeCounter))
	e := ewma.NewMovingAverage()

	// Loop calculation, exit the loop (stop the speed test) if the file is downloaded (both are equal)
	for contentLength != contentRead {
		currentTime := time.Now()
		if currentTime.After(nextTime) {
			timeCounter++
			nextTime = timeStart.Add(timeSlice * time.Duration(timeCounter))
			e.Add(float64(contentRead - lastContentRead))
			lastContentRead = contentRead
		}
		// Exit the loop (stop the speed test) if it exceeds the download speed test time
		if currentTime.After(timeEnd) {
			break
		}
		bufferRead, err := response.Body.Read(buffer)
		if err != nil {
			if err != io.EOF { // If an error occurs during the file download process (such as Timeout) and it is not because the file is downloaded, exit the loop (stop the speed test)
				break
			} else if contentLength == -1 { // If the file is downloaded and the file size is unknown, exit the loop (stop the speed test). For example: https://speed.cloudflare.com/__down?bytes=200000000 If it is downloaded within 10 seconds, it will cause the speed test result to be significantly lower or even displayed as 0.00 (when the download speed is too fast)
				break
			}
			// Get the previous time slice
			last_time_slice := timeStart.Add(timeSlice * time.Duration(timeCounter-1))
			// Downloaded data amount / (current time - previous time slice / time slice)
			e.Add(float64(contentRead-lastContentRead) / (float64(currentTime.Sub(last_time_slice)) / float64(timeSlice)))
		}
		contentRead += int64(bufferRead)
	}
	return e.Value() / (Timeout.Seconds() / 120)
}
