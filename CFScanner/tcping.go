package CFScanner

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"
)

type Ping struct {
	wg      *sync.WaitGroup
	m       *sync.Mutex
	ips     []*net.IPAddr
	csv     PingDelaySet
	control chan bool
	bar     *Bar
}

func checkPingDefault() {
	if Routines <= 0 {
		Routines = defaultRoutines
	}
	if TCPPort <= 0 || TCPPort >= 65535 {
		TCPPort = defaultPort
	}
	if PingTimes <= 0 {
		PingTimes = defaultPingTimes
	}
}

func NewPing() *Ping {
	checkPingDefault()
	ips := loadIPRanges(TestIP, TestIPNum)
	return &Ping{
		wg:      &sync.WaitGroup{},
		m:       &sync.Mutex{},
		ips:     ips,
		csv:     make(PingDelaySet, 0),
		control: make(chan bool, Routines),
		bar:     NewBar(len(ips), "Available:", ""),
	}
}

func (p *Ping) Run() PingDelaySet {
	if len(p.ips) == 0 {
		return p.csv
	}
	if Httping {
		fmt.Printf("Starting latency test (Mode: HTTP, Port: %d, Range: %v ~ %v ms, Packet Loss: %.2f)\n", TCPPort, InputMinDelay.Milliseconds(), InputMaxDelay.Milliseconds(), InputMaxLossRate)
	} else {
		fmt.Printf("Starting latency test (Mode: TCP, Port: %d, Range: %v ~ %v ms, Packet Loss: %.2f)\n", TCPPort, InputMinDelay.Milliseconds(), InputMaxDelay.Milliseconds(), InputMaxLossRate)
	}
	for _, ip := range p.ips {
		p.wg.Add(1)
		p.control <- false
		go p.start(ip)
	}
	p.wg.Wait()
	p.bar.Done()
	sort.Sort(p.csv)
	return p.csv
}

func (p *Ping) start(ip *net.IPAddr) {
	defer p.wg.Done()
	p.tcpingHandler(ip)
	<-p.control
}

// bool connectionSucceed float32 time
func (p *Ping) tcping(ip *net.IPAddr) (bool, time.Duration) {
	startTime := time.Now()
	var fullAddress string
	if isIPv4(ip.String()) {
		fullAddress = fmt.Sprintf("%s:%d", ip.String(), TCPPort)
	} else {
		fullAddress = fmt.Sprintf("[%s]:%d", ip.String(), TCPPort)
	}
	conn, err := net.DialTimeout("tcp", fullAddress, tcpConnectTimeout)
	if err != nil {
		return false, 0
	}
	defer conn.Close()
	duration := time.Since(startTime)
	return true, duration
}

// pingReceived pingTotalTime
func (p *Ping) checkConnection(ip *net.IPAddr) (recv int, totalDelay time.Duration) {
	if Httping {
		recv, totalDelay = p.httping(ip)
		return
	}
	for i := 0; i < PingTimes; i++ {
		if ok, delay := p.tcping(ip); ok {
			recv++
			totalDelay += delay
		}
	}
	return
}

func (p *Ping) appendIPData(data *PingData) {
	p.m.Lock()
	defer p.m.Unlock()
	p.csv = append(p.csv, CloudflareIPData{
		PingData: data,
	})
}

// handle tcping
func (p *Ping) tcpingHandler(ip *net.IPAddr) {
	recv, totalDlay := p.checkConnection(ip)
	nowAble := len(p.csv)
	if recv != 0 {
		nowAble++
	}
	p.bar.Grow(1, strconv.Itoa(nowAble))
	if recv == 0 {
		return
	}
	data := &PingData{
		IP:       ip,
		Sended:   PingTimes,
		Received: recv,
		Delay:    totalDlay / time.Duration(recv),
	}
	p.appendIPData(data)
}
