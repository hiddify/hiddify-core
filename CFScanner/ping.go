package CFScanner

import (
	"encoding/csv"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

// Whether to print the test results
func NoPrintResult() bool {
	return PrintNum == 0
}

// Whether to output to a file
func noOutput() bool {
	return Output == "" || Output == " "
}

type PingData struct {
	IP       *net.IPAddr
	Sended   int
	Received int
	Delay    time.Duration
}

type CloudflareIPData struct {
	*PingData
	lossRate      float32
	DownloadSpeed float64
}

// Calculate the loss rate
func (cf *CloudflareIPData) getLossRate() float32 {
	if cf.lossRate == 0 {
		pingLost := cf.Sended - cf.Received
		cf.lossRate = float32(pingLost) / float32(cf.Sended)
	}
	return cf.lossRate
}

func (cf *CloudflareIPData) toString() []string {
	result := make([]string, 6)
	result[0] = cf.IP.String()
	result[1] = strconv.Itoa(cf.Sended)
	result[2] = strconv.Itoa(cf.Received)
	result[3] = strconv.FormatFloat(float64(cf.getLossRate()), 'f', 2, 32)
	result[4] = strconv.FormatFloat(cf.Delay.Seconds()*1000, 'f', 2, 32)
	result[5] = strconv.FormatFloat(cf.DownloadSpeed/1024/1024, 'f', 2, 32)
	return result
}

func ExportCsv(data []CloudflareIPData) {
	if noOutput() || len(data) == 0 {
		return
	}
	fp, err := os.Create(Output)
	if err != nil {
		log.Fatalf("Failed to create file [%s]: %v", Output, err)
		return
	}
	defer fp.Close()
	w := csv.NewWriter(fp) // Create a new file writer
	_ = w.Write([]string{"IP Address", "Sent", "Received", "Loss Rate", "Average Delay", "Download Speed (MB/s)"})
	_ = w.WriteAll(convertToString(data))
	w.Flush()
}

func convertToString(data []CloudflareIPData) [][]string {
	result := make([][]string, 0)
	for _, v := range data {
		result = append(result, v.toString())
	}
	return result
}

// Delay and loss rate sorting
type PingDelaySet []CloudflareIPData

// Delay condition filtering
func (s PingDelaySet) FilterDelay() (data PingDelaySet) {
	if InputMaxDelay > maxDelay || InputMinDelay < minDelay { // When the input delay conditions are not within the default range, no filtering is performed
		return s
	}
	if InputMaxDelay == maxDelay && InputMinDelay == minDelay { // When the input delay conditions are the default values, no filtering is performed
		return s
	}
	for _, v := range s {
		if v.Delay > time.Duration(InputMaxDelay) { // Upper limit of average delay, if the delay is greater than the maximum value, the subsequent data does not meet the condition, and the loop is exited directly
			break
		}
		if v.Delay < InputMinDelay { // Lower limit of average delay, if the delay is less than the minimum value, it does not meet the condition and is skipped
			continue
		}
		data = append(data, v) // When the delay meets the condition, it is added to the new array
	}
	return
}

// Loss rate condition filtering
func (s PingDelaySet) FilterLossRate() (data PingDelaySet) {
	if InputMaxLossRate >= maxLossRate { // When the input loss condition is the default value, no filtering is performed
		return s
	}
	for _, v := range s {
		if v.getLossRate() > InputMaxLossRate { // Upper limit of loss rate
			break
		}
		data = append(data, v) // When the loss rate meets the condition, it is added to the new array
	}
	return
}

func (s PingDelaySet) Len() int {
	return len(s)
}
func (s PingDelaySet) Less(i, j int) bool {
	iRate, jRate := s[i].getLossRate(), s[j].getLossRate()
	if iRate != jRate {
		return iRate < jRate
	}
	return s[i].Delay < s[j].Delay
}
func (s PingDelaySet) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Download speed sorting
type DownloadSpeedSet []CloudflareIPData

func (s DownloadSpeedSet) Len() int {
	return len(s)
}
func (s DownloadSpeedSet) Less(i, j int) bool {
	return s[i].DownloadSpeed > s[j].DownloadSpeed
}
func (s DownloadSpeedSet) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s DownloadSpeedSet) Print() {
	if NoPrintResult() {
		return
	}
	if len(s) <= 0 { // When the length of the IP array (number of IPs) is 0, skip outputting the result
		fmt.Println("\n[Info] The number of complete speed test results is 0, skipping output of results.")
		return
	}
	dateString := convertToString(s) // Convert to a multidimensional array [][]string
	if len(dateString) < PrintNum {  // If the length of the IP array (number of IPs) is less than the print count, change the count to the number of IPs
		PrintNum = len(dateString)
	}
	headFormat := "%-16s%-5s%-5s%-5s%-6s%-11s\n"
	dataFormat := "%-18s%-8s%-8s%-8s%-10s%-15s\n"
	for i := 0; i < PrintNum; i++ { // If the IP to be output contains IPv6, the spacing needs to be adjusted
		if len(dateString[i][0]) > 15 {
			headFormat = "%-40s%-5s%-5s%-5s%-6s%-11s\n"
			dataFormat = "%-42s%-8s%-8s%-8s%-10s%-15s\n"
			break
		}
	}
	fmt.Printf(headFormat, "IP Address", "Sent", "Received", "Loss Rate", "Average Delay", "Download Speed (MB/s)")
	for i := 0; i < PrintNum; i++ {
		fmt.Printf(dataFormat, dateString[i][0], dateString[i][1], dateString[i][2], dateString[i][3], dateString[i][4], dateString[i][5])
	}
	if !noOutput() {
		fmt.Printf("\nComplete speed test results have been written to the %v file and can be viewed using Notepad/Spreadsheet software.\n", Output)
	}
}
